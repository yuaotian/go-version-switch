package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"go-version-switch/internal/config"
)

// InstallOptions 安装选项
type InstallOptions struct {
	Version string // 版本号
	Arch    string // 架构
}

// InstallVersion 安装指定版本的Go
func InstallVersion(baseDir string, opts InstallOptions) error {
	// 确保配置目录存在
	configDir := filepath.Join(baseDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 如果未指定架构，使用当前系统架构
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}

	// 转换架构名称
	arch := normalizeArch(opts.Arch)
	if arch == "" {
		return fmt.Errorf("不支持的架构: %s", opts.Arch)
	}

	// 获取版本列表
	list, err := GetVersionList(baseDir, false)
	if err != nil {
		return fmt.Errorf("获取版本列表失败: %v", err)
	}

	// 查找指定版本和架构的发布版本
	var targetRelease *GoRelease
	for _, v := range list.Versions {
		if v.Version == opts.Version && strings.EqualFold(v.Arch, arch) {
			targetRelease = v
			break
		}
	}

	if targetRelease == nil {
		return fmt.Errorf("未找到版本 %s 的 %s 架构版本", opts.Version, arch)
	}

	// 下载并解压Go版本
	if err := DownloadAndExtract(targetRelease, baseDir); err != nil {
		return fmt.Errorf("安装失败: %v", err)
	}

	// 保存版本信息到配置
	versionDir := filepath.Join(baseDir, "go-version", fmt.Sprintf("go-%s-%s", opts.Version, strings.ToLower(arch)))
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	if err := cfg.AddVersion(opts.Version, versionDir); err != nil {
		return fmt.Errorf("保存版本信息失败: %v", err)
	}

	return nil
}

// UseVersion 切换到指定版本
func UseVersion(baseDir string, opts InstallOptions) error {
	// 如果未指定架构，使用当前系统架构
	if opts.Arch == "" {
		opts.Arch = runtime.GOARCH
	}

	// 转换架构名称
	arch := normalizeArch(opts.Arch)
	if arch == "" {
		return fmt.Errorf("不支持的架构: %s", opts.Arch)
	}

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("加载配置失败: %v", err)
	}

	// 检查版本是否已安装
	versionDir := filepath.Join(baseDir, "go-version", fmt.Sprintf("go-%s-%s", opts.Version, strings.ToLower(arch)))
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("版本 %s (%s) 未安装，请先安装", opts.Version, arch)
	}

	// 设置为当前Go环境
	if err := SetAsCurrentGo(versionDir); err != nil {
		return fmt.Errorf("切换版本失败: %v", err)
	}

	// 更新配置中的当前版本
	if err := cfg.SetCurrentVersion(opts.Version); err != nil {
		return fmt.Errorf("保存当前版本信息失败: %v", err)
	}

	fmt.Printf("✅ 已成功切换到 Go %s (%s)\n", opts.Version, arch)
	fmt.Printf("⚠️ 请重启终端和编辑器以使更改生效\n")

	return nil
}

// normalizeArch 标准化架构名称
func normalizeArch(arch string) string {
	arch = strings.ToLower(arch)
	switch arch {
	case "x86", "386", "32":
		return "x86"
	case "x64", "amd64", "x86-64", "64":
		return "x86-64"
	case "arm":
		return "ARM"
	case "arm64":
		return "ARM64"
	default:
		return ""
	}
}
