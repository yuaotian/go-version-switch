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

	// 生成目标文件名和路径
	fileName := filepath.Base(release.DownloadURL)
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

	// 检查目标目录是否已存在
	if _, err := os.Stat(targetDir); err == nil {
		fmt.Printf("🗑️ 清理已存在的目录: %s\n", targetDir)
		if err := os.RemoveAll(targetDir); err != nil {
			return fmt.Errorf("❌ 清理目录失败: %v", err)
		}
		fmt.Printf("✅ 目录清理完成\n")
	}

	// 解压文件
	fmt.Printf("📦 正在解压文件...\n")
	if err := unzip(downloadPath, targetDir); err != nil {
		return fmt.Errorf("❌ 解压失败: %v", err)
	}

	fmt.Printf("✨ Go %s (%s) 解压成功!\n", release.Version, release.Arch)

	// 询问是否设置环境变量
	fmt.Print("\n🔧 是否立即将此版本设置为系统Go环境? [Y/n] ")
	var answer string
	fmt.Scanln(&answer)
	if answer == "" || strings.ToLower(answer) == "y" {
		if err := SetAsCurrentGo(targetDir); err != nil {
			return fmt.Errorf("❌ 设置环境变量失败: %v", err)
		}
		fmt.Printf("✅ 环境变量设置成功\n")
		fmt.Printf("⚠️ 注意：某些程序可能需要重启才能识别新的环境变量：\n")
		fmt.Printf("   • 终端 (PowerShell, CMD 等)\n")
		fmt.Printf("   • 编辑器 (VSCode, IntelliJ IDEA 等)\n")
		fmt.Printf("   • 其他使用Go环境的应用\n")
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
	Progress *DownloadProgress
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Writer.Write(p)
	if err != nil {
		return n, err
	}

	pw.Progress.Downloaded += int64(n)
	pw.Progress.showProgress()
	return n, nil
}

// showProgress 显示下载进度
func (p *DownloadProgress) showProgress() {
	percent := float64(p.Downloaded) / float64(p.Total) * 100
	elapsed := time.Since(p.StartTime).Seconds()
	speed := float64(p.Downloaded) / elapsed / 1024 / 1024 // MB/s

	// 计算进度条
	completed := int(float64(progressWidth) * float64(p.Downloaded) / float64(p.Total))
	bar := strings.Repeat(progressChar, completed) + strings.Repeat(emptyChar, progressWidth-completed)

	// 计算剩余时间
	var eta string
	if speed > 0 {
		remainingBytes := p.Total - p.Downloaded
		remainingSeconds := float64(remainingBytes) / (speed * 1024 * 1024)
		eta = fmt.Sprintf("%.0fs", remainingSeconds)
	} else {
		eta = "计算中..."
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

// unzip 解压zip文件
func unzip(zipFile string, destDir string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer r.Close()

	// 首先创建目标目录
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	for _, f := range r.File {
		// 去除 "go/" 前缀
		name := strings.TrimPrefix(f.Name, "go/")
		if name == "" {
			continue
		}

		path := filepath.Join(destDir, name)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			continue
		}

		// 确保父目录存在
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
