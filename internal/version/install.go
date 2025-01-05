package version

import (
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

// InstallVersion 优化后的安装函数
func InstallVersion(baseDir string, opts InstallOptions) error {
    // 验证和准备安装环境
    if err := prepareInstallEnvironment(baseDir, &opts); err != nil {
        return err
    }

    // 查找目标版本
    targetRelease, err := findTargetRelease(baseDir, opts)
    if err != nil {
        return err
    }

    // 处理本地文件
    localFile := NewLocalFileHandler(baseDir, opts, targetRelease)
    if err := localFile.Handle(); err != nil {
        return err
    }

    // 保存版本信息
    return saveVersionConfig(baseDir, opts)
}

// LocalFileHandler 本地文件处理器
type LocalFileHandler struct {
    BaseDir       string
    Opts          InstallOptions
    TargetRelease *GoRelease
    LocalPath     string
}

func NewLocalFileHandler(baseDir string, opts InstallOptions, release *GoRelease) *LocalFileHandler {
    downloadDir := filepath.Join(baseDir, "down")
    filename := fmt.Sprintf("go%s.windows-%s.zip",
        opts.Version, strings.ToLower(opts.Arch))

    return &LocalFileHandler{
        BaseDir:       baseDir,
        Opts:          opts,
        TargetRelease: release,
        LocalPath:     filepath.Join(downloadDir, filename),
    }
}

func (h *LocalFileHandler) Handle() error {
    if _, err := os.Stat(h.LocalPath); err == nil {
        return h.handleExistingFile()
    }
    return h.handleNewDownload()
}

// UseVersion 切换到指定版本
func UseVersion(baseDir string, opts InstallOptions) error {
    // 如果未指定架构，使用当前系统架构
    if opts.Arch == "" {
        opts.Arch = runtime.GOARCH
    }

    // 转换架构名称
    //fmt.Println("输入架构 ",opts.Arch)
    arch := normalizeArch(opts.Arch)
   // fmt.Println("标准化架构 ",arch)
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

// extractGo 解压Go安装包
func extractGo(zipPath, version, arch string) (string, error) {
    // 构建解压目录
    extractDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "go-version")
    if err := os.MkdirAll(extractDir, 0755); err != nil {
        return "", fmt.Errorf("创建解压目录失败: %v", err)
    }

    // 目标目录
    targetDir := filepath.Join(extractDir, fmt.Sprintf("go-%s-%s", version, arch))

    fmt.Printf("📂 解压目录: %s\n", targetDir)
    // 检查并清理已存在的目录
    if _, err := os.Stat(targetDir); err == nil {
        fmt.Printf("🗑️  检测到已存在的目录: %s\n", targetDir)
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

    // 解压文件
    if err := unzip(zipPath, targetDir); err != nil {
        return "", fmt.Errorf("❌ 解压失败: %v", err)
    }
    fmt.Printf("✅ 解压完成，安装目录: %s\n", targetDir)
    fmt.Printf("✨ Go %s (%s) 解压成功!\n", version, arch)

    // 询问是否设置环境变量
    fmt.Print("\n🔧 是否立即将此版本设置为系统Go环境? [Y/n] ")
    var answer string
    fmt.Scanln(&answer)
    if answer == "" || strings.ToLower(answer) == "y" {
        if err := SetAsCurrentGo(targetDir); err != nil {
            return "", fmt.Errorf("❌ 设置环境变量失败: %v", err)
        }
        fmt.Printf("✅ 环境变量设置成功\n")
        fmt.Printf("⚠️ 注意：某些程序可能需要重启才能识别新的环境变量：\n")
        fmt.Printf("   • 终端 (PowerShell, CMD 等)\n")
        fmt.Printf("   • 编辑器 (VSCode, IntelliJ IDEA 等)\n")
        fmt.Printf("   • 其他使用Go环境的应用\n")
        fmt.Println("  • 如果环境变量设置失败，请手动设置GOROOT环境变量")
        fmt.Println("🔄 如果需要回滚，请使用：go-version-switch -rollback")

    }
    return targetDir, nil
}

// prepareInstallEnvironment 准备安装环境
func prepareInstallEnvironment(baseDir string, opts *InstallOptions) error {
    // 确保配置目录存在
    configDir := filepath.Join(baseDir, "config")
    if err := os.MkdirAll(configDir, 0755); err != nil {
        return fmt.Errorf("创建配置目录失败: %v", err)
    }

    // 如果未指定架构，使用当前系统架构
    if opts.Arch == "" {
        fmt.Printf("🔎 未指定架构，使用当前系统架构: %s\n", runtime.GOARCH)
        opts.Arch = runtime.GOARCH
    }

    // 转换架构名称
   // fmt.Println("输入架构 ",opts.Arch)
    arch := normalizeArch(opts.Arch)
   // fmt.Println("标准化架构 ",arch)
    if arch == "" {
        return fmt.Errorf("不支持的架构: %s", opts.Arch)
    }

    return nil
}

// findTargetRelease 查找目标版本
func findTargetRelease(baseDir string, opts InstallOptions) (*GoRelease, error) {
    // 获取版本列表
    list, err := GetVersionList(baseDir, false)
    if err != nil {
        return nil, fmt.Errorf("获取版本列表失败: %v", err)
    }
   // fmt.Println("输入架构 ",opts.Arch)
    // 查找指定版本和架构的发布版本
    arch := normalizeArch(opts.Arch)
   // fmt.Println("标准化架构 ",arch)
    for _, v := range list.Versions {
        if v.Version == opts.Version && strings.EqualFold(v.Arch, arch) {
            return v, nil
        }
    }

    return nil, fmt.Errorf("未找到版本 %s 的 %s 架构版本", opts.Version, arch)
}

// saveVersionConfig 保存版本配置
func saveVersionConfig(baseDir string, opts InstallOptions) error {
    versionDir := filepath.Join(baseDir, "go-version",
        fmt.Sprintf("go-%s-%s", opts.Version, strings.ToLower(opts.Arch)))

    cfg, err := config.LoadConfig()
    if err != nil {
        return fmt.Errorf("加载配置失败: %v", err)
    }

    if err := cfg.AddVersion(opts.Version, versionDir); err != nil {
        return fmt.Errorf("保存版本信息失败: %v", err)
    }

    return nil
}

// LocalFileHandler 的方法实现
func (h *LocalFileHandler) handleExistingFile() error {
    fmt.Printf("📦 发现本地已有安装包: %s\n", h.LocalPath)
    fmt.Println("🔍 正在验证文件完整性...")

    verifier := &FileVerifier{
        FilePath:     h.LocalPath,
        ExpectedHash: h.TargetRelease.SHA256,
    }

    if err := verifier.Verify(); err == nil {
        fmt.Println("✅ 本地文件验证成功，将直接使用")
        _, err := extractGo(h.LocalPath, h.Opts.Version, h.Opts.Arch)
        if err != nil {
            return fmt.Errorf("%v", err)
        }

        return nil
    } else {
        fmt.Printf("⚠️ 本地文件验证失败: %v\n", err)
        fmt.Println("🔄 将重新下载文件...")
        os.Remove(h.LocalPath)
        return h.handleNewDownload()
    }
}

func (h *LocalFileHandler) handleNewDownload() error {
    if err := DownloadAndExtract(h.TargetRelease, h.BaseDir); err != nil {
        return fmt.Errorf("%v", err)
    }
    return nil
}
