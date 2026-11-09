package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	goDownloadURL = "https://go.dev/dl/"
	// 官方版本索引：include=all 才会返回全部历史版本（默认只给最近两个大版本）
	goVersionAPI = "https://go.dev/dl/?mode=json&include=all"
)

// goReleaseEntry 官方 JSON 索引的一个版本
type goReleaseEntry struct {
	Version string          `json:"version"`
	Stable  bool            `json:"stable"`
	Files   []goReleaseFile `json:"files"`
}

// goReleaseFile 官方 JSON 索引里的单个文件
type goReleaseFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

// FetchVersions 获取可用的Go版本列表。
// 优先走官方 JSON 索引（结构稳定，不会因下载页改版而挂），失败再退回解析 HTML（老实现）。
func FetchVersions() ([]*GoRelease, error) {
	fmt.Println("正在从官网获取版本列表...")

	releases, err := fetchVersionsFromAPI()
	if err == nil && len(releases) > 0 {
		return releases, nil
	}
	fmt.Printf("⚠️ 官方 JSON 索引不可用（%v），回退到解析下载页...\n", err)

	return fetchVersionsFromHTML()
}

// fetchVersionsFromAPI 用 https://go.dev/dl/?mode=json&include=all 获取全部版本
func fetchVersionsFromAPI() ([]*GoRelease, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(goVersionAPI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var entries []goReleaseEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, err
	}

	releases := make([]*GoRelease, 0, len(entries)*2)
	for _, entry := range entries {
		version := strings.TrimPrefix(entry.Version, "go")
		for _, f := range entry.Files {
			if f.OS != "windows" || f.Kind != "archive" {
				continue
			}
			arch := normalizeReleaseArch(f.Arch)
			if arch == "" {
				continue
			}
			releases = append(releases, &GoRelease{
				Version:     version,
				Kind:        "Archive",
				OS:          "Windows",
				Arch:        arch,
				Size:        fmt.Sprintf("%dMB", f.Size/1024/1024),
				SHA256:      f.SHA256,
				DownloadURL: goDownloadURL + f.Filename,
			})
		}
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("JSON 索引里没有解析到 Windows 版本")
	}

	fmt.Printf("解析到 %d 个Windows版本（官方 JSON 索引）\n", len(releases))
	return releases, nil
}

// normalizeReleaseArch 把官方索引里的架构名统一成内部命名
func normalizeReleaseArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "386":
		return "x86"
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	default:
		return ""
	}
}

// fetchVersionsFromHTML 解析 go.dev/dl 下载页（JSON 接口失败时的兜底）
func fetchVersionsFromHTML() ([]*GoRelease, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(goDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("获取版本列表失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	return parseVersions(string(body))
}

// parseVersions 解析HTML页面获取版本信息
func parseVersions(html string) ([]*GoRelease, error) {
	var releases []*GoRelease

	// 使用正则表达式匹配所有版本行
	rowRegex := regexp.MustCompile(`<tr[^>]*>\s*<td[^>]*><a[^>]*href="([^"]+)"[^>]*>([^<]+)</a></td>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*>([^<]+)</td>\s*<td[^>]*><tt>([a-f0-9]{64})</tt></td>`)
	matches := rowRegex.FindAllStringSubmatch(html, -1)

	fmt.Printf("找到 %d 个版本条目\n", len(matches))

	for _, match := range matches {
		if len(match) < 8 {
			continue
		}

		downloadURL := match[1] // 下载链接
		filename := match[2]    // 文件名
		kind := match[3]        // 类型 (Archive/Installer)
		os := match[4]          // 操作系统
		var arch string
		if strings.Contains(match[5], "x86-64") {
			arch = "amd64"
		} else {
			arch = match[5]
		}
		size := match[6]   // 大小
		sha256 := match[7] // SHA256

		// 解析版本号
		versionRegex := regexp.MustCompile(`go(\d+\.\d+\.\d+)`)
		versionMatch := versionRegex.FindStringSubmatch(filename)
		if len(versionMatch) < 2 {
			continue
		}
		version := versionMatch[1]

		// 创建版本信息对象
		release := &GoRelease{
			Version:     version,
			Kind:        strings.TrimSpace(kind),
			OS:          strings.TrimSpace(os),
			Arch:        strings.TrimSpace(arch),
			Size:        strings.TrimSpace(size),
			SHA256:      sha256,
			DownloadURL: "https://go.dev" + downloadURL,
		}

		// 只添加 Windows 的 Archive 版本
		if release.OS == "Windows" && release.Kind == "Archive" {
			// 标准化架构名称
			switch {
			case strings.Contains(strings.ToLower(release.Arch), "386"):
				release.Arch = "x86"
			case strings.Contains(strings.ToLower(release.Arch), "amd64"):
				release.Arch = "amd64"
			case strings.Contains(strings.ToLower(release.Arch), "arm64"):
				release.Arch = "arm64"
			case strings.Contains(strings.ToLower(release.Arch), "arm"):
				release.Arch = "arm"
			}
			releases = append(releases, release)
		}
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("未找到可用的Windows版本")
	}

	fmt.Printf("解析到 %d 个Windows版本\n", len(releases))
	return releases, nil
}

// releaseKey 版本条目的唯一键（版本 + 架构 + 系统）
func releaseKey(r *GoRelease) string {
	return r.Version + "|" + strings.ToLower(r.Arch) + "|" + strings.ToLower(r.OS)
}

// MergeReleases 合并新旧版本列表：新数据覆盖同键旧数据，其余历史版本保留。
// 官方接口如果只返回最近几个版本，也不会把缓存里的历史版本冲掉。
func MergeReleases(old, fresh []*GoRelease) []*GoRelease {
	if len(old) == 0 {
		return fresh
	}

	index := make(map[string]int, len(old)+len(fresh))
	merged := make([]*GoRelease, 0, len(old)+len(fresh))
	for _, r := range old {
		index[releaseKey(r)] = len(merged)
		merged = append(merged, r)
	}
	for _, r := range fresh {
		if i, ok := index[releaseKey(r)]; ok {
			merged[i] = r
			continue
		}
		index[releaseKey(r)] = len(merged)
		merged = append(merged, r)
	}
	return merged
}

// SaveVersionsCache 保存版本信息到缓存（先写临时文件再原子替换，避免写一半崩掉留下坏 JSON）
func SaveVersionsCache(releases []*GoRelease, cacheFile string) error {
	data, err := json.MarshalIndent(releases, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化版本信息失败: %v", err)
	}

	tmpFile := cacheFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("写入版本缓存失败: %v", err)
	}
	if err := os.Rename(tmpFile, cacheFile); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("替换版本缓存失败: %v", err)
	}
	return nil
}

// LoadVersionsCache 从缓存加载版本信息
func LoadVersionsCache(cacheFile string) ([]*GoRelease, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var releases []*GoRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, err
	}

	// 标准化架构名称
	for _, release := range releases {
		switch {
		case strings.Contains(strings.ToLower(release.Arch), "386"):
			release.Arch = "x86"
		case strings.Contains(strings.ToLower(release.Arch), "amd64"):
			release.Arch = "amd64"
		case strings.Contains(strings.ToLower(release.Arch), "arm64"):
			release.Arch = "arm64"
		case strings.Contains(strings.ToLower(release.Arch), "arm"):
			release.Arch = "arm"
		}
	}

	return releases, nil
}
