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

const (
	exeSuffix = ".exe" // Windows 可执行文件后缀
)

// GoVersion Go版本信息
type GoVersion struct {
	Version string // 版本号
	Path    string // 安装路径
	Arch    string // 系统架构
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
		arch = "x86-64"
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
