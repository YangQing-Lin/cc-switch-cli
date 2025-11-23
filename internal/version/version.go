package version

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CleanupOldUpdateDirs 清理 /tmp 目录下的历史更新临时目录
// 静默执行，失败不报错
func CleanupOldUpdateDirs() {
	tmpDir := os.TempDir()
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return // 静默失败
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 匹配 ccs-update-* 和 ccs-install-* 模式
		if strings.HasPrefix(name, "ccs-update-") || strings.HasPrefix(name, "ccs-install-") {
			fullPath := filepath.Join(tmpDir, name)
			_ = os.RemoveAll(fullPath) // 静默删除，忽略错误
		}
	}
}

// Version 当前版本
const Version = "1.8.9"

// BuildDate 构建日期（由编译时注入）
var BuildDate = "unknown"

// GitCommit Git 提交哈希（由编译时注入）
var GitCommit = "unknown"

const (
	githubAPIURL     = "https://api.github.com/repos/YangQing-Lin/cc-switch-cli/releases/latest"
	githubReleaseURL = "https://github.com/YangQing-Lin/cc-switch-cli/releases"
	httpTimeout      = 10 * time.Second
)

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

// GetVersion 获取版本信息
func GetVersion() string {
	return Version
}

// GetBuildDate 获取构建日期
func GetBuildDate() string {
	return BuildDate
}

// GetGitCommit 获取 Git 提交哈希
func GetGitCommit() string {
	return GitCommit
}

// CheckForUpdate 检查是否有新版本
func CheckForUpdate() (*ReleaseInfo, bool, error) {
	client := &http.Client{Timeout: httpTimeout}

	req, err := http.NewRequest("GET", githubAPIURL, nil)
	if err != nil {
		return nil, false, fmt.Errorf("创建请求失败: %w", err)
	}

	// 添加 User-Agent 避免 GitHub API 限制
	req.Header.Set("User-Agent", "cc-switch-cli/"+Version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("GitHub API 返回错误: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("读取响应失败: %w", err)
	}

	var release ReleaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, false, fmt.Errorf("解析响应失败: %w", err)
	}

	// 比较版本：移除 v 前缀后比较
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersion := strings.TrimPrefix(Version, "v")

	hasUpdate := latestVersion != currentVersion

	return &release, hasUpdate, nil
}

// UpdateError 自定义更新错误类型，包含失败原因和下载地址
type UpdateError struct {
	Reason      string
	DownloadURL string
}

func (e *UpdateError) Error() string {
	return fmt.Sprintf("%s\n\n推荐手动下载：\n%s", e.Reason, e.DownloadURL)
}

// DownloadUpdate 下载并安装更新
func DownloadUpdate(release *ReleaseInfo) error {
	// 确定当前平台的压缩包文件名
	archiveName := getArchiveNameForPlatform(release.TagName)

	// 查找匹配的资源
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == archiveName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return &UpdateError{
			Reason:      fmt.Sprintf("未找到适合当前平台 (%s-%s) 的安装包", runtime.GOOS, runtime.GOARCH),
			DownloadURL: githubReleaseURL,
		}
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "ccs-update-*")
	if err != nil {
		return &UpdateError{
			Reason:      fmt.Sprintf("创建临时目录失败: %v (可能磁盘空间不足或权限不足)", err),
			DownloadURL: githubReleaseURL,
		}
	}
	defer os.RemoveAll(tmpDir)

	// 下载压缩包到临时目录
	archivePath := filepath.Join(tmpDir, archiveName)
	if err := downloadFile(archivePath, downloadURL); err != nil {
		return &UpdateError{
			Reason:      fmt.Sprintf("下载失败: %v (请检查网络连接)", err),
			DownloadURL: githubReleaseURL,
		}
	}

	// 安装二进制文件
	return InstallBinary(archivePath, false)
}

// InstallBinary 从本地文件安装二进制文件（支持压缩包或裸二进制）
// sourcePath: 源文件路径（.tar.gz, .zip 或裸二进制）
// skipPlatformCheck: 是否跳过平台验证
func InstallBinary(sourcePath string, skipPlatformCheck bool) error {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return &UpdateError{
			Reason:      fmt.Sprintf("获取可执行文件路径失败: %v", err),
			DownloadURL: githubReleaseURL,
		}
	}

	// 解析符号链接
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return &UpdateError{
			Reason:      fmt.Sprintf("解析符号链接失败: %v", err),
			DownloadURL: githubReleaseURL,
		}
	}

	// 创建临时目录用于解压
	tmpDir, err := os.MkdirTemp("", "ccs-install-*")
	if err != nil {
		return &UpdateError{
			Reason:      fmt.Sprintf("创建临时目录失败: %v (可能磁盘空间不足或权限不足)", err),
			DownloadURL: githubReleaseURL,
		}
	}
	defer os.RemoveAll(tmpDir)

	var binaryPath string

	// 判断是压缩包还是裸二进制
	if isArchive(sourcePath) {
		// 平台验证（从文件名推断）
		if !skipPlatformCheck {
			if err := validatePlatformFromFilename(sourcePath); err != nil {
				return &UpdateError{
					Reason:      err.Error(),
					DownloadURL: githubReleaseURL,
				}
			}
		}

		// 解压压缩包
		binaryPath, err = extractBinary(sourcePath, tmpDir)
		if err != nil {
			return &UpdateError{
				Reason:      fmt.Sprintf("解压失败: %v (压缩包可能损坏)", err),
				DownloadURL: githubReleaseURL,
			}
		}
	} else {
		// 裸二进制文件，直接复制到临时目录
		binaryName := "ccs"
		if runtime.GOOS == "windows" {
			binaryName = "ccs.exe"
		}
		binaryPath = filepath.Join(tmpDir, binaryName)

		// 复制文件
		if err := copyFile(sourcePath, binaryPath); err != nil {
			return &UpdateError{
				Reason:      fmt.Sprintf("复制二进制文件失败: %v", err),
				DownloadURL: githubReleaseURL,
			}
		}
	}

	// 设置可执行权限 (Unix-like 系统)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(binaryPath, 0755); err != nil {
			return &UpdateError{
				Reason:      fmt.Sprintf("设置可执行权限失败: %v", err),
				DownloadURL: githubReleaseURL,
			}
		}
	}

	// 备份当前版本
	backupPath := exePath + ".old"
	if err := os.Rename(exePath, backupPath); err != nil {
		return &UpdateError{
			Reason:      fmt.Sprintf("备份当前版本失败: %v (可能程序正在运行或权限不足)", err),
			DownloadURL: githubReleaseURL,
		}
	}

	// 移动新版本到位
	if err := os.Rename(binaryPath, exePath); err != nil {
		// 恢复备份
		os.Rename(backupPath, exePath)
		return &UpdateError{
			Reason:      fmt.Sprintf("安装新版本失败: %v (已恢复原版本)", err),
			DownloadURL: githubReleaseURL,
		}
	}

	// 删除备份
	os.Remove(backupPath)

	return nil
}

// getArchiveNameForPlatform 获取当前平台的压缩包文件名
// 格式: cc-switch-cli-1.2.0-linux-amd64.tar.gz
func getArchiveNameForPlatform(tagName string) string {
	// 移除 v 前缀（tagName 格式为 v1.2.0）
	version := strings.TrimPrefix(tagName, "v")
	osName := runtime.GOOS
	arch := runtime.GOARCH

	if runtime.GOOS == "windows" {
		return fmt.Sprintf("cc-switch-cli-%s-%s-%s.zip", version, osName, arch)
	}
	return fmt.Sprintf("cc-switch-cli-%s-%s-%s.tar.gz", version, osName, arch)
}

// extractBinary 从压缩包中提取二进制文件
func extractBinary(archivePath, destDir string) (string, error) {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	} else if strings.HasSuffix(archivePath, ".tar.gz") {
		return extractTarGz(archivePath, destDir)
	}
	return "", fmt.Errorf("不支持的压缩格式: %s", archivePath)
}

// extractTarGz 解压 tar.gz 文件
func extractTarGz(archivePath, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	binaryName := "ccs"

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// 只提取二进制文件
		if filepath.Base(header.Name) == binaryName {
			targetPath := filepath.Join(destDir, binaryName)
			outFile, err := os.Create(targetPath)
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, tr); err != nil {
				return "", err
			}

			return targetPath, nil
		}
	}

	return "", fmt.Errorf("压缩包中未找到二进制文件: %s", binaryName)
}

// extractZip 解压 zip 文件
func extractZip(archivePath, destDir string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	binaryName := "ccs.exe"

	for _, f := range r.File {
		if filepath.Base(f.Name) == binaryName {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			targetPath := filepath.Join(destDir, binaryName)
			outFile, err := os.Create(targetPath)
			if err != nil {
				return "", err
			}
			defer outFile.Close()

			if _, err := io.Copy(outFile, rc); err != nil {
				return "", err
			}

			return targetPath, nil
		}
	}

	return "", fmt.Errorf("压缩包中未找到二进制文件: %s", binaryName)
}

// downloadFile 下载文件
func downloadFile(filepath string, url string) error {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("下载请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败: HTTP %d", resp.StatusCode)
	}

	// 创建文件
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 写入数据
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// GetReleasePageURL 获取 Release 页面 URL
func GetReleasePageURL() string {
	return githubReleaseURL
}

// isArchive 判断文件是否为压缩包
func isArchive(filename string) bool {
	return strings.HasSuffix(filename, ".tar.gz") || strings.HasSuffix(filename, ".zip")
}

// validatePlatformFromFilename 从文件名验证平台信息
// 文件名格式: cc-switch-cli-1.7.0-linux-amd64.tar.gz
func validatePlatformFromFilename(filename string) error {
	basename := filepath.Base(filename)

	// 移除扩展名
	basename = strings.TrimSuffix(basename, ".tar.gz")
	basename = strings.TrimSuffix(basename, ".zip")

	// 分割文件名: cc-switch-cli-1.7.0-linux-amd64
	parts := strings.Split(basename, "-")
	if len(parts) < 2 {
		return fmt.Errorf("无法从文件名推断平台信息: %s\n提示: 使用 --force 跳过平台验证", basename)
	}

	// 提取 OS 和 ARCH（最后两个部分）
	fileOS := parts[len(parts)-2]
	fileArch := parts[len(parts)-1]

	// 验证平台
	if fileOS != runtime.GOOS {
		return fmt.Errorf("平台不匹配: 文件是为 %s 构建的，但当前系统是 %s\n提示: 使用 --force 跳过平台验证", fileOS, runtime.GOOS)
	}

	if fileArch != runtime.GOARCH {
		return fmt.Errorf("架构不匹配: 文件是为 %s 构建的，但当前系统是 %s\n提示: 使用 --force 跳过平台验证", fileArch, runtime.GOARCH)
	}

	return nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}
