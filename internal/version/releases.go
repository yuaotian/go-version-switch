package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

const (
	goDownloadURL = "https://go.dev/dl/"
)

// FetchVersions 获取可用的Go版本列表
func FetchVersions() ([]*GoRelease, error) {
	fmt.Println("正在从官网获取版本列表...")

	// 获取下载页面内容
	resp, err := http.Get(goDownloadURL)
	if err != nil {
		return nil, fmt.Errorf("获取版本列表失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应内容失败: %v", err)
	}

	htmlContent := string(body)

	return parseVersions(htmlContent)
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
		arch := match[5]        // 架构
		size := match[6]        // 大小
		sha256 := match[7]      // SHA256

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
				release.Arch = "x86-64"
			case strings.Contains(strings.ToLower(release.Arch), "arm64"):
				release.Arch = "ARM64"
			case strings.Contains(strings.ToLower(release.Arch), "arm"):
				release.Arch = "ARM"
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

// GetLatestVersion 获取最新的稳定版本
func GetLatestVersion() (string, error) {
	releases, err := FetchVersions()
	if err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("未找到可用版本")
	}

	return releases[0].Version, nil
}

// GetVersionInfo 获取指定版本的详细信息
func GetVersionInfo(version string) (*GoRelease, error) {
	releases, err := FetchVersions()
	if err != nil {
		return nil, err
	}

	version = ParseVersion(version)
	for _, release := range releases {
		if release.Version == version {
			return release, nil
		}
	}

	return nil, fmt.Errorf("未找到版本 %s", version)
}

// SaveVersionsCache 保存版本信息到缓存
func SaveVersionsCache(releases []*GoRelease, cacheFile string) error {
	data, err := json.MarshalIndent(releases, "", "    ")
	if err != nil {
		return fmt.Errorf("序列化版本信息失败: %v", err)
	}

	return os.WriteFile(cacheFile, data, 0644)
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
			release.Arch = "x86-64"
		case strings.Contains(strings.ToLower(release.Arch), "arm64"):
			release.Arch = "ARM64"
		case strings.Contains(strings.ToLower(release.Arch), "arm"):
			release.Arch = "ARM"
		}
	}

	return releases, nil
}
