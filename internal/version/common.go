package version

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
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

// RenderProgressBar 渲染进度条（percent 会被夹到 [0,1]，避免负数 Repeat 导致 panic）
func (p *ProgressBar) RenderProgressBar(percent float64) string {
	if percent < 0 || math.IsNaN(percent) {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}
	completed := int(float64(p.Width) * percent)
	if completed < 0 {
		completed = 0
	}
	if completed > p.Width {
		completed = p.Width
	}
	return strings.Repeat(p.FilledChar, completed) +
		strings.Repeat(p.EmptyChar, p.Width-completed)
}

// AssumeYes 为 true 时跳过交互确认（-yes，用于脚本 / CI / 无人值守安装）
var AssumeYes bool

// confirm 统一的交互确认。AssumeYes 直接通过；无输入（管道、CI）按默认值 y 处理。
func confirm(prompt string) bool {
	if AssumeYes {
		return true
	}
	fmt.Print(prompt)
	var answer string
	_, _ = fmt.Scanln(&answer)
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
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
