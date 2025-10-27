package version

import "time"

// ReleaseInfo GitHub Release 信息
type ReleaseInfo struct {
	TagName     string  `json:"tag_name"`
	Name        string  `json:"name"`
	Body        string  `json:"body"`
	Assets      []Asset `json:"assets"`
	HTMLURL     string  `json:"html_url"`
	PublishedAt string  `json:"published_at"`
}

// Asset Release 资源文件
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// 通用常量（两种构建模式都可用）
const (
	githubAPIURL     = "https://api.github.com/repos/YangQing-Lin/cc-switch-cli/releases/latest"
	githubReleaseURL = "https://github.com/YangQing-Lin/cc-switch-cli/releases"
	httpTimeout      = 10 * time.Second
)
