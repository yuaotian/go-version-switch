package version

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// EnvBackup 环境变量备份结构
type EnvBackup struct {
	Timestamp  string `json:"timestamp"`
	GOROOT     string `json:"goroot"`
	Path       string `json:"path"`
	BackupFile string `json:"backup_file"`
}

// SetAsCurrentGo 设置指定目录为当前Go环境（向后兼容）
func SetAsCurrentGo(goRoot string) error {
	return SetupGoEnvironment(goRoot)
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
		fmt.Printf("发现现有Go安装: %s\n", existingGoRoot)
		// 备份当前环境变量
		if err := backupEnvironment(); err != nil {
			fmt.Printf("警告: 备份环境变量失败: %v\n", err)
		}
	}

	// 设置GOROOT
	if err := manageGoRoot(newGoRoot); err != nil {
		return fmt.Errorf("设置GOROOT失败: %v", err)
	}

	// 更新PATH
	if err := manageGoPath(); err != nil {
		return fmt.Errorf("更新PATH失败: %v", err)
	}

	// 通知用户
	notifyUser()
	return nil
}

// isValidGoRoot 检查是否为有效的Go安装目录（向后兼容）
func isValidGoRoot(dir string) bool {
	return validateGoRootPath(dir) == nil
}

// backupEnvironment 备份环境变量
func backupEnvironment() error {
	// 创建备份目录
	backupDir := filepath.Join(filepath.Dir(os.Args[0]), "backup")
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

	// 创建备份对象
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(backupDir, fmt.Sprintf("env_backup_%s.json", timestamp))
	backup := EnvBackup{
		Timestamp:  timestamp,
		GOROOT:     goroot,
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

	fmt.Println("✅ GOROOT环境变量已更新")
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

	// 检查是否已包含%GOROOT%\bin
	if strings.Contains(currentPath, "%GOROOT%\\bin") {
		fmt.Println("✅ PATH已包含Go路径")
		return nil
	}

	// 添加%GOROOT%\bin到PATH
	newPath := currentPath + string(os.PathListSeparator) + "%GOROOT%\\bin"

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

	// 广播环境变量更改消息
	broadcastEnvChange()

	fmt.Println("✅ PATH环境变量已更新")
	return nil
}

// notifyUser 通知用户重启相关程序
func notifyUser() {
	fmt.Println("\n✨ Go环境变量设置完成！")
	fmt.Println("\n⚠️ 请重启以下程序以使环境变量生效：")
	fmt.Println("1. Visual Studio Code")
	fmt.Println("2. IntelliJ IDEA")
	fmt.Println("3. 终端 (Terminal)")
	fmt.Println("4. PowerShell")
	fmt.Println("\n重启后，请在终端中运行 'go version' 验证安装是否成功")
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
