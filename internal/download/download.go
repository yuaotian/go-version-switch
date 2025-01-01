package download

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"go-bits-switch/internal/version"
)

// DownloadInfo 下载信息
type DownloadInfo struct {
	Version  string // 版本号
	URL      string // 下载地址
	SavePath string // 保存路径
}

// DownloadVersion 下载指定版本
func DownloadVersion(info *DownloadInfo) error {
	// 创建保存目录
	saveDir := filepath.Dir(info.SavePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("创建保存目录失败: %v", err)
	}

	// 创建临时文件
	tmpFile := info.SavePath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %v", err)
	}
	defer out.Close()

	// 发起HTTP请求
	resp, err := http.Get(info.URL)
	if err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	// 创建进度显示
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		return fmt.Errorf("保存文件失败: %v", err)
	}

	// 重命名临时文件
	if err := os.Rename(tmpFile, info.SavePath); err != nil {
		return fmt.Errorf("重命名文件失败: %v", err)
	}

	return nil
}

// VerifyChecksum 验证文件校验和
func VerifyChecksum(filePath string, expectedHash string) error {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	// 计算SHA256
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("计算校验和失败: %v", err)
	}

	// 比较校验和
	actualHash := hex.EncodeToString(hash.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("校验和不匹配:\n期望: %s\n实际: %s", expectedHash, actualHash)
	}

	return nil
}

// WriteCounter 用于显示下载进度
type WriteCounter struct {
	Total uint64
}

// Write 实现io.Writer接口
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

// PrintProgress 打印下载进度
func (wc *WriteCounter) PrintProgress() {
	fmt.Printf("\r下载进度: %.2f MB", float64(wc.Total)/1024/1024)
}

// DownloadAndInstall 下载并安装指定版本
func DownloadAndInstall(info *version.GoRelease, baseDir string) error {
	// 创建下载信息
	downloadInfo := &DownloadInfo{
		Version:  info.Version,
		URL:      info.DownloadURL,
		SavePath: filepath.Join(baseDir, fmt.Sprintf("go%s", info.Version)),
	}

	// 下载版本
	if err := DownloadVersion(downloadInfo); err != nil {
		return fmt.Errorf("下载失败: %v", err)
	}

	// 验证校验和
	if err := VerifyChecksum(downloadInfo.SavePath, info.SHA256); err != nil {
		return fmt.Errorf("校验和验证失败: %v", err)
	}

	return nil
}
