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
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("获取Go版本失败: %v", err)
	}

	// 解析版本信息
	versionStr := string(output)
	versionRegex := regexp.MustCompile(`go version go([\d.]+)`)
	matches := versionRegex.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return nil, fmt.Errorf("解析版本信息失败")
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

	return &GoVersion{
		Version: matches[1],
		Path:    goroot,
		Arch:    runtime.GOARCH,
	}, nil
}

// GetInstalledVersions 获取已安装的Go版本列表
func GetInstalledVersions(baseDir string) ([]*GoVersion, error) {
	// 检查并创建基础目录
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			return nil, fmt.Errorf("创建Go版本目录失败: %v", err)
		}
		return []*GoVersion{}, nil
	}

	var versions []*GoVersion

	// 遍历目录获取已安装版本
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != baseDir {
			// 检查是否是有效的Go安装目录
			if _, err := os.Stat(filepath.Join(path, "bin", "go"+exeSuffix)); err == nil {
				version := filepath.Base(path)
				versions = append(versions, &GoVersion{
					Version: strings.TrimPrefix(version, "go"),
					Path:    path,
					Arch:    runtime.GOARCH,
				})
			}
			return filepath.SkipDir
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("遍历Go版本目录失败: %v", err)
	}

	return versions, nil
}

// IsValidVersion 检查版本号格式是否正确
func IsValidVersion(version string) bool {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return false
	}
	for _, part := range parts {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}
	return true
}
