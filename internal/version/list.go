package version

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// VersionList 版本列表信息
type VersionList struct {
	Versions       []*GoRelease      `json:"versions"`         // 所有可用版本
	LastUpdateTime time.Time         `json:"last_update_time"` // 上次更新时间
	InstalledPaths map[string]string `json:"installed_paths"`  // 已安装版本的路径
	CurrentVersion string            `json:"current_version"`  // 当前使用的版本
}

const (
	updateInterval = 7 * 24 * time.Hour // 默认7天更新一次
)

// GetVersionList 获取版本列表
func GetVersionList(baseDir string, forceUpdate bool) (*VersionList, error) {
	list := &VersionList{
		InstalledPaths: make(map[string]string),
	}

	// 获取当前版本
	current, err := GetCurrentVersion()
	if err == nil {
		list.CurrentVersion = current.Version
		list.InstalledPaths[current.Version] = current.Path
	}

	// 获取已安装版本
	versionDir := filepath.Join(baseDir, "go-version")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, fmt.Errorf("创建版本目录失败: %v", err)
	}

	// 遍历版本目录获取已安装版本
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		fmt.Printf("警告: 读取版本目录失败: %v\n", err)
	} else {
		for _, entry := range entries {
			if entry.IsDir() {
				dirName := entry.Name()
				if strings.HasPrefix(dirName, "go-") {
					// 从目录名解析版本号和架构
					parts := strings.Split(strings.TrimPrefix(dirName, "go-"), "-")
					if len(parts) >= 2 {
						version := parts[0]
						path := filepath.Join(versionDir, dirName)
						list.InstalledPaths[version] = path
					}
				}
			}
		}
	}

	// 确保配置目录存在
	configDir := filepath.Join(baseDir, "config")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("创建配置目录失败: %v", err)
	}

	// 获取缓存文件路径
	cacheFile := filepath.Join(configDir, "versions.json")
	needUpdate := forceUpdate

	// 如果缓存文件不存在，需要更新
	if !needUpdate {
		if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
			needUpdate = true
		}
	}

	// 如果不需要更新，尝试从缓存加载
	if !needUpdate {
		versions, err := LoadVersionsCache(cacheFile)
		if err == nil {
			list.Versions = versions
			list.LastUpdateTime = getFileModTime(cacheFile)
			return list, nil
		}
		// 如果加载缓存失败，需要更新
		needUpdate = true
	}

	// 需要更新时，从官网获取版本列表
	if needUpdate {
		fmt.Println("正在从官网获取版本列表...")
		versions, err := FetchVersions()
		if err != nil {
			return nil, fmt.Errorf("获取版本列表失败: %v", err)
		}

		list.Versions = versions
		list.LastUpdateTime = time.Now()

		// 保存到缓存
		if err := SaveVersionsCache(versions, cacheFile); err != nil {
			fmt.Printf("警告: 保存版本缓存失败: %v\n", err)
		}
	}

	// 对版本进行排序
	sort.Slice(list.Versions, func(i, j int) bool {
		return compareVersions(list.Versions[i].Version, list.Versions[j].Version) > 0
	})

	// 过滤只显示当前系统架构的版本
	filteredVersions := make([]*GoRelease, 0)
	currentArch := runtime.GOARCH
	// 添加当前系统架构的标记
	for _, v := range list.Versions {
		// 标记当前系统架构
		if strings.Contains(strings.ToLower(v.DownloadURL), currentArch) {
			v.IsCurrentArch = true
		}
		filteredVersions = append(filteredVersions, v)
	}
	list.Versions = filteredVersions

	return list, nil
}

// PrintVersionList 打印版本列表
func (l *VersionList) PrintVersionList() {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Go版本管理器 - 版本列表")
	fmt.Println(strings.Repeat("=", 80))

	// 打印更新时间
	fmt.Printf("📅 版本列表更新时间: %s\n", l.LastUpdateTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("📦 可用版本总数: %d\n", len(l.Versions))
	fmt.Printf("💿 已安装版本数: %d\n", len(l.InstalledPaths))
	fmt.Println(strings.Repeat("-", 80))

	// 打印当前版本详细信息
	if l.CurrentVersion != "" {
		fmt.Println("🎯 当前系统Go环境:")
		fmt.Printf("   • 版本: %s\n", l.CurrentVersion)
		if path, ok := l.InstalledPaths[l.CurrentVersion]; ok {
			fmt.Printf("   • 安装路径: %s\n", path)
		}
		// 获取GOPATH、GOROOT、GOARCH
		gopath := os.Getenv("GOPATH")
		goroot := os.Getenv("GOROOT")
		goarch := os.Getenv("GOARCH")
		if goroot != "" {
			fmt.Printf("   • GOROOT: %s\n", goroot)
		}
		if gopath != "" {
			fmt.Printf("   • GOPATH: %s\n", gopath)
		}
		if goarch != "" {
			fmt.Printf("   • GOARCH: %s\n", goarch)
		}
		fmt.Println(strings.Repeat("-", 80))
	}

	if len(l.Versions) == 0 {
		fmt.Println("⚠️ 未找到可用的Go版本")
		fmt.Println("请检查网络连接后重试，或使用 -update 参数强制更新版本列表")
		return
	}

	// 统计信息
	osCount := make(map[string]int)
	archCount := make(map[string]int)
	for _, v := range l.Versions {
		osCount[v.OS]++
		archCount[v.Arch]++
	}

	// 打印统计信息
	fmt.Println("📊 版本分布统计:")
	fmt.Println("操作系统分布:")
	for os, count := range osCount {
		icon := "🪟"
		if os == "Linux" {
			icon = "🐧"
		} else if os == "Darwin" {
			icon = "🍎"
		}
		fmt.Printf("   %s %s: %d 个版本\n", icon, os, count)
	}

	fmt.Println("架构分布:")
	for arch, count := range archCount {
		icon := "💻"
		if arch == "ARM64" || arch == "ARM" {
			icon = "📱"
		}
		fmt.Printf("   %s %s: %d 个版本\n", icon, arch, count)
	}
	fmt.Println(strings.Repeat("-", 80))

	// 打印版本列表表头
	fmt.Printf("%-12s %-10s %-20s %-8s %-15s %s\n",
		"版本号", "系统", "架构/位数", "大小", "状态", "校验和")
	fmt.Println(strings.Repeat("-", 85))

	// 创建版本分组映射
	versionGroups := make(map[string][]*GoRelease)
	for _, v := range l.Versions {
		versionGroups[v.Version] = append(versionGroups[v.Version], v)
	}

	// 获取所有版本号并排序
	versions := make([]string, 0, len(versionGroups))
	for version := range versionGroups {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool {
		v1Parts := strings.Split(versions[i], ".")
		v2Parts := strings.Split(versions[j], ".")

		// 确保两个版本号都有3个部分
		for len(v1Parts) < 3 {
			v1Parts = append(v1Parts, "0")
		}
		for len(v2Parts) < 3 {
			v2Parts = append(v2Parts, "0")
		}

		// 依次比较每个部分
		for k := 0; k < 3; k++ {
			num1, _ := strconv.Atoi(v1Parts[k])
			num2, _ := strconv.Atoi(v2Parts[k])

			if num1 != num2 {
				return num1 > num2 // 降序排列，新版本在前
			}
		}
		return false
	})

	// 按版本号分组输出
	for _, version := range versions {
		releases := versionGroups[version]
		sort.Slice(releases, func(i, j int) bool {
			archOrder := map[string]int{
				"x86-64": 1,
				"x86":    2,
				"ARM64":  3,
				"ARM":    4,
			}
			orderI := archOrder[releases[i].Arch]
			orderJ := archOrder[releases[j].Arch]
			if orderI == 0 {
				orderI = 99
			}
			if orderJ == 0 {
				orderJ = 99
			}
			return orderI < orderJ
		})

		for _, v := range releases {
			status := "可安装"
			if _, ok := l.InstalledPaths[v.Version]; ok {
				if v.Version == l.CurrentVersion {
					status = "当前版本 📍"
				} else {
					status = "已安装 ✓"
				}
			}

			osIcon := "🪟"
			if v.OS == "Linux" {
				osIcon = "🐧"
			} else if v.OS == "Darwin" {
				osIcon = "🍎"
			}

			var archIcon, archDisplay string
			switch v.Arch {
			case "x86":
				archIcon = "🖥️"
				archDisplay = "x86/32位"
			case "x86-64":
				archIcon = "💻"
				archDisplay = "x64/64位"
			case "ARM64":
				archIcon = "📱"
				archDisplay = "ARM/64位"
			case "ARM":
				archIcon = "📟"
				archDisplay = "ARM/32位"
			default:
				archIcon = "🔧"
				archDisplay = v.Arch
			}

			// 为当前使用的版本添加标记
			if v.Version == l.CurrentVersion && v.IsCurrentArch {
				archDisplay += " ✅"
			}

			fmt.Printf("%-12s %s%-8s %s%-20s %-8s %-15s %.8s...\n",
				v.Version,
				osIcon, v.OS,
				archIcon, archDisplay,
				v.Size,
				status,
				v.SHA256)
		}
	}
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("提示: 使用 'go-version-switch -install <版本号>' 安装指定版本")
	fmt.Println("      使用 'go-version-switch -use <版本号>' 切换到指定版本")
	fmt.Println("      使用 'go-version-switch -install <版本号> -arch <架构>' 安装指定架构的版本")
	fmt.Println("      使用 'go-version-switch -use <版本号> -arch <架构>' 切换到指定架构的版本")
	fmt.Println("架构选项: x86 (32位), x64 (64位), arm (32位), arm64 (64位)")
	fmt.Println(strings.Repeat("=", 80))
}

// getSortedVersions 获取排序后的版本号列表
func getSortedVersions(versionGroups map[string][]*GoRelease) []string {
	versions := make([]string, 0, len(versionGroups))
	for version := range versionGroups {
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})
	return versions
}

// shouldUpdateVersions 检查是否需要更新版本列表
func shouldUpdateVersions() bool {
	cached, err := loadVersionListCache()
	if err != nil {
		return true
	}
	return time.Since(cached.LastUpdateTime) > updateInterval
}

// getVersionCachePath 获取版本缓存文件路径
func getVersionCachePath() string {
	execPath, _ := os.Executable()
	return filepath.Join(filepath.Dir(execPath), "data", "config", "versions.json")
}

// saveVersionListCache 保存版本列表到缓存
func saveVersionListCache(list *VersionList) error {
	cacheFile := getVersionCachePath()
	return SaveVersionsCache(list.Versions, cacheFile)
}

// loadVersionListCache 从缓存加载版本列表
func loadVersionListCache() (*VersionList, error) {
	cacheFile := getVersionCachePath()
	versions, err := LoadVersionsCache(cacheFile)
	if err != nil {
		return nil, err
	}

	return &VersionList{
		Versions:       versions,
		LastUpdateTime: getFileModTime(cacheFile),
	}, nil
}

// getFileModTime 获取文件修改时间
func getFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// compareVersions 比较两个版本号
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// 补齐版本号长度
	for len(parts1) < 3 {
		parts1 = append(parts1, "0")
	}
	for len(parts2) < 3 {
		parts2 = append(parts2, "0")
	}

	// 逐段比较
	for i := 0; i < 3; i++ {
		if parts1[i] > parts2[i] {
			return 1
		}
		if parts1[i] < parts2[i] {
			return -1
		}
	}
	return 0
}
