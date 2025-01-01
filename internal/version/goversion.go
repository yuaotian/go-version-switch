package version

import (
	"fmt"
	"runtime"
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

// GetDownloadInfo 获取当前系统对应的下载信息
func GetDownloadInfo(version string) *GoRelease {
	// 根据当前系统和架构选择合适的版本
	var arch string
	switch runtime.GOARCH {
	case "386":
		arch = "x86"
	case "amd64":
		arch = "x86-64"
	case "arm":
		arch = "arm"
	case "arm64":
		arch = "ARM64"
	default:
		arch = runtime.GOARCH
	}

	// 构建下载URL
	downloadURL := fmt.Sprintf("https://go.dev/dl/go%s.windows-%s.zip", version, runtime.GOARCH)

	return &GoRelease{
		Version:     version,
		Kind:        "Archive",
		OS:          "Windows",
		Arch:        arch,
		DownloadURL: downloadURL,
	}
}

// IsValidArch 检查架构是否支持
func IsValidArch(arch string) bool {
	validArchs := map[string]bool{
		"386":   true,
		"amd64": true,
		"arm":   true,
		"arm64": true,
	}
	return validArchs[arch]
}

// GetSupportedArchs 获取支持的架构列表
func GetSupportedArchs() []string {
	return []string{
		"386",   // x86
		"amd64", // x86-64
		"arm",   // ARM
		"arm64", // ARM64
	}
}

// GetArchDisplayName 获取架构的显示名称
func GetArchDisplayName(arch string) string {
	archNames := map[string]string{
		"386":   "x86",
		"amd64": "x86-64",
		"arm":   "ARM",
		"arm64": "ARM64",
	}
	if name, ok := archNames[arch]; ok {
		return name
	}
	return arch
}

// FormatVersion 格式化版本号
func FormatVersion(version string) string {
	if len(version) > 0 && version[0] != 'v' {
		return "v" + version
	}
	return version
}

// ParseVersion 解析版本号
func ParseVersion(version string) string {
	if len(version) > 0 && version[0] == 'v' {
		return version[1:]
	}
	return version
}

// GetVersionDirName 获取版本目录名称
func GetVersionDirName(version string) string {
	return fmt.Sprintf("go-version-bits-%s", version)
}

// GetVersionFromDirName 从目录名称获取版本号
func GetVersionFromDirName(dirName string) string {
	prefix := "go-version-bits-"
	if strings.HasPrefix(dirName, prefix) {
		return strings.TrimPrefix(dirName, prefix)
	}
	return ""
}
