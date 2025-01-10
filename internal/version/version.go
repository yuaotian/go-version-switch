package version

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// GoRelease 表示一个Go版本发布信息
type GoRelease struct {
	Version       string // 版本号
	Kind          string // 类型 (Archive/Installer)
	OS            string // 操作系统
	Arch          string // 架构
	Size          string // 文件大小
	SHA256        string // SHA256校验和
	DownloadURL   string // 下载URL
	IsCurrentArch bool   // 是否为当前系统架构
}

const (
	exeSuffix = ".exe" // Windows 可执行文件后缀
)

// GoVersion Go版本信息
type GoVersion struct {
	Version string // 版本号
	Path    string // 安装路径
	Arch    string // 系统架构
}

// ParseVersion 解析版本号
func ParseVersion(version string) string {
	if len(version) > 0 && version[0] == 'v' {
		return version[1:]
	}
	return version
}

// GetCurrentVersion 获取当前系统的Go版本
func GetCurrentVersion() (*GoVersion, error) {
	cmd := exec.Command("go", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// 检查是否是因为 go 命令不存在
		if _, err := exec.LookPath("go"); err != nil {
			return nil, fmt.Errorf("未找到Go安装，请先安装Go")
		}
		return nil, fmt.Errorf("获取Go版本失败: %v", err)
	}

	// 解析版本信息
	versionStr := string(output)
	versionRegex := regexp.MustCompile(`go version go([\d.]+)`)
	matches := versionRegex.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("解析版本信息失败，输出: %s", versionStr)
	}

	// 获取Go安装路径
	goroot := os.Getenv("GOROOT")
	if goroot == "" {
		// 如果环境变量未设置，尝试从go命令获取
		cmd = exec.Command("go", "env", "GOROOT")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("获取GOROOT失败: %v", err)
		}
		goroot = strings.TrimSpace(string(output))
	}

	// 验证GOROOT路径是否存在且有效
	if _, err := os.Stat(goroot); err != nil {
		return nil, fmt.Errorf("GOROOT路径无效: %s", goroot)
	}

	// 获取系统架构
	arch := runtime.GOARCH
	switch arch {
	case "386":
		arch = "x86"
	case "amd64":
		arch = "amd64"
	case "arm":
		arch = "ARM"
	case "arm64":
		arch = "ARM64"
	}

	return &GoVersion{
		Version: matches[1],
		Path:    goroot,
		Arch:    arch,
	}, nil
}

// GetInstalledVersions 获取已安装的Go版本列表
func GetInstalledVersions(baseDir string) ([]*GoVersion, error) {
	// 检查并创建基础目录
	versionDir := filepath.Join(baseDir, "go-version")
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		if err := os.MkdirAll(versionDir, 0755); err != nil {
			return nil, fmt.Errorf("创建Go版本目录失败: %v", err)
		}
		return []*GoVersion{}, nil
	}

	var versions []*GoVersion

	// 遍历目录获取已安装版本
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil, fmt.Errorf("读取版本目录失败: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			dirName := entry.Name()
			if strings.HasPrefix(dirName, "go-") {
				// 从目录名解析版本号和架构
				parts := strings.Split(strings.TrimPrefix(dirName, "go-"), "-")
				if len(parts) >= 2 {
					version := parts[0]
					arch := parts[1]
					path := filepath.Join(versionDir, dirName)

					// 检查是否是有效的Go安装目录
					if isValidGoRoot(path) {
						versions = append(versions, &GoVersion{
							Version: version,
							Path:    path,
							Arch:    arch,
						})
					}
				}
			}
		}
	}

	return versions, nil
}

// IsValidVersion 检查版本号格式是否正确
func IsValidVersion(version string) bool {
	// 移除可能的 'v' 前缀
	version = strings.TrimPrefix(version, "v")

	// 检查版本号格式
	parts := strings.Split(version, ".")
	if len(parts) < 2 || len(parts) > 3 {
		return false
	}

	// 验证每个部分都是有效的数字
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err != nil || num < 0 {
			return false
		}
	}

	return true
}

// normalizeArch 标准化架构名称
func normalizeArch(arch string) string {
	arch = strings.ToLower(arch)
	switch arch {
	case "86", "x86", "386", "32":
		return "x86"
	case "64", "amd64", "x86-64", "x64":
		return "amd64"
	case "arm":
		return "ARM"
	case "arm64":
		return "ARM64"
	default:
		return ""
	}

}

// checkDownloadDirectory 检查下载目录中的安装包
func checkDownloadDirectory(baseDir, targetArch string) error {
	downDir := filepath.Join(baseDir, "down")
	if _, err := os.Stat(downDir); err == nil {
		entries, err := os.ReadDir(downDir)
		if err == nil {
			var zipFiles []string
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				name := strings.ToLower(entry.Name())
				// 检查是否是zip文件且包含目标架构
				if strings.HasSuffix(name, ".zip") && strings.Contains(name, strings.ToLower(targetArch)) {
					zipFiles = append(zipFiles, entry.Name())
				}
			}

			if len(zipFiles) > 0 {
				fmt.Printf("✅ 在下载目录找到 %d 个 %s 架构安装包\n", len(zipFiles), targetArch)

				var selectedZip string
				if len(zipFiles) == 1 {
					selectedZip = zipFiles[0]
					fmt.Printf("📦 找到单个安装包: %s\n", selectedZip)
				} else {
					fmt.Println("\n📋 找到多个安装包:")
					for i, zip := range zipFiles {
						fmt.Printf("   [%d] %s\n", i+1, zip)
					}

					var choice int
					fmt.Print("\n⌨️  请选择要使用的安装包 (输入数字): ")
					_, err := fmt.Scanf("%d", &choice)
					if err != nil || choice < 1 || choice > len(zipFiles) {
						return fmt.Errorf("❌ 无效的选择")
					}
					selectedZip = zipFiles[choice-1]
					fmt.Printf("✅ 已选择: %s\n\n", selectedZip)
				}

				// 执行安装
				zipPath := filepath.Join(downDir, selectedZip)
				fmt.Printf("🚀 开始安装 %s...\n", selectedZip)

				// 提取版本号
				versionMatch := regexp.MustCompile(`go(\d+\.\d+\.\d+)`).FindStringSubmatch(selectedZip)
				if len(versionMatch) < 2 {
					return fmt.Errorf("❌ 无法从文件名解析版本号: %s", selectedZip)
				}

				opts := InstallOptions{
					Version: versionMatch[1],
					Arch:    targetArch,
					ZipPath: zipPath, // 使用本地zip文件
				}

				if err := InstallVersion(baseDir, opts); err != nil {
					return fmt.Errorf("❌ 安装失败: %v", err)
				}
				return nil
			}
		}
	}
	return fmt.Errorf("❌ 未找到有效的 %s 架构安装包", targetArch)
}

// HandleArchitectureSwitch 处理架构切换
func HandleArchitectureSwitch(baseDir string, archFlag string) error {
	fmt.Println("\n🔍 开始搜索架构目录...")

	// 标准化用户输入的架构名称
	targetArch := normalizeArch(archFlag)
	if targetArch == "" {
		return fmt.Errorf("❌ 不支持的架构类型: %s", archFlag)
	}

	fmt.Printf("🎯 目标架构: %s\n", targetArch)

	// 获取 go-version 目录
	goVersionDir := filepath.Join(baseDir, "go-version")
	if _, err := os.Stat(goVersionDir); os.IsNotExist(err) {
		return fmt.Errorf("❌ go-version 目录不存在: %s", goVersionDir)
	}

	fmt.Printf("📂 正在检查目录: %s\n", goVersionDir)

	// 读取目录内容
	entries, err := os.ReadDir(goVersionDir)
	if err != nil {
		return fmt.Errorf("❌ 读取目录失败: %v", err)
	}

	// 查找匹配的架构目录
	var matchedDirs []string
	var invalidDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		dirPath := filepath.Join(goVersionDir, dirName)

		// 检查目录完整性
		isValid := isValidGoRoot(dirPath)

		// 处理不同的命名格式
		var foundArch string

		// 格式1: go-1.20.1-amd64
		if strings.HasPrefix(dirName, "go-") {
			parts := strings.Split(dirName, "-")
			if len(parts) >= 3 {
				foundArch = normalizeArch(parts[len(parts)-1])
			}
		}

		// 格式2: amd64
		if foundArch == "" {
			foundArch = normalizeArch(dirName)
		}

		// 格式3: x86-64 或 x86
		if foundArch == "" && (strings.Contains(dirName, "x86") || strings.Contains(dirName, "arm")) {
			foundArch = normalizeArch(dirName)
		}

		if foundArch == targetArch {
			if isValid {
				matchedDirs = append(matchedDirs, dirName)
			} else {
				invalidDirs = append(invalidDirs, dirName)
			}
		}
	}

	// 如果没有找到有效目录，或者目录不完整，检查 down 目录
	if len(matchedDirs) == 0 {
		if len(invalidDirs) > 0 {
			fmt.Printf("⚠️ 发现 %d 个不完整的 %s 架构目录，正在检查下载目录...\n", len(invalidDirs), targetArch)
			for _, dir := range invalidDirs {
				fmt.Printf("   • %s\n", dir)
			}
		} else {
			fmt.Printf("⚠️ 未找到有效的 %s 架构目录，正在检查下载目录...\n", targetArch)
		}

		// 检查下载目录
		if err := checkDownloadDirectory(baseDir, targetArch); err != nil {
			if len(invalidDirs) > 0 {
				return fmt.Errorf("❌ 存在不完整的目录且未找到有效的安装包，请手动修复或重新安装")
			}
			return err
		}
		return nil
	}

	fmt.Printf("✅ 找到 %d 个有效的 %s 架构目录\n", len(matchedDirs), targetArch)

	var selectedDir string
	if len(matchedDirs) == 1 {
		selectedDir = matchedDirs[0]
		fmt.Printf("📌 找到单个架构目录: %s\n", selectedDir)
	} else {
		fmt.Println("\n📋 找到多个匹配的架构目录:")
		for i, dir := range matchedDirs {
			fmt.Printf("   [%d] %s\n", i+1, dir)
		}

		var choice int
		fmt.Print("\n⌨️  请选择要使用的架构 (输入数字): ")
		_, err := fmt.Scanf("%d", &choice)
		if err != nil || choice < 1 || choice > len(matchedDirs) {
			return fmt.Errorf("❌ 无效的选择")
		}
		selectedDir = matchedDirs[choice-1]
		fmt.Printf("✅ 已选择: %s\n\n", selectedDir)
	}

	// 设置选中的目录为当前Go环境
	goRoot := filepath.Join(goVersionDir, selectedDir)

	// 验证目录完整性
	requiredFiles := []string{
		"bin/go" + exeSuffix,
		"pkg",
		"src",
	}

	fmt.Printf("🔍 正在验证目录完整性: %s\n", goRoot)
	for _, file := range requiredFiles {
		path := filepath.Join(goRoot, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			fmt.Printf("❌ 目录不完整，缺少必要文件: %s\n", file)
			fmt.Println("🔄 尝试从下载目录安装...")
			return checkDownloadDirectory(baseDir, targetArch)
		}
	}
	fmt.Println("✅ 目录完整性验证通过")

	if err := SetAsCurrentGo(goRoot); err != nil {
		return fmt.Errorf("❌ 设置Go环境失败: %v", err)
	}

	fmt.Printf("\n🎉 成功切换到架构: %s\n", selectedDir)
	fmt.Printf("📂 Go安装路径: %s\n", goRoot)
	return nil
}
