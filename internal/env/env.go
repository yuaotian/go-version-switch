package env

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/registry"
)

const (
	gorootEnvKey = "GOROOT"
	pathEnvKey   = "PATH"
)

// UpdateGoRoot 更新GOROOT环境变量
func UpdateGoRoot(newPath string) error {
	// 打开系统环境变量注册表
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Session Manager\Environment`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 设置GOROOT
	if err := key.SetStringValue(gorootEnvKey, newPath); err != nil {
		return fmt.Errorf("设置GOROOT失败: %v", err)
	}

	// 通知系统环境变量已更新
	return broadcastEnvironmentChange()
}

// UpdatePath 更新PATH环境变量
func UpdatePath(goRoot string) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `System\CurrentControlSet\Control\Session Manager\Environment`, registry.ALL_ACCESS)
	if err != nil {
		return fmt.Errorf("打开注册表失败: %v", err)
	}
	defer key.Close()

	// 获取当前PATH
	currentPath, _, err := key.GetStringValue(pathEnvKey)
	if err != nil {
		return fmt.Errorf("获取PATH失败: %v", err)
	}

	// 移除旧的Go路径
	paths := strings.Split(currentPath, ";")
	newPaths := make([]string, 0)
	for _, path := range paths {
		if !strings.Contains(strings.ToLower(path), "\\go\\bin") {
			newPaths = append(newPaths, path)
		}
	}

	// 添加新的Go路径
	goBinPath := filepath.Join(goRoot, "bin")
	newPaths = append(newPaths, goBinPath)

	// 更新PATH
	newPath := strings.Join(newPaths, ";")
	if err := key.SetStringValue(pathEnvKey, newPath); err != nil {
		return fmt.Errorf("更新PATH失败: %v", err)
	}

	return broadcastEnvironmentChange()
}

// BackupEnv 备份当前环境变量
func BackupEnv() error {
	// 获取当前环境变量
	goroot := os.Getenv("GOROOT")
	path := os.Getenv("PATH")

	// 创建备份数据
	backup := map[string]string{
		"GOROOT": goroot,
		"PATH":   path,
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(backup, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化环境变量失败: %v", err)
	}

	// 生成备份文件名
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join("data", "backup", fmt.Sprintf("env_backup_%s.json", timestamp))

	// 确保备份目录存在
	backupDir := filepath.Dir(backupFile)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(backupFile, data, 0644); err != nil {
		return fmt.Errorf("写入备份文件失败: %v", err)
	}

	return nil
}

// RestoreEnv 从备份恢复环境变量
func RestoreEnv(backupFile string) error {
	content, err := os.ReadFile(backupFile)
	if err != nil {
		return fmt.Errorf("读取备份文件失败: %v", err)
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case gorootEnvKey:
			if err := UpdateGoRoot(value); err != nil {
				return err
			}
		case pathEnvKey:
			if err := UpdatePath(value); err != nil {
				return err
			}
		}
	}

	return nil
}

// broadcastEnvironmentChange 通知系统环境变量已更新
func broadcastEnvironmentChange() error {
	// TODO: 实现通知系统环境变量更新的逻辑
	// 这里需要使用Windows API发送WM_SETTINGCHANGE消息
	return nil
}
