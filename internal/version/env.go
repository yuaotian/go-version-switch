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

// SetupGoEnvironment 设置Go环境变量
func SetupGoEnvironment(newGoRoot string) error {
    // 检查管理员权限
    isAdmin, err := checkAdminPrivileges()
    if err != nil {
        return fmt.Errorf("检查管理员权限失败: %v", err)
    }

    if !isAdmin {
        return fmt.Errorf("需要管理员权限才能修改系统环境变量")
    }

    // 检测现有Go安装
    existingGoRoot, err := detectExistingGo()
    if err != nil {
        fmt.Printf("警告: 检测现有Go安装时出错: %v\n", err)
    } else if existingGoRoot != "" {
        fmt.Printf("🔍 发现现有Go安装: %s\n", existingGoRoot)
        // 备份当前环境变量
        if err := backupEnvironment(); err != nil {
            fmt.Printf("警告: 备份环境变量失败: %v\n", err)
        }
    }

    // 根据目录名判断架构
    arch := "amd64" // 默认值
    if strings.Contains(strings.ToLower(newGoRoot), "-x86-64") {
        arch = "amd64"
    } else if strings.Contains(strings.ToLower(newGoRoot), "-x86") {
        arch = "386"
    } else if strings.Contains(strings.ToLower(newGoRoot), "-arm64") {
        arch = "arm64"
    } else if strings.Contains(strings.ToLower(newGoRoot), "-arm") {
        arch = "arm"
    }

    // 设置GOARCH
    cmd := exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
        "/v", "GOARCH", "/t", "REG_SZ", "/d", arch, "/f")
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("设置GOARCH失败: %v\n%s", err, output)
    }

    // 更新当前进程的GOARCH
    if err := os.Setenv("GOARCH", arch); err != nil {
        return fmt.Errorf("更新当前进程GOARCH失败: %v", err)
    }

    fmt.Printf("✅ GOARCH环境变量已更新为: %s\n", arch)

    // 设置GOROOT
    if err := manageGoRoot(newGoRoot); err != nil {
        return fmt.Errorf("设置GOROOT失败: %v", err)
    }

    // 更新PATH
    if err := manageGoPath(); err != nil {
        return fmt.Errorf("更新PATH失败: %v", err)
    }

    return nil
}

// isValidGoRoot 检查是否为有效的Go安装目录（向后兼容）
func isValidGoRoot(dir string) bool {
    return validateGoRootPath(dir) == nil
}

// backupEnvironment 备份环境变量
func backupEnvironment() error {
    // 创建备份目录
    backupDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "backup_env")
    if err := os.MkdirAll(backupDir, 0755); err != nil {
        return fmt.Errorf("创建备份目录失败: %v", err)
    }

    // 获取当前GOROOT
    goroot := os.Getenv("GOROOT")

    // 获取当前PATH
    cmd := exec.Command("REG", "QUERY", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "Path")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("获取PATH环境变量失败: %v", err)
    }

    re := regexp.MustCompile(`REG_(?:EXPAND_)?SZ\s+(.+)`)
    matches := re.FindStringSubmatch(string(output))
    if len(matches) < 2 {
        return fmt.Errorf("解析PATH环境变量失败")
    }
    path := strings.TrimSpace(matches[1])

    // 获取当前 GOARCH
    cmd = exec.Command("REG", "QUERY", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "GOARCH")
    output, err = cmd.CombinedOutput()
    var currentArch string
    if err == nil {
        re := regexp.MustCompile(`REG_(?:EXPAND_)?SZ\s+(.+)`)
        if matches := re.FindStringSubmatch(string(output)); len(matches) >= 2 {
            currentArch = strings.TrimSpace(matches[1])
        }
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

// manageGoRoot 管理GOROOT环境变量
func manageGoRoot(goRoot string) error {
    // 验证路径
    if err := validateGoRootPath(goRoot); err != nil {
        return fmt.Errorf("Go路径验证失败: %v", err)
    }

    // 设置GOROOT环境变量
    cmd := exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
        "/v", "GOROOT", "/t", "REG_SZ", "/d", goRoot, "/f")
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("设置GOROOT失败: %v\n%s", err, output)
    }

    // 更新当前进程的环境变量
    if err := os.Setenv("GOROOT", goRoot); err != nil {
        return fmt.Errorf("更新当前进程GOROOT失败: %v", err)
    }

    fmt.Printf("✅ GOROOT环境变量已更新: %s\n", goRoot)
    return nil
}

// manageGoPath 管理PATH环境变量
func manageGoPath() error {
    // 获取系统PATH
    cmd := exec.Command("REG", "QUERY", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
        "/v", "Path")
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("获取系统PATH失败: %v", err)
    }

    // 解析PATH值
    re := regexp.MustCompile(`REG_(?:EXPAND_)?SZ\s+(.+)`)
    matches := re.FindStringSubmatch(string(output))
    if len(matches) < 2 {
        return fmt.Errorf("解析系统PATH失败")
    }

    currentPath := strings.TrimSpace(matches[1])

    // 移除现有的 %GOROOT%\bin（如果存在）
    pathParts := strings.Split(currentPath, string(os.PathListSeparator))
    newParts := make([]string, 0)
    for _, part := range pathParts {
        if !strings.Contains(strings.ToLower(part), "goroot") {
            newParts = append(newParts, part)
        }
    }

    // 将 %GOROOT%\bin 添加到 PATH 的最前面
    newPath := "%GOROOT%\\bin" + string(os.PathListSeparator) + strings.Join(newParts, string(os.PathListSeparator))

    // 更新系统PATH
    cmd = exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
        "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", newPath, "/f")
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("更新系统PATH失败: %v\n%s", err, output)
    }

    // 更新当前进程的PATH
    if err := os.Setenv("PATH", newPath); err != nil {
        return fmt.Errorf("更新当前进程PATH失败: %v", err)
    }

    fmt.Println("✅ PATH环境变量已更新（Go目录已移至系统PATH最前）")
    return nil
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

    // 恢复 GOROOT
    if backup.GOROOT != "" {
        cmd := exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
            "/v", "GOROOT", "/t", "REG_SZ", "/d", backup.GOROOT, "/f")
        if output, err := cmd.CombinedOutput(); err != nil {
            return fmt.Errorf("恢复 GOROOT 失败: %v\n%s", err, output)
        }
        os.Setenv("GOROOT", backup.GOROOT)
    }

    // 恢复 GOARCH
    if backup.GOARCH != "" {
        cmd := exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
            "/v", "GOARCH", "/t", "REG_SZ", "/d", backup.GOARCH, "/f")
        if output, err := cmd.CombinedOutput(); err != nil {
            return fmt.Errorf("恢复 GOARCH 失败: %v\n%s", err, output)
        }
        os.Setenv("GOARCH", backup.GOARCH)
    }

    // 恢复 PATH
    if backup.Path != "" {
        cmd := exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment",
            "/v", "Path", "/t", "REG_EXPAND_SZ", "/d", backup.Path, "/f")
        if output, err := cmd.CombinedOutput(); err != nil {
            return fmt.Errorf("恢复 PATH 失败: %v\n%s", err, output)
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
