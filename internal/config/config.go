package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 表示工具的配置信息
type Config struct {
	BaseDir        string            `json:"base_dir"`        // Go版本安装的基础目录
	CurrentVersion string            `json:"current_version"` // 当前使用的Go版本
	Versions       map[string]string `json:"versions"`        // 已安装的版本映射 version -> path
}

var (
	// 获取程序当前目录
	execDir, _        = os.Executable()
	dataDir           = filepath.Join(filepath.Dir(execDir), "data")
	defaultConfigPath = filepath.Join(dataDir, "config", "config.json")
)

// LoadConfig 加载配置文件
func LoadConfig() (*Config, error) {
	// 确保配置目录存在
	configDir := filepath.Dir(defaultConfigPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 尝试读取配置文件
	data, err := os.ReadFile(defaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果配置文件不存在，创建默认配置
			return createDefaultConfig()
		}
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config) error {
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(defaultConfigPath, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// createDefaultConfig 创建默认配置
func createDefaultConfig() (*Config, error) {
	// 创建所有必要的目录
	goVersionDir := filepath.Join(dataDir, "go-version")
	downloadDir := filepath.Join(dataDir, "down")

	// 创建目录
	dirs := []string{
		filepath.Join(dataDir, "config"),
		downloadDir,
		goVersionDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建目录失败 %s: %v", dir, err)
		}
	}

	config := &Config{
		BaseDir:  goVersionDir,
		Versions: make(map[string]string),
	}

	// 保存默认配置
	if err := SaveConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// AddVersion 添加新版本到配置
func (c *Config) AddVersion(version, path string) error {
	// 确保路径使用正确的目录名称
	dirName := filepath.Base(path)
	if !strings.HasPrefix(dirName, "go-version-bits-") {
		return fmt.Errorf("无效的版本目录名称: %s", dirName)
	}

	c.Versions[version] = path
	return SaveConfig(c)
}

// RemoveVersion 从配置中移除版本
func (c *Config) RemoveVersion(version string) error {
	delete(c.Versions, version)
	return SaveConfig(c)
}

// SetCurrentVersion 设置当前使用的版本
func (c *Config) SetCurrentVersion(version string) error {
	if _, exists := c.Versions[version]; !exists {
		return fmt.Errorf("版本 %s 未安装", version)
	}
	c.CurrentVersion = version
	return SaveConfig(c)
}

// GetDownloadDir 获取下载目录
func GetDownloadDir() string {
	return filepath.Join(dataDir, "down")
}

// GetVersionDir 根据版本号获取安装目录
func GetVersionDir(version string) string {
	return filepath.Join(dataDir, "go-version", fmt.Sprintf("go-version-bits-%s", version))
}

// GetVersionPath 获取指定版本的安装路径
func (c *Config) GetVersionPath(version string) string {
	if path, exists := c.Versions[version]; exists {
		return path
	}
	return ""
}
