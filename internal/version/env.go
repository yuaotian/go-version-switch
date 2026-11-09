package version

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "runtime"
    "sort"
    "strings"
    "time"
)

// EnvBackup 环境变量备份结构
type EnvBackup struct {
    Timestamp  string `json:"timestamp"`
    GOROOT     string `json:"goroot"`
    GOARCH     string `json:"goarch"`
    Path       string `json:"path"`
    BackupFile string `json:"backup_file"`
}

const (
    // 用户级环境变量：不需要管理员权限，且优先级高于机器级同名变量
    regEnvUser = `HKEY_CURRENT_USER\Environment`
    // 机器级环境变量：只读，用来发现旧版本遗留的全局 GOROOT/PATH
    regEnvMachine = `HKEY_LOCAL_MACHINE\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
)

// readRegValue 读取注册表字符串值，不存在时返回 error
func readRegValue(key, name string) (string, error) {
    out, err := exec.Command("REG", "QUERY", key, "/v", name).CombinedOutput()
    if err != nil {
        return "", err
    }
    re := regexp.MustCompile(`REG_(?:EXPAND_)?SZ\s+(.+)`)
    m := re.FindStringSubmatch(string(out))
    if len(m) < 2 {
        return "", fmt.Errorf("解析注册表值失败: %s", name)
    }
    return strings.TrimRight(m[1], "\r\n"), nil
}

// writeRegValue 写入注册表字符串值
func writeRegValue(key, name, kind, value string) error {
    out, err := exec.Command("REG", "ADD", key, "/v", name, "/t", kind, "/d", value, "/f").CombinedOutput()
    if err != nil {
        return fmt.Errorf("写入 %s 失败: %v\n%s", name, err, out)
    }
    return nil
}

// deleteRegValue 删除注册表字符串值
func deleteRegValue(key, name string) error {
    out, err := exec.Command("REG", "DELETE", key, "/v", name, "/f").CombinedOutput()
    if err != nil {
        return fmt.Errorf("删除 %s 失败: %v\n%s", name, err, out)
    }
    return nil
}

// SetAsCurrentGo 设置指定目录为当前Go环境（向后兼容）
func SetAsCurrentGo(goRoot string) error {
    // 先备份当前环境
    fmt.Println("📦 正在备份当前环境变量...")
    if err := backupEnvironment(); err != nil {
        return fmt.Errorf("备份环境变量失败: %v", err)
    }
    fmt.Println("✅ 环境变量备份完成")

    // 尝试设置新环境
    if err := SetupGoEnvironment(goRoot); err != nil {
        fmt.Println("❌ 设置新环境失败，准备回滚...")

        // 如果设置失败，尝试回滚
        backupDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "backup_env")
        fmt.Printf("🔍 正在查找最新的备份文件 (目录: %s)...\n", backupDir)

        latestBackup, rollbackErr := GetLatestBackup(backupDir)
        if rollbackErr != nil {
            fmt.Println("❌ 无法找到有效的备份文件")
            return fmt.Errorf("设置环境变量失败且无法回滚: %v (回滚错误: %v)", err, rollbackErr)
        }

        fmt.Printf("📂 找到最新备份文件: %s\n", latestBackup)
        fmt.Println("🔄 开始执行环境变量回滚...")

        if rollbackErr := RestoreEnvironment(latestBackup); rollbackErr != nil {
            fmt.Println("❌ 回滚操作失败")
            return fmt.Errorf("设置环境变量失败且回滚失败: %v (回滚错误: %v)", err, rollbackErr)
        }

        fmt.Println("✅ 环境变量已成功回滚到之前的配置")
        return fmt.Errorf("设置环境变量失败，已回滚到之前的配置: %v", err)
    }

    fmt.Println("✅ 新环境设置成功")
    return nil
}

// detectArchFromGoRoot 从安装目录名推断 GOARCH
func detectArchFromGoRoot(goRoot string) string {
    lower := strings.ToLower(goRoot)
    switch {
    case strings.Contains(lower, "-arm64"):
        return "arm64"
    case strings.Contains(lower, "-arm"):
        return "arm"
    case strings.Contains(lower, "-x86-64"):
        return "amd64"
    case strings.Contains(lower, "-x86"):
        return "386"
    default:
        return "amd64"
    }
}

// SetGoRootOnly 只把 GOROOT / GOARCH 指到指定目录，不修改 PATH。
// 用于 -no-path：PATH 交给 mise / shims 等工具托管，govs 只负责提供安装目录。
func SetGoRootOnly(goRoot string) error {
    if err := backupEnvironment(); err != nil {
        return fmt.Errorf("备份环境变量失败: %v", err)
    }

    if err := manageGoRoot(goRoot); err != nil {
        return err
    }

    arch := detectArchFromGoRoot(goRoot)
    if err := writeRegValue(regEnvUser, "GOARCH", "REG_SZ", arch); err != nil {
        return fmt.Errorf("设置GOARCH失败: %v", err)
    }
    _ = os.Setenv("GOARCH", arch)

    broadcastEnvChange()

    fmt.Printf("ℹ️ 已跳过 PATH 修改（-no-path），GOARCH=%s；用哪个 go 由 mise / 你的 PATH 方案决定\n", arch)
    return nil
}

// SetupGoEnvironment 设置Go环境变量（用户级，无需管理员权限）
func SetupGoEnvironment(newGoRoot string) error {
    // 只改用户级环境变量（HKCU）。如果当前有管理员权限，顺手清掉旧版本写进机器级的
    // GOROOT / go bin 条目 —— Windows 的 PATH 是「机器级在前、用户级在后」，
    // 机器级残留不清掉，用户级切换永远会被盖住。
    if isAdmin, adminErr := checkAdminPrivileges(); adminErr == nil && isAdmin {
        cleanupMachineLevelEnv()
    } else {
        fmt.Println("ℹ️ 非管理员运行：只更新用户级环境变量（推荐做法）")
    }

    // 检测现有Go安装（仅用于提示）
    if existingGoRoot, err := detectExistingGo(); err == nil && existingGoRoot != "" {
        fmt.Printf("🔍 发现现有Go安装: %s\n", existingGoRoot)
    }

    // 根据目录名判断架构
    arch := detectArchFromGoRoot(newGoRoot)

    // 设置GOARCH（用户级）
    if err := writeRegValue(regEnvUser, "GOARCH", "REG_SZ", arch); err != nil {
        return fmt.Errorf("设置GOARCH失败: %v", err)
    }

    // 更新当前进程的GOARCH
    if err := os.Setenv("GOARCH", arch); err != nil {
        return fmt.Errorf("更新当前进程GOARCH失败: %v", err)
    }

    fmt.Printf("✅ GOARCH 已更新为: %s（用户级）\n", arch)

    // 设置GOROOT
    if err := manageGoRoot(newGoRoot); err != nil {
        return fmt.Errorf("设置GOROOT失败: %v", err)
    }

    // 更新用户级 PATH
    if err := manageGoPath(newGoRoot); err != nil {
        return fmt.Errorf("更新PATH失败: %v", err)
    }

    // 通知已经打开的桌面程序刷新环境变量
    broadcastEnvChange()

    return nil
}

// isValidGoRoot 检查是否为有效的Go安装目录（向后兼容）
func isValidGoRoot(dir string) bool {
    return validateGoRootPath(dir) == nil
}

// backupEnvironment 备份环境变量（用户级为准，用户级没有 PATH 时退回机器级）
func backupEnvironment() error {
    // 创建备份目录
    backupDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "backup_env")
    if err := os.MkdirAll(backupDir, 0755); err != nil {
        return fmt.Errorf("创建备份目录失败: %v", err)
    }

    // 获取当前GOROOT（用户级优先）
    goroot, err := readRegValue(regEnvUser, "GOROOT")
    if err != nil || goroot == "" {
        goroot = os.Getenv("GOROOT")
    }

    // 获取当前PATH：用户级没有就备份机器级的（回滚时才知道原来什么样）
    path, err := readRegValue(regEnvUser, "Path")
    if err != nil || path == "" {
        if machinePath, machineErr := readRegValue(regEnvMachine, "Path"); machineErr == nil {
            path = machinePath
        }
    }

    // 获取当前 GOARCH
    currentArch, err := readRegValue(regEnvUser, "GOARCH")
    if err != nil || currentArch == "" {
        currentArch, _ = readRegValue(regEnvMachine, "GOARCH")
    }
    if currentArch == "" {
        currentArch = runtime.GOARCH // 如果获取失败，使用当前系统架构作为默认值
    }

    // 创建备份对象
    timestamp := time.Now().Format("20060102_150405")
    backupFile := filepath.Join(backupDir, fmt.Sprintf("env_backup_%s.json", timestamp))
    backup := EnvBackup{
        Timestamp:  timestamp,
        GOROOT:     goroot,
        GOARCH:     currentArch,
        Path:       path,
        BackupFile: backupFile,
    }

    // 保存备份
    data, err := json.MarshalIndent(backup, "", "    ")
    if err != nil {
        return fmt.Errorf("序列化备份数据失败: %v", err)
    }

    if err := os.WriteFile(backupFile, data, 0644); err != nil {
        return fmt.Errorf("写入备份文件失败: %v", err)
    }

    fmt.Printf("✅ 环境变量已备份到: %s\n", backupFile)
    return nil
}

// detectExistingGo 检测现有的Go安装
func detectExistingGo() (string, error) {
    // 尝试使用go env命令获取GOROOT
    cmd := exec.Command("go", "env", "GOROOT")
    output, err := cmd.CombinedOutput()
    if err == nil {
        goRoot := strings.TrimSpace(string(output))
        if goRoot != "" {
            return goRoot, nil
        }
    }

    // 如果go命令失败，尝试从环境变量获取
    return os.Getenv("GOROOT"), nil
}

// manageGoRoot 管理GOROOT环境变量（写用户级，不需要管理员权限）
func manageGoRoot(goRoot string) error {
    // 验证路径
    if err := validateGoRootPath(goRoot); err != nil {
        return fmt.Errorf("Go路径验证失败: %v", err)
    }

    // 设置用户级 GOROOT（用户级优先于机器级）
    if err := writeRegValue(regEnvUser, "GOROOT", "REG_SZ", goRoot); err != nil {
        return fmt.Errorf("设置GOROOT失败: %v", err)
    }

    // 更新当前进程的环境变量
    if err := os.Setenv("GOROOT", goRoot); err != nil {
        return fmt.Errorf("更新当前进程GOROOT失败: %v", err)
    }

    fmt.Printf("✅ GOROOT环境变量已更新（用户级）: %s\n", goRoot)
    return nil
}

// manageGoPath 管理用户级PATH：清掉所有 Go bin 条目，再把 %GOROOT%\bin 放到最前
func manageGoPath(goRoot string) error {
    // 用户级 PATH 可能还不存在，此时从空开始（机器级 PATH 仍然生效且在前，见 cleanupMachineLevelEnv）
    currentPath, err := readRegValue(regEnvUser, "Path")
    if err != nil {
        currentPath = ""
    }

    // 移除所有指向 Go 安装 bin 的条目：
    //   • %GOROOT%\bin（govs 自己写的）
    //   • 旧版本 govs 写死的 ...\go-version\go-x.y.z-arch\bin / C:\Program Files\Go\bin
    pathParts := strings.Split(currentPath, string(os.PathListSeparator))
    newParts := make([]string, 0, len(pathParts))
    for _, part := range pathParts {
        if strings.TrimSpace(part) == "" {
            continue
        }
        if isGoBinPath(part) {
            continue
        }
        newParts = append(newParts, part)
    }

    // 将 %GOROOT%\bin 添加到用户级 PATH 最前面
    goBin := filepath.Join(goRoot, "bin")
    newPath := goBin + string(os.PathListSeparator) + strings.Join(newParts, string(os.PathListSeparator))

    if err := writeRegValue(regEnvUser, "Path", "REG_EXPAND_SZ", newPath); err != nil {
        return fmt.Errorf("更新用户级PATH失败: %v", err)
    }

    // 当前进程用展开后的真实路径（注册表里存的是字面路径，这里没必要再塞 %GOROOT%）
    if err := os.Setenv("PATH", newPath+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
        return fmt.Errorf("更新当前进程PATH失败: %v", err)
    }

    fmt.Println("✅ 用户级 PATH 已更新（Go 目录已移至最前）")
    return nil
}

// cleanupMachineLevelEnv 清理旧版本写进机器级的 Go 环境变量。
// Windows 的 PATH 是「机器级在前、用户级在后」，机器级里只要还留着 go 的 bin 目录，
// 用户级切换就永远不生效，所以这里必须清掉；没有管理员权限时只提示。
func cleanupMachineLevelEnv() {
    if machineGoRoot, err := readRegValue(regEnvMachine, "GOROOT"); err == nil && machineGoRoot != "" {
        if strings.Contains(strings.ToLower(machineGoRoot), "go-version") {
            if delErr := deleteRegValue(regEnvMachine, "GOROOT"); delErr == nil {
                fmt.Printf("🧹 已删除机器级 GOROOT（旧版遗留）: %s\n", machineGoRoot)
            } else {
                fmt.Printf("⚠️ 机器级 GOROOT 清理失败（需管理员）: %v\n", delErr)
            }
        }
    }

    machinePath, err := readRegValue(regEnvMachine, "Path")
    if err != nil || machinePath == "" {
        return
    }
    parts := strings.Split(machinePath, string(os.PathListSeparator))
    kept := make([]string, 0, len(parts))
    removed := make([]string, 0, 2)
    for _, p := range parts {
        if isGoBinPath(p) {
            removed = append(removed, p)
            continue
        }
        kept = append(kept, p)
    }
    if len(removed) == 0 {
        return
    }
    if err := writeRegValue(regEnvMachine, "Path", "REG_EXPAND_SZ", strings.Join(kept, string(os.PathListSeparator))); err != nil {
        fmt.Printf("⚠️ 机器级 PATH 清理失败（需管理员）: %v\n", err)
        return
    }
    for _, r := range removed {
        fmt.Printf("🧹 已从机器级 PATH 移除: %s\n", r)
    }
}

// isGoBinPath 判断某个 PATH 条目是否指向 Go 安装目录的 bin。
// 切换版本时要把这类条目全部清掉，再由 %GOROOT%\bin 统一接管。
func isGoBinPath(part string) bool {
    p := strings.TrimSpace(strings.ToLower(part))
    if p == "" {
        return false
    }
    if strings.Contains(p, "goroot") {
        return true
    }
    if !strings.HasSuffix(p, `\bin`) {
        return false
    }
    return strings.Contains(p, `\go-version\`) ||
        strings.HasSuffix(p, `\go\bin`) ||
        strings.Contains(p, `\golang\`)
}

// validateGoRootPath 验证Go根目录路径
func validateGoRootPath(goRoot string) error {
    // 检查路径是否存在
    if _, err := os.Stat(goRoot); os.IsNotExist(err) {
        return fmt.Errorf("目录不存在: %s", goRoot)
    }

    // 检查是否为有效的Go安装目录
    requiredPaths := []string{
        filepath.Join(goRoot, "bin", "go"+executableExtension()),
        filepath.Join(goRoot, "pkg"),
        filepath.Join(goRoot, "src"),
    }

    for _, path := range requiredPaths {
        if _, err := os.Stat(path); os.IsNotExist(err) {
            return fmt.Errorf("无效的Go安装目录，缺少必要文件: %s", path)
        }
    }

    return nil
}

// executableExtension 返回可执行文件扩展名
func executableExtension() string {
    if runtime.GOOS == "windows" {
        return ".exe"
    }
    return ""
}

// checkAdminPrivileges 检查是否具有管理员权限
func checkAdminPrivileges() (bool, error) {
    if runtime.GOOS != "windows" {
        return false, fmt.Errorf("暂不支持在 %s 系统上运行", runtime.GOOS)
    }

    cmd := exec.Command("net", "session")
    err := cmd.Run()
    return err == nil, nil
}

// broadcastEnvChange 广播环境变量更改消息
func broadcastEnvChange() {
    cmd := exec.Command("powershell", "-Command", `
$source = @'
using System;
using System.Runtime.InteropServices;
public class Win32 {
    [DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
    public static extern IntPtr SendMessageTimeout(
        IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
        uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
}
'@
Add-Type -TypeDefinition $source -Language CSharp
$HWND_BROADCAST = [IntPtr]0xffff
$WM_SETTINGCHANGE = 0x1a
$result = [UIntPtr]::Zero
[Win32]::SendMessageTimeout($HWND_BROADCAST, $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment", 2, 5000, [ref]$result)
	`)
    _ = cmd.Run()
}

// RestoreEnvironment 恢复环境变量
func RestoreEnvironment(backupFile string) error {
    // 读取备份文件
    data, err := os.ReadFile(backupFile)
    if err != nil {
        return fmt.Errorf("读取备份文件失败: %v", err)
    }

    var backup EnvBackup
    if err := json.Unmarshal(data, &backup); err != nil {
        return fmt.Errorf("解析备份数据失败: %v", err)
    }

    // 恢复 GOROOT（用户级）
    if backup.GOROOT != "" {
        if err := writeRegValue(regEnvUser, "GOROOT", "REG_SZ", backup.GOROOT); err != nil {
            return fmt.Errorf("恢复 GOROOT 失败: %v", err)
        }
        os.Setenv("GOROOT", backup.GOROOT)
    }

    // 恢复 GOARCH（用户级）
    if backup.GOARCH != "" {
        if err := writeRegValue(regEnvUser, "GOARCH", "REG_SZ", backup.GOARCH); err != nil {
            return fmt.Errorf("恢复 GOARCH 失败: %v", err)
        }
        os.Setenv("GOARCH", backup.GOARCH)
    }

    // 恢复 PATH（用户级）
    if backup.Path != "" {
        if err := writeRegValue(regEnvUser, "Path", "REG_EXPAND_SZ", backup.Path); err != nil {
            return fmt.Errorf("恢复 PATH 失败: %v", err)
        }
        os.Setenv("PATH", backup.Path)
    }

    // 广播环境变量更改
    broadcastEnvChange()

    // 显示恢复信息
    fmt.Printf("\n✅ 环境变量已从备份文件恢复: %s\n", backupFile)
    fmt.Printf("已恢复的配置:\n")
    fmt.Printf("- GOROOT: %s\n", backup.GOROOT)
    fmt.Printf("- GOARCH: %s\n", backup.GOARCH)
    fmt.Printf("- PATH: 已更新\n")

    // 提醒用户重启程序
    fmt.Println("\n⚠️ 请重启以下程序以使环境变量生效：")
    fmt.Println("1. Visual Studio Code")
    fmt.Println("2. IntelliJ IDEA")
    fmt.Println("3. 终端 (Terminal)")
    fmt.Println("4. PowerShell")
    fmt.Println("\n重启后，请在终端中运行 'go version' 验证配置是否正确")

    return nil
}

// CheckAdminPrivileges 导出 checkAdminPrivileges 函数
func CheckAdminPrivileges() (bool, error) {
    return checkAdminPrivileges()
}

// GetLatestBackup 获取最新的备份文件
func GetLatestBackup(backupDir string) (string, error) {
    // 确保备份目录存在
    if _, err := os.Stat(backupDir); os.IsNotExist(err) {
        return "", fmt.Errorf("备份目录不存在: %s", backupDir)
    }

    // 读取备份目录中的所有文件
    files, err := os.ReadDir(backupDir)
    if err != nil {
        return "", fmt.Errorf("读取备份目录失败: %v", err)
    }

    // 筛选并排序备份文件
    var backupFiles []string
    for _, file := range files {
        if !file.IsDir() && strings.HasPrefix(file.Name(), "env_backup_") && strings.HasSuffix(file.Name(), ".json") {
            backupFiles = append(backupFiles, filepath.Join(backupDir, file.Name()))
        }
    }

    if len(backupFiles) == 0 {
        return "", fmt.Errorf("未找到有效的备份文件")
    }

    // 按文件名排序（因为文件名包含时间戳，所以最后一个就是最新的）
    sort.Strings(backupFiles)
    latestBackup := backupFiles[len(backupFiles)-1]

    // 验证备份文件
    if err := validateBackupFile(latestBackup); err != nil {
        return "", fmt.Errorf("备份文件验证失败: %v", err)
    }

    return latestBackup, nil
}

// validateBackupFile 验证备份文件的完整性
func validateBackupFile(backupFile string) error {
    data, err := os.ReadFile(backupFile)
    if err != nil {
        return fmt.Errorf("读取备份文件失败: %v", err)
    }

    var backup EnvBackup
    if err := json.Unmarshal(data, &backup); err != nil {
        return fmt.Errorf("解析备份文件失败: %v", err)
    }

    // 验证必要字段
    if backup.Timestamp == "" {
        return fmt.Errorf("备份文件缺少时间戳")
    }
    if backup.GOROOT == "" {
        return fmt.Errorf("备份文件缺少 GOROOT")
    }
    if backup.GOARCH == "" {
        return fmt.Errorf("备份文件缺少 GOARCH")
    }
    if backup.Path == "" {
        return fmt.Errorf("备份文件缺少 PATH")
    }

    return nil
}
