package version

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadProgress 下载进度结构
type DownloadProgress struct {
	Total      int64
	Downloaded int64
	StartTime  time.Time
	lastRender time.Time // 上次刷新时间，用来限制刷屏频率
}

// 进度条字符
const (
	progressWidth = 40
	progressChar  = "█"
	emptyChar     = "░"
)

// DownloadAndExtract 下载并解压Go版本
func DownloadAndExtract(release *GoRelease, baseDir string) error {
	// 创建下载目录
	downloadDir := filepath.Join(baseDir, "down")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return fmt.Errorf("📁 创建下载目录失败: %v", err)
	}

	// 创建版本目录
	versionDir := filepath.Join(baseDir, "go-version")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return fmt.Errorf("📁 创建版本目录失败: %v", err)
	}
	//fmt.Println("输入架构 ",release.Arch)
	// 生成标准化的文件名
	arch := normalizeArch(release.Arch)
	//fmt.Println("标准化架构 ",arch)
	if arch == "" {
		return fmt.Errorf("不支持的架构: %s", release.Arch)
	}
	fileName := fmt.Sprintf("go%s.windows-%s.zip", release.Version, strings.ToLower(arch))
	downloadPath := filepath.Join(downloadDir, fileName)
	fmt.Printf("📥 正在下载 Go %s (%s)...\n", release.Version, release.Arch)
	fmt.Printf("📂 下载目录: %s\n", downloadDir)
	fmt.Printf("📦 目标文件: %s\n", downloadPath)

	// 检查是否已存在下载文件
	if _, err := os.Stat(downloadPath); err == nil {
		fmt.Printf("💡 发现已下载的文件: %s\n", downloadPath)
		fmt.Printf("🔍 正在验证文件完整性...\n")
		if err := verifyChecksum(downloadPath, release.SHA256); err == nil {
			fmt.Printf("✅ 文件验证成功，跳过下载\n")
		} else {
			fmt.Printf("⚠️ 文件验证失败: %v\n", err)
			fmt.Printf("🗑️ 删除损坏的文件...\n")
			if err := os.Remove(downloadPath); err != nil {
				return fmt.Errorf("删除损坏的文件失败: %v", err)
			}
			fmt.Printf("📥 开始重新下载...\n")
			if err := downloadWithProgress(release.DownloadURL, downloadPath); err != nil {
				return fmt.Errorf("❌ 下载失败: %v", err)
			}
		}
	} else {
		// 文件不存在，直接下载
		fmt.Printf("📥 开始下载文件...\n")
		if err := downloadWithProgress(release.DownloadURL, downloadPath); err != nil {
			return fmt.Errorf("❌ 下载失败: %v", err)
		}
	}

	// 验证下载文件
	fmt.Printf("🔍 正在验证文件完整性...\n")
	if err := verifyChecksum(downloadPath, release.SHA256); err != nil {
		return fmt.Errorf("❌ %v", err)
	}
	fmt.Printf("✅ 文件验证成功\n")

	// 生成解压目标目录
	targetDir := filepath.Join(versionDir, fmt.Sprintf("go-%s-%s", release.Version, strings.ToLower(release.Arch)))
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
			return fmt.Errorf("清理目录失败，请手动删除目录 %s 后重试: %v", targetDir, err)
		}
	}

	if err := unzip(downloadPath, targetDir); err != nil {
		return fmt.Errorf("❌ 解压失败: %v", err)
	}

	fmt.Printf("✨ Go %s (%s) 解压成功!\n", release.Version, release.Arch)

	// 询问是否设置环境变量（-yes / 无人值守时按默认 Y 处理）
	if confirm("\n🔧 是否立即将此版本设置为系统Go环境? [Y/n] ") {
		if err := SetAsCurrentGo(targetDir); err != nil {
			return fmt.Errorf("❌ 设置环境变量失败: %v", err)
		}
		fmt.Printf("✅ 环境变量设置成功\n")
		fmt.Printf("⚠️ 注意：某些程序可能需要重启才能识别新的环境变量：\n")
		fmt.Printf("   • 终端 (PowerShell, CMD 等)\n")
		fmt.Printf("   • 编辑器 (VSCode, IntelliJ IDEA 等)\n")
		fmt.Printf("   • 其他使用Go环境的应用\n")
		fmt.Println("  • 如果环境变量设置失败，请手动设置GOROOT环境变量")
		fmt.Println("🔄 如果需要回滚，请使用：go-version-switch -rollback")
	}

	return nil
}

// downloadWithProgress 带进度显示的下载
func downloadWithProgress(url string, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	progress := &DownloadProgress{
		Total:     resp.ContentLength,
		StartTime: time.Now(),
	}

	// 创建多重写入器，同时写入文件和计算进度
	writer := &ProgressWriter{
		Writer:   out,
		Progress: progress,
	}

	_, err = io.Copy(writer, resp.Body)
	fmt.Println() // 进度条结束后换行
	return err
}

// ProgressWriter 进度显示写入器
type ProgressWriter struct {
	Writer   io.Writer
	Progress interface {
		UpdateProgress(n int64)
	}
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.Progress.UpdateProgress(int64(n))
	return n, nil
}

// showProgress 显示下载进度。
// 注意：服务端可能不返回 Content-Length（chunked / 代理），此时 Total<=0，
// 绝不能拿它当除数，否则 percent=NaN、completed=MinInt，strings.Repeat 直接 panic。
func (p *DownloadProgress) showProgress() {
	elapsed := time.Since(p.StartTime).Seconds()
	if elapsed <= 0 {
		elapsed = 1e-9
	}
	speed := float64(p.Downloaded) / elapsed / 1024 / 1024 // MB/s

	// 长度未知：只报已下载量与速度
	if p.Total <= 0 {
		fmt.Printf("\r⏳ 已下载 %.1fMB %.1fMB/s", float64(p.Downloaded)/1024/1024, speed)
		return
	}

	percent := float64(p.Downloaded) / float64(p.Total) * 100
	if percent > 100 {
		percent = 100
	}
	if percent < 0 {
		percent = 0
	}

	completed := int(float64(progressWidth) * percent / 100)
	if completed < 0 {
		completed = 0
	}
	if completed > progressWidth {
		completed = progressWidth
	}
	bar := strings.Repeat(progressChar, completed) + strings.Repeat(emptyChar, progressWidth-completed)

	// 计算剩余时间
	eta := "计算中..."
	if speed > 0 {
		remainingBytes := p.Total - p.Downloaded
		if remainingBytes < 0 {
			remainingBytes = 0
		}
		eta = fmt.Sprintf("%.0fs", float64(remainingBytes)/(speed*1024*1024))
	}

	// 使用 \r 回到行首，刷新进度显示
	fmt.Printf("\r⏳ 下载进度: [%s] %.1f%% %.1fMB/s ETA: %s",
		bar, percent, speed, eta)
}

// verifyChecksum 验证文件校验和
func verifyChecksum(filePath string, expectedHash string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("校验和不匹配\n期望: %s\n实际: %s", expectedHash, actualHash)
	}

	return nil
}

// unzip 解压文件并显示进度
func unzip(src, dest string) error {
	// 打开zip文件
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开zip文件失败: %v", err)
	}
	defer r.Close()

	// 获取压缩包中的文件总数
	totalFiles := len(r.File)
	fmt.Printf("📦 正在解压文件 (共 %d 个文件)...\n", totalFiles)

	// 创建目标目录
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	// 遍历压缩文件
	step := totalFiles / 100
	if step < 1 {
		step = 1
	}
	for i, f := range r.File {
		// 处理文件路径，移除第一级 "go/" 目录
		fpath := strings.TrimPrefix(f.Name, "go/")

		// 构建目标路径
		fpath = filepath.Join(dest, fpath)

		// 显示进度：每 1% 刷新一次，避免上万个文件把控制台刷爆
		if i%step == 0 || i == totalFiles-1 {
			fmt.Printf("\r📦 解压进度: %d/%d", i+1, totalFiles)
		}

		// 如果是目录，创建它
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		// 创建目标文件
		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		// 打开压缩文件
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		// 复制内容
		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	fmt.Println() // 进度显示完成后换行
	return nil
}

// ProgressReader 是一个用于跟踪读取进度的 io.Reader 包装器
type ProgressReader struct {
	Reader     io.Reader
	OnProgress func(n int64)
}

func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.Reader.Read(p)
	if n > 0 {
		pr.OnProgress(int64(n))
	}
	return
}

// DownloadManager 下载管理器
type DownloadManager struct {
	URL         string
	DestPath    string
	ProgressBar *ProgressBar
	ContentSize int64
	Downloaded  int64
	StartTime   time.Time
}

func NewDownloadManager(url, destPath string) *DownloadManager {
	return &DownloadManager{
		URL:         url,
		DestPath:    destPath,
		ProgressBar: NewDefaultProgressBar(),
	}
}

func (dm *DownloadManager) Download() error {
	resp, err := http.Get(dm.URL)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()

	dm.ContentSize = resp.ContentLength
	dm.StartTime = time.Now()

	file, err := os.Create(dm.DestPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	writer := &ProgressWriter{
		Writer:   file,
		Progress: dm,
	}

	_, err = io.Copy(writer, resp.Body)
	fmt.Println() // 进度条结束后换行
	return err
}

func (dm *DownloadManager) UpdateProgress(n int64) {
	dm.Downloaded += n

	// 未知长度时退化为纯速度显示，避免除零
	if dm.ContentSize <= 0 {
		speed := float64(dm.Downloaded) / time.Since(dm.StartTime).Seconds() / 1024 / 1024
		fmt.Printf("\r⏳ 已下载 %.1fMB %.1fMB/s", float64(dm.Downloaded)/1024/1024, speed)
		return
	}

	percent := float64(dm.Downloaded) / float64(dm.ContentSize)
	speed := float64(dm.Downloaded) / time.Since(dm.StartTime).Seconds() / 1024 / 1024

	bar := dm.ProgressBar.RenderProgressBar(percent)
	eta := dm.calculateETA(speed)

	fmt.Printf("\r⏳ 下载进度: [%s] %.1f%% %.1fMB/s ETA: %s",
		bar, percent*100, speed, eta)
}

// calculateETA 计算预计剩余时间
func (dm *DownloadManager) calculateETA(speed float64) string {
	if speed <= 0 {
		return "计算中..."
	}

	remainingBytes := dm.ContentSize - dm.Downloaded
	remainingSeconds := float64(remainingBytes) / (speed * 1024 * 1024)
	return fmt.Sprintf("%.0fs", remainingSeconds)
}

func (p *DownloadProgress) UpdateProgress(n int64) {
	p.Downloaded += n
	// 限流：最多每 200ms 刷新一次，避免 32KB 一次 Write 把控制台刷爆
	if time.Since(p.lastRender) < 200*time.Millisecond && (p.Total <= 0 || p.Downloaded < p.Total) {
		return
	}
	p.lastRender = time.Now()
	p.showProgress()
}
