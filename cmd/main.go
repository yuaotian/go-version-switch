package main

import (
	"flag"
	"fmt"
	"os"

	"go-bits-switch/internal/config"
	"go-bits-switch/internal/download"
	"go-bits-switch/internal/env"
	"go-bits-switch/internal/version"
)

func main() {
	// 解析命令行参数
	showVersion := flag.Bool("version", false, "显示当前Go版本")
	listVersions := flag.Bool("list", false, "列出可用的Go版本")
	forceUpdate := flag.Bool("update", false, "强制更新版本列表")
	installVersion := flag.String("install", "", "安装指定版本")
	useVersion := flag.String("use", "", "切换到指定版本")
	backupEnv := flag.Bool("backup", false, "备份当前环境变量")
	restoreEnv := flag.String("restore", "", "从指定文件恢复环境变量")

	flag.Parse()

	// 获取配置
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 处理命令
	switch {
	case *showVersion:
		handleShowVersion()
	case *listVersions:
		handleListVersions(cfg, *forceUpdate)
	case *installVersion != "":
		handleInstallVersion(cfg, *installVersion)
	case *useVersion != "":
		handleUseVersion(cfg, *useVersion)
	case *backupEnv:
		handleBackupEnv()
	case *restoreEnv != "":
		handleRestoreEnv(*restoreEnv)
	default:
		flag.Usage()
	}
}

// handleShowVersion 显示当前Go版本
func handleShowVersion() {
	current, err := version.GetCurrentVersion()
	if err != nil {
		fmt.Printf("获取当前版本失败: %v\n", err)
		return
	}

	fmt.Printf("当前Go版本: %s\n", current.Version)
	fmt.Printf("安装路径: %s\n", current.Path)
	fmt.Printf("系统架构: %s\n", current.Arch)
}

// handleListVersions 列出可用的Go版本
func handleListVersions(cfg *config.Config, forceUpdate bool) {
	list, err := version.GetVersionList(cfg.BaseDir, forceUpdate)
	if err != nil {
		fmt.Printf("获取版本列表失败: %v\n", err)
		return
	}

	list.PrintVersionList()
}

// handleInstallVersion 安装指定版本
func handleInstallVersion(cfg *config.Config, ver string) {
	// 检查版本号格式
	ver = version.ParseVersion(ver)
	if !version.IsValidVersion(ver) {
		fmt.Printf("无效的版本号格式: %s\n", ver)
		return
	}

	// 获取版本信息
	info, err := version.GetVersionInfo(ver)
	if err != nil {
		fmt.Printf("获取版本信息失败: %v\n", err)
		return
	}

	// 下载并安装
	fmt.Printf("开始下载 Go %s...\n", ver)
	if err := download.DownloadAndInstall(info, cfg.BaseDir); err != nil {
		fmt.Printf("安装失败: %v\n", err)
		return
	}

	fmt.Printf("Go %s 安装成功!\n", ver)
}

// handleUseVersion 切换到指定版本
func handleUseVersion(cfg *config.Config, ver string) {
	// 检查版本号格式
	ver = version.ParseVersion(ver)
	if !version.IsValidVersion(ver) {
		fmt.Printf("无效的版本号格式: %s\n", ver)
		return
	}

	// 获取已安装版本
	installed, err := version.GetInstalledVersions(cfg.BaseDir)
	if err != nil {
		fmt.Printf("获取已安装版本失败: %v\n", err)
		return
	}

	// 查找指定版本
	var targetVersion *version.GoVersion
	for _, v := range installed {
		if v.Version == ver {
			targetVersion = v
			break
		}
	}

	if targetVersion == nil {
		fmt.Printf("版本 %s 未安装，请先使用 -install 安装\n", ver)
		return
	}

	// 备份当前环境变量
	if err := env.BackupEnv(); err != nil {
		fmt.Printf("备份环境变量失败: %v\n", err)
		return
	}

	// 更新环境变量
	if err := env.UpdateGoRoot(targetVersion.Path); err != nil {
		fmt.Printf("更新GOROOT失败: %v\n", err)
		return
	}

	fmt.Printf("已切换到 Go %s\n", ver)
	fmt.Println("请重新打开终端或运行 'go version' 验证切换是否成功")
}

// handleBackupEnv 备份环境变量
func handleBackupEnv() {
	if err := env.BackupEnv(); err != nil {
		fmt.Printf("备份环境变量失败: %v\n", err)
		return
	}

	fmt.Println("环境变量已备份")
}

// handleRestoreEnv 恢复环境变量
func handleRestoreEnv(backupFile string) {
	if err := env.RestoreEnv(backupFile); err != nil {
		fmt.Printf("恢复环境变量失败: %v\n", err)
		return
	}

	fmt.Println("环境变量已恢复")
	fmt.Println("请重新打开终端使更改生效")
}
