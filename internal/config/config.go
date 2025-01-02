package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config 表示工具的配置信息
type Config struct {
	BaseDir        string            `json:"base_dir"`        // Go版本安装的基础目录
	CurrentVersion string            `json:"current_version"` // 当前使用的Go版本
	Versions       map[string]string `json:"versions"`        // 已安装的版本映射 version -> path
	LastUpdate     CustomTime        `json:"last_update"`     // 上次更新时间
}

// CustomTime 自定义时间类型，用于格式化 JSON 输出
type CustomTime struct {
	time.Time
}

// MarshalJSON 自定义时间的 JSON 序列化方法
func (ct CustomTime) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, ct.Format("2006-01-02 15:04:05"))), nil
}

// UnmarshalJSON 自定义时间的 JSON 反序列化方法
func (ct *CustomTime) UnmarshalJSON(data []byte) error {
	// 去掉引号
	str := strings.Trim(string(data), `"`)
	// 尝试解析自定义格式
	t, err := time.Parse("2006-01-02 15:04:05", str)
	if err != nil {
		// 如果解析失败，尝试解析 ISO 格式
		t, err = time.Parse(time.RFC3339, str)
		if err != nil {
			return err
		}
	}
	ct.Time = t
	return nil
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
	configDir := filepath.Join(filepath.Dir(os.Args[0]), "data", "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %v", err)
	}

	configFile := filepath.Join(configDir, "config.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			// 如果配置文件不存在，创建默认配置
			defaultTime := time.Date(2024, 1, 1, 23, 59, 59, 0, time.Local)
			config := &Config{
				BaseDir:    filepath.Join(filepath.Dir(os.Args[0]), "data", "go-version"),
				Versions:   make(map[string]string),
				LastUpdate: CustomTime{Time: defaultTime},
			}
			return config, SaveConfig(config)
		}
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	// 确保版本映射已初始化
	if config.Versions == nil {
		config.Versions = make(map[string]string)
	}

	return &config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config) error {
	configFile := filepath.Join(filepath.Dir(os.Args[0]), "data", "config", "config.json")
	data, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := os.WriteFile(configFile, data, 0644); err != nil {
		return fmt.Errorf("保存配置文件失败: %v", err)
	}

	return nil
}

// AddVersion 添加新版本到配置
func (c *Config) AddVersion(version, path string) error {
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


