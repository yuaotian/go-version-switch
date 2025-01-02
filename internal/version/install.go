package version

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

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
	
	// 如果未找到版本，则返回错误
	if targetRelease == nil {
		return fmt.Errorf("未找到版本 %s 的 %s 架构版本", opts.Version, arch)
	}

	// 检查本地是否已有对应版本的压缩包
	downloadDir := filepath.Join(baseDir, "down")
	filename := fmt.Sprintf("go%s.windows-%s.zip", opts.Version, strings.ToLower(arch))
	localZipPath := filepath.Join(downloadDir, filename)

	if _, err := os.Stat(localZipPath); err == nil {
		fmt.Printf("📦 发现本地已有安装包: %s\n", localZipPath)
		// 验证文件完整性
		fmt.Println("🔍 正在验证文件完整性...")
		if err := verifyDownloadedFile(localZipPath, targetRelease.SHA256); err == nil {
			fmt.Println("✅ 本地文件验证成功，将直接使用")
			// 使用本地文件进行安装
			extractDir, err := extractGo(localZipPath, opts.Version, arch)
			if err != nil {
				return fmt.Errorf("解压失败: %v", err)
			}
			fmt.Printf("✅ 解压完成，安装目录: %s\n", extractDir)
		} else {
			fmt.Printf("⚠️ 本地文件验证失败: %v\n", err)
			fmt.Println("🔄 将重新下载文件...")
			// 删除损坏的文件
			os.Remove(localZipPath)
			// 继续下载新文件
			if err := DownloadAndExtract(targetRelease, baseDir); err != nil {
				return fmt.Errorf("安装失败: %v", err)
			}
		}
	} else {
		// 本地没有文件，直接下载
		if err := DownloadAndExtract(targetRelease, baseDir); err != nil {
			return fmt.Errorf("安装失败: %v", err)
		}
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

// verifyDownloadedFile 验证下载文件的完整性
func verifyDownloadedFile(filePath string, expectedHash string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 创建 SHA256 哈希对象
	hash := sha256.New()

	// 读取文件内容并计算哈希
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("计算文件哈希失败: %v", err)
	}

	// 获取计算出的哈希值
	actualHash := hex.EncodeToString(hash.Sum(nil))

	// 比较哈希值
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("文件哈希值不匹配\n期望值: %s\n实际值: %s", expectedHash, actualHash)
	}

	return nil
}

// downloadGo 下载指定版本的Go
func downloadGo(version, arch string, expectedHash string) (string, error) {
	// 构建下载URL和文件名
	filename := fmt.Sprintf("go%s.windows-%s.zip", version, arch)
	downloadURL := fmt.Sprintf("https://dl.google.com/go/%s", filename)

	// 创建下载目录
	downloadDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "down")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return "", fmt.Errorf("创建下载目录失败: %v", err)
	}

	downloadPath := filepath.Join(downloadDir, filename)

	// 检查文件是否已存在
	if _, err := os.Stat(downloadPath); err == nil {
		fmt.Printf("📦 发现已下载的文件: %s\n", downloadPath)
		// 验证文件完整性
		fmt.Println("🔍 正在验证文件完整性...")
		if err := verifyDownloadedFile(downloadPath, expectedHash); err == nil {
			fmt.Println("✅ 文件验证成功")
			return downloadPath, nil
		} else {
			fmt.Printf("⚠️ 文件验证失败: %v\n", err)
			fmt.Println("🔄 将重新下载文件...")
			// 删除损坏的文件
			os.Remove(downloadPath)
		}
	}

	fmt.Printf("📥 正在下载 Go %s (%s)...\n", version, arch)
	fmt.Printf("📂 下载目录: %s\n", downloadDir)
	fmt.Printf("📥 下载地址: %s\n", downloadURL)

	// TODO: 实现下载逻辑
	return "", fmt.Errorf("下载功能尚未实现")
}

// extractGo 解压Go安装包
func extractGo(zipPath, version, arch string) (string, error) {
	// 构建解压目录
	extractDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "go-version")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %v", err)
	}

	// 目标目录
	targetDir := filepath.Join(extractDir, fmt.Sprintf("go-%s-%s", version, arch))

	// 检查并清理已存在的目录
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("🗑️ 检测到已存在的目录: %s\n", targetDir)
		fmt.Println("⚠️ 如果清理失败，请确保：")
		fmt.Println("   1. 没有程序正在使用该目录下的文件")
		fmt.Println("   2. 关闭所有相关的终端和编辑器")
		fmt.Println("   3. 退出正在运行的 Go 程序")

		// 等待一小段时间，让用户有机会看到提示
		time.Sleep(2 * time.Second)

		if err := os.RemoveAll(targetDir); err != nil {
			return "", fmt.Errorf("清理目录失败，请手动删除目录 %s 后重试: %v", targetDir, err)
		}
	}

	fmt.Printf("📂 解压目录: %s\n", targetDir)
	fmt.Println("📦 正在解压文件...")

	// 打开zip文件
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("打开zip文件失败: %v", err)
	}
	defer reader.Close()

	// 获取压缩包中的文件总数
	totalFiles := len(reader.File)
	extractedFiles := 0
	lastPercent := 0

	fmt.Printf("📦 正在解压文件 (共 %d 个文件)...\n", totalFiles)
	fmt.Print("进度: [")

	// 遍历并解压文件
	for _, file := range reader.File {
		// 更新进度显示
		extractedFiles++
		percent := extractedFiles * 100 / totalFiles
		for i := lastPercent; i < percent; i++ {
			if i%2 == 0 {
				fmt.Print("=")
			}
		}
		lastPercent = percent

		// 构建目标路径
		path := filepath.Join(extractDir, file.Name)

		// 确保目标路径在解压目录内
		if !strings.HasPrefix(path, extractDir) {
			fmt.Print("]\n") // 确保在错误时关闭进度条
			return "", fmt.Errorf("非法的文件路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				fmt.Print("]\n")
				return "", fmt.Errorf("创建目录失败: %v", err)
			}
			continue
		}

		// 创建父目录
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			fmt.Print("]\n")
			return "", fmt.Errorf("创建父目录失败: %v", err)
		}

		// 创建文件
		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fmt.Print("]\n")
			return "", fmt.Errorf("创建文件失败: %v", err)
		}

		// 打开压缩文件
		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			fmt.Print("]\n")
			return "", fmt.Errorf("打开压缩文件失败: %v", err)
		}

		// 复制文件内容
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			fmt.Print("]\n")
			return "", fmt.Errorf("解压文件失败: %v", err)
		}
	}

	fmt.Print("] 100%\n")

	// 重命名解压后的目录
	goDir := filepath.Join(extractDir, "go")
	if err := os.Rename(goDir, targetDir); err != nil {
		return "", fmt.Errorf("重命名目录失败: %v", err)
	}

	fmt.Printf("✨ Go %s (%s) 解压成功!\n", version, arch)
	return targetDir, nil
}
