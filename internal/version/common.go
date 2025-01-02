package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// ProgressBar 进度条配置
type ProgressBar struct {
	Width      int
	FilledChar string
	EmptyChar  string
}

// NewDefaultProgressBar 创建默认进度条
func NewDefaultProgressBar() *ProgressBar {
	return &ProgressBar{
		Width:      50,
		FilledChar: "█",
		EmptyChar:  "░",
	}
}

// RenderProgressBar 渲染进度条
func (p *ProgressBar) RenderProgressBar(percent float64) string {
	completed := int(float64(p.Width) * percent)
	return strings.Repeat(p.FilledChar, completed) +
		strings.Repeat(p.EmptyChar, p.Width-completed)
}

// FileVerifier 文件验证器
type FileVerifier struct {
	FilePath     string
	ExpectedHash string
}

// Verify 验证文件完整性
func (v *FileVerifier) Verify() error {
	file, err := os.Open(v.FilePath)
	if err != nil {
		return fmt.Errorf("打开文件失败: %v", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("计算文件哈希失败: %v", err)
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(actualHash, v.ExpectedHash) {
		return fmt.Errorf("文件校验失败\n期望值: %s\n实际值: %s",
			v.ExpectedHash, actualHash)
	}

	return nil
}
