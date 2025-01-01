package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"go-version-switch/internal/version"
)

var (
	listFlag    bool
	updateFlag  bool
	installFlag string
	useFlag     string
	archFlag    string
	baseDir     string
)

func init() {
	// 获取可执行文件所在目录作为基础目录
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("获取程序路径失败: %v\n", err)
		os.Exit(1)
	}
	baseDir = filepath.Join(filepath.Dir(execPath), "data")

	// 解析命令行参数
	flag.BoolVar(&listFlag, "list", false, "列出所有可用的Go版本")
	flag.BoolVar(&updateFlag, "update", false, "强制更新版本列表")
	flag.StringVar(&installFlag, "install", "", "安装指定版本")
	flag.StringVar(&useFlag, "use", "", "切换到指定版本")
	flag.StringVar(&archFlag, "arch", "", "指定架构 (x86/x64/arm/arm64)")
}

func main() {
	flag.Parse()

	// 创建基础目录
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Printf("创建数据目录失败: %v\n", err)
		os.Exit(1)
	}

	// 处理版本列表命令
	if listFlag {
		list, err := version.GetVersionList(baseDir, updateFlag)
		if err != nil {
			fmt.Printf("获取版本列表失败: %v\n", err)
			os.Exit(1)
		}
		list.PrintVersionList()
		return
	}

	// 处理安装命令
	if installFlag != "" {
		opts := version.InstallOptions{
			Version: installFlag,
			Arch:    archFlag,
		}
		if err := version.InstallVersion(baseDir, opts); err != nil {
			fmt.Printf("安装失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 处理切换版本命令
	if useFlag != "" {
		opts := version.InstallOptions{
			Version: useFlag,
			Arch:    archFlag,
		}
		if err := version.UseVersion(baseDir, opts); err != nil {
			fmt.Printf("切换版本失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 如果没有指定任何命令，显示帮助信息
	flag.Usage()
}
