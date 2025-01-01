package version

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

var envMutex sync.Mutex

// backupEnvVars 备份环境变量
func backupEnvVars() error {
	// 获取当前时间作为备份文件名的一部分
	timestamp := time.Now().Format("20060102_150405")
	backupDir := filepath.Join(filepath.Dir(os.Args[0]), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}

	backupFile := filepath.Join(backupDir, fmt.Sprintf("env_backup_%s.reg", timestamp))

	// 导出环境变量到注册表文件
	cmd := exec.Command("REG", "EXPORT", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", backupFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("备份环境变量失败: %v\n%s", err, output)
	}

	fmt.Printf("✅ 环境变量已备份到: %s\n", backupFile)
	return nil
}

// SetAsCurrentGo 设置指定目录为当前Go环境
func SetAsCurrentGo(goRoot string) error {
	envMutex.Lock()
	defer envMutex.Unlock()

	fmt.Println("⚠️ 正在修改系统环境变量，请确保以管理员权限运行...")

	// 备份环境变量
	if err := backupEnvVars(); err != nil {
		fmt.Printf("警告: 备份环境变量失败: %v\n", err)
		fmt.Println("建议在继续之前手动备份系统环境变量")
		fmt.Print("是否继续? [y/N] ")
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(answer) != "y" {
			return fmt.Errorf("操作已取消")
		}
	}

	// 规范化路径
	goRoot = filepath.Clean(goRoot)

	// 检查目录是否存在
	if _, err := os.Stat(goRoot); os.IsNotExist(err) {
		return fmt.Errorf("Go目录不存在: %s", goRoot)
	}

	// 检查是否为有效的Go安装目录
	if !isValidGoRoot(goRoot) {
		return fmt.Errorf("无效的Go安装目录: %s", goRoot)
	}

	// 保存原始环境变量，以便回滚
	originalGoRoot := os.Getenv("GOROOT")
	originalPath := os.Getenv("PATH")

	fmt.Printf("📝 当前GOROOT: %s\n", originalGoRoot)
	fmt.Printf("📝 新GOROOT: %s\n", goRoot)

	// 设置GOROOT环境变量
	if err := setEnvVar("GOROOT", goRoot); err != nil {
		return fmt.Errorf("设置GOROOT失败: %v", err)
	}
	fmt.Println("✅ GOROOT环境变量已更新")

	// 更新PATH环境变量
	binDir := filepath.Join(goRoot, "bin")
	if err := addToPath(binDir); err != nil {
		// 回滚GOROOT
		_ = setEnvVar("GOROOT", originalGoRoot)
		return fmt.Errorf("更新PATH失败: %v", err)
	}
	fmt.Println("✅ PATH环境变量已更新")

	// 验证安装
	if err := verifyGoInstallation(); err != nil {
		// 回滚所有更改
		_ = setEnvVar("GOROOT", originalGoRoot)
		_ = setEnvVar("PATH", originalPath)
		return fmt.Errorf("验证安装失败: %v", err)
	}

	fmt.Println("\n✨ Go环境已成功切换！")
	fmt.Println("⚠️ 注意事项：")
	fmt.Println("1. 某些程序可能需要重启才能识别新的环境变量")
	fmt.Println("2. 建议重启以下程序：")
	fmt.Println("   • 终端 (PowerShell, CMD 等)")
	fmt.Println("   • 编辑器 (VSCode, IntelliJ IDEA 等)")
	fmt.Println("   • 其他使用Go环境的应用")
	fmt.Printf("3. 环境变量备份文件位于: %s\\backup\n", filepath.Dir(os.Args[0]))

	return nil
}

// isValidGoRoot 检查是否为有效的Go安装目录
func isValidGoRoot(dir string) bool {
	// 检查必要的文件和目录是否存在
	requiredPaths := []string{
		filepath.Join(dir, "bin", "go"+executableExtension()),
		filepath.Join(dir, "pkg"),
		filepath.Join(dir, "src"),
	}

	for _, path := range requiredPaths {
		cleanPath := filepath.Clean(path)
		if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
			return false
		}
	}

	return true
}

// executableExtension 根据操作系统返回可执行文件扩展名
func executableExtension() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// setEnvVar 设置环境变量
func setEnvVar(name, value string) error {
	// 更新当前进程的环境变量
	if err := os.Setenv(name, value); err != nil {
		return fmt.Errorf("设置当前进程环境变量失败: %v", err)
	}

	if runtime.GOOS == "windows" {
		// 在Windows上使用REG ADD命令设置系统环境变量
		cmd := exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", name, "/t", "REG_SZ", "/d", value, "/f")
		if output, err := cmd.CombinedOutput(); err != nil {
			// 回滚当前进程的环境变量
			_ = os.Setenv(name, os.Getenv(name))
			return fmt.Errorf("设置系统环境变量失败: %v\n%s", err, output)
		}

		// 广播环境变量更改消息
		broadcastEnvChange()
	} else {
		return fmt.Errorf("暂不支持在 %s 系统上设置环境变量", runtime.GOOS)
	}
	return nil
}

// addToPath 添加目录到PATH环境变量
func addToPath(dir string) error {
	// 获取系统PATH环境变量
	cmd := exec.Command("REG", "QUERY", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "Path")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("获取系统PATH失败: %v", err)
	}

	// 使用正则表达式解析PATH值
	re := regexp.MustCompile(`REG_SZ\s+(.+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return fmt.Errorf("解析系统PATH失败")
	}
	currentPath := strings.TrimSpace(matches[1])

	// 规范化所有路径分隔符
	currentPath = strings.ReplaceAll(currentPath, "/", "\\")
	paths := strings.Split(currentPath, string(os.PathListSeparator))

	// 清理并规范化所有路径
	var cleanPaths []string
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			cleanPaths = append(cleanPaths, filepath.Clean(path))
		}
	}

	// 检查路径是否已存在
	dir = filepath.Clean(dir)
	for _, path := range cleanPaths {
		if strings.EqualFold(path, dir) { // Windows 路径不区分大小写
			return nil // 路径已存在，无需添加
		}
	}

	// 添加新路径
	cleanPaths = append(cleanPaths, dir)
	newPath := strings.Join(cleanPaths, string(os.PathListSeparator))

	if runtime.GOOS == "windows" {
		// 在Windows上使用REG ADD命令更新系统PATH
		cmd = exec.Command("REG", "ADD", "HKEY_LOCAL_MACHINE\\SYSTEM\\CurrentControlSet\\Control\\Session Manager\\Environment", "/v", "Path", "/t", "REG_SZ", "/d", newPath, "/f")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("更新系统PATH失败: %v\n%s", err, output)
		}

		// 更新当前进程的PATH
		if err := os.Setenv("PATH", newPath); err != nil {
			return fmt.Errorf("更新当前进程PATH失败: %v", err)
		}

		// 广播环境变量更改消息
		broadcastEnvChange()
	} else {
		return fmt.Errorf("暂不支持在 %s 系统上更新PATH", runtime.GOOS)
	}
	return nil
}

// broadcastEnvChange 广播环境变量更改消息
func broadcastEnvChange() {
	// 使用 SendMessageTimeout 发送 WM_SETTINGCHANGE 消息
	// 这需要调用 Windows API，这里我们使用 PowerShell 命令来实现
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

// verifyGoInstallation 验证Go安装是否正确
func verifyGoInstallation() error {
	// 运行 go version 命令验证安装
	cmd := exec.Command("go", "version")
	cmd.Env = os.Environ() // 确保使用更新后的环境变量
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("运行 'go version' 失败: %v\n%s", err, output)
	}
	return nil
}
