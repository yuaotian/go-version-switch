package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go-version-switch/internal/version"
)

// Command 定义命令结构
type Command struct {
	Name        string
	Description string
	Example     string
}

var (
	listFlag     bool
	updateFlag   bool
	installFlag  string
	useFlag      string
	archFlag     string
	rollbackFlag bool
	helpFlag     bool
	baseDir      string
)

// 定义所有支持的命令
var commands = []Command{
	{
		Name:        "list",
		Description: "列出所有可用的Go版本",
		Example:     "go-version-switch -list",
	},
	{
		Name:        "update",
		Description: "强制更新可用的Go版本列表",
		Example:     "go-version-switch -list -update",
	},
	{
		Name:        "install",
		Description: "安装指定版本的Go",
		Example:     "go-version-switch -install 1.20.1 -arch x64",
	},
	{
		Name:        "use",
		Description: "切换到指定的Go版本",
		Example:     "go-version-switch -use 1.20.1",
	},
	{
		Name:        "rollback",
		Description: "回滚到上一次的环境变量配置",
		Example:     "go-version-switch -rollback",
	},
	{
		Name:        "help",
		Description: "查看帮助信息",
		Example:     "go-version-switch -help",
	},
}

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
	flag.BoolVar(&rollbackFlag, "rollback", false, "回滚到上一次的环境变量配置")
}

// printHelp 打印格式化的帮助信息
func printHelp() {
	fmt.Println(`
  ____        __     __            _               ____          _ _       _      
 / ___| ___   \ \   / /__ _ __ ___(_) ___  _ __   / ___|_      _(_) |_ ___| |__   
 | |  _ / _ \  \ \ / / _ \ '__/ __| |/ _ \| '_ \  \___ \ \ /\ / / | __/ __| '_ \  
 | |_| | (_) |  \ V /  __/ |  \__ \ | (_) | | | |  ___) \ V  V /| | || (__| | | | 
  \____|\___/    \_/ \___|_|  |___/_|\___/|_| |_| |____/ \_/\_/ |_|\__\___|_| |_| 
                                                                                   `)
	fmt.Println("\n🚀 Go Version Manager - 帮助信息")
	fmt.Println("\n📋 用法:")
	fmt.Printf("  %s [命令] [参数]\n", filepath.Base(os.Args[0]))

	fmt.Println("\n⚡ 支持的命令:")
	for _, cmd := range commands {
		fmt.Printf("  -%-12s %s\n", cmd.Name, cmd.Description)
	}

	fmt.Println("\n🔧 参数说明:")
	fmt.Println("  -arch string    指定架构，支持以下格式:")
	fmt.Println("                  • x86, 386, 32       (32位)")
	fmt.Println("                  • x64, amd64, x86-64 (64位)")
	fmt.Println("                  • arm                (ARM)")
	fmt.Println("                  • arm64              (ARM64)")

	fmt.Println("\n📝 使用示例:")
	fmt.Println("  1. 列出可用版本:")
	fmt.Printf("     %s -list\n", filepath.Base(os.Args[0]))

	fmt.Println("\n  2. 安装指定版本:")
	fmt.Printf("     %s -install 1.20.1 -arch x64\n", filepath.Base(os.Args[0]))

	fmt.Println("\n  3. 切换到指定版本:")
	fmt.Printf("     %s -use 1.20.1\n", filepath.Base(os.Args[0]))

	fmt.Println("\n  4. 直接切换架构:")
	fmt.Printf("     %s -arch x64\n", filepath.Base(os.Args[0]))
	fmt.Printf("     %s -arch x86\n", filepath.Base(os.Args[0]))

	fmt.Println("\n  5. 回滚环境变量:")
	fmt.Printf("     %s -rollback\n", filepath.Base(os.Args[0]))

	fmt.Println("\n  6. 强制更新版本列表:")
	fmt.Printf("     %s -list -update\n", filepath.Base(os.Args[0]))

	fmt.Println("\n📌 注意事项:")
	fmt.Println("  • 修改系统环境变量需要管理员权限")
	fmt.Println("  • 切换版本后需要重启终端和编辑器")
	fmt.Println("  • 如果安装失败，可以使用 -rollback 回滚")
	fmt.Println("  • 支持自动检测和使用本地安装包")

	fmt.Println("\n💡 目录说明:")
	fmt.Println("  • go-version/: Go版本安装目录")
	fmt.Println("  • down/: 安装包下载目录")
	fmt.Println("  • backup_env/: 环境变量备份目录")
	fmt.Println("  • config/: 配置文件目录")

	fmt.Println("\n🔗 更多信息:")
	fmt.Println("  项目地址: https://github.com/yuaotian/go-version-switch")
	fmt.Println("  问题反馈: https://github.com/yuaotian/go-version-switch/issues")
}

// findSimilarCommand 查找相似命令
func findSimilarCommand(input string) string {
	input = strings.TrimPrefix(input, "-")
	var bestMatch string
	bestScore := 0

	for _, cmd := range commands {
		score := 0
		shorter, longer := input, cmd.Name
		if len(shorter) > len(longer) {
			shorter, longer = longer, shorter
		}

		for i := range shorter {
			if i < len(longer) && shorter[i] == longer[i] {
				score++
			}
		}

		if score > bestScore {
			bestScore = score
			bestMatch = cmd.Name
		}
	}

	// 如果相似度超过50%，返回建议
	if float64(bestScore)/float64(len(input)) > 0.5 {
		return bestMatch
	}
	return ""
}

// printRefreshTips 打印环境变量刷新提示
func printRefreshTips() {
	fmt.Println("\n💡 如果终端环境变量未更新，请尝试以下方法手动刷新:")
	fmt.Println("\n[PowerShell]")
	fmt.Println("方法1: $env:Path = [System.Environment]::GetEnvironmentVariable(\"Path\",\"Machine\") + \";\" + [System.Environment]::GetEnvironmentVariable(\"Path\",\"User\")")
	fmt.Println("方法2: refreshenv  # 需要安装 Chocolatey")
	fmt.Println("\n[CMD]")
	fmt.Println("方法1: refreshenv  # 需要安装 Chocolatey")
	fmt.Println("方法2: set PATH=%PATH%")
}

func main() {
	flag.Parse()

	// 检查未识别的参数
	for _, arg := range flag.Args() {
		if strings.HasPrefix(arg, "-") {
			if similar := findSimilarCommand(arg); similar != "" {
				fmt.Printf("未知参数: %s\n你是否想要使用 -%s?\n", arg, similar)
				for _, cmd := range commands {
					if cmd.Name == similar {
						fmt.Printf("-%s: %s\n示例: %s\n", cmd.Name, cmd.Description, cmd.Example)
						os.Exit(1)
					}
				}
			} else {
				fmt.Printf("未知参数: %s\n请使用 -h 或 --help 查看帮助信息\n", arg)
				os.Exit(1)
			}
		}
	}

	// 创建基础目录
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Printf("创建数据目录失败: %v\n", err)
		os.Exit(1)
	}

	// 处理帮助信息显示
	if helpFlag || len(os.Args) == 1 {
		printHelp()
		return
	}

	// 处理架构切换
	if archFlag != "" && !listFlag && !updateFlag &&
		installFlag == "" && useFlag == "" && !rollbackFlag {
		if err := version.HandleArchitectureSwitch(baseDir, archFlag); err != nil {
			fmt.Printf("切换架构失败: %v\n", err)
			os.Exit(1)
		}
		printRefreshTips()
		return
	}

	fmt.Println(`
  ____        __     __            _               ____          _ _       _      
 / ___| ___   \ \   / /__ _ __ ___(_) ___  _ __   / ___|_      _(_) |_ ___| |__   
 | |  _ / _ \  \ \ / / _ \ '__/ __| |/ _ \| '_ \  \___ \ \ /\ / / | __/ __| '_ \  
 | |_| | (_) |  \ V /  __/ |  \__ \ | (_) | | | |  ___) \ V  V /| | || (__| | | | 
  \____|\___/    \_/ \___|_|  |___/_|\___/|_| |_| |____/ \_/\_/ |_|\__\___|_| |_| 
                                                                                   `)
	// 处理回滚命令
	if rollbackFlag {
		if err := handleRollback(); err != nil {
			fmt.Printf("回滚失败: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 处理版本列表命令
	if listFlag {
		list, err := version.GetVersionList(baseDir, updateFlag)
		if err != nil {
			fmt.Printf("获取版本列表失败: ")
			fmt.Println(err)
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
			fmt.Printf("安装失败: ")
			fmt.Println(err)
			os.Exit(1)
		}
		printRefreshTips()
		return
	}

	// 处理切换版本命令
	if useFlag != "" {
		opts := version.InstallOptions{
			Version: useFlag,
			Arch:    archFlag,
		}
		if err := version.UseVersion(baseDir, opts); err != nil {
			fmt.Printf("切换版本失败: ")
			fmt.Println(err)
			os.Exit(1)
		}
		printRefreshTips()
		return
	}
}

// handleRollback 处理环境变量回滚
func handleRollback() error {
	// 检查管理员权限
	isAdmin, err := version.CheckAdminPrivileges()
	if err != nil {
		return fmt.Errorf("检查管理员权限失败: %v", err)
	}
	if !isAdmin {
		return fmt.Errorf("需要管理员权限才能修改系统环境变量")
	}

	// 获取最新的备份文件
	backupDir := filepath.Join(baseDir, "backup_env")
	latestBackup, err := version.GetLatestBackup(backupDir)
	if err != nil {
		return fmt.Errorf("获取备份文件失败: %v", err)
	}

	fmt.Printf("正在从备份文件恢复环境变量: %s\n", latestBackup)

	// 执行回滚
	if err := version.RestoreEnvironment(latestBackup); err != nil {
		return fmt.Errorf("回滚失败: %v", err)
	}

	printRefreshTips()
	return nil
}
