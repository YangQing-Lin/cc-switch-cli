package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/YangQing-Lin/cc-switch-cli/internal/testutil"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

type errorReadCloser struct {
	err error
}

func (e errorReadCloser) Read(_ []byte) (int, error) {
	return 0, e.err
}

func (e errorReadCloser) Close() error {
	return nil
}

type timeoutError struct{}

func (timeoutError) Error() string {
	return "timeout"
}

func (timeoutError) Timeout() bool {
	return true
}

func (timeoutError) Temporary() bool {
	return true
}

func resetVersionHooks(t *testing.T) {
	t.Helper()
	origHTTPUpdate := httpClientForUpdate
	origHTTPDownload := httpClientForDownload
	origTempDir := tempDirFunc
	origReadDir := readDirFunc
	origRemoveAll := removeAllFunc
	origMkdirTemp := mkdirTempFunc
	origExecutable := executableFunc
	origEval := evalSymlinksFunc
	origChmod := chmodFunc
	origRename := renameFunc
	origCreate := createFileFunc

	t.Cleanup(func() {
		httpClientForUpdate = origHTTPUpdate
		httpClientForDownload = origHTTPDownload
		tempDirFunc = origTempDir
		readDirFunc = origReadDir
		removeAllFunc = origRemoveAll
		mkdirTempFunc = origMkdirTemp
		executableFunc = origExecutable
		evalSymlinksFunc = origEval
		chmodFunc = origChmod
		renameFunc = origRename
		createFileFunc = origCreate
	})
}

func nextVersion(current string) string {
	normalized := normalizeVersion(current)
	parts, ok := parseSemverParts(normalized)
	if !ok {
		return normalized + ".1"
	}
	parts[2]++
	return fmt.Sprintf("%d.%d.%d", parts[0], parts[1], parts[2])
}

func prevVersion(current string) string {
	normalized := normalizeVersion(current)
	parts, ok := parseSemverParts(normalized)
	if !ok {
		return normalized
	}
	switch {
	case parts[2] > 0:
		parts[2]--
	case parts[1] > 0:
		parts[1]--
		parts[2] = 0
	case parts[0] > 0:
		parts[0]--
		parts[1] = 0
		parts[2] = 0
	}
	return fmt.Sprintf("%d.%d.%d", parts[0], parts[1], parts[2])
}

func releaseJSON(t *testing.T, tagName string) string {
	t.Helper()
	data, err := json.Marshal(ReleaseInfo{TagName: tagName})
	if err != nil {
		t.Fatalf("生成 release JSON 失败: %v", err)
	}
	return string(data)
}

func fakeExecutable(t *testing.T, content string) string {
	t.Helper()
	exeName := "ccs"
	if runtime.GOOS == "windows" {
		exeName = "ccs.exe"
	}
	exePath := filepath.Join(t.TempDir(), exeName)
	if err := os.WriteFile(exePath, []byte(content), 0600); err != nil {
		t.Fatalf("创建可执行文件失败: %v", err)
	}
	executableFunc = func() (string, error) { return exePath, nil }
	evalSymlinksFunc = func(path string) (string, error) { return path, nil }
	return exePath
}

func createArchiveWithBinary(t *testing.T, format string, content []byte) string {
	t.Helper()
	binaryName := "ccs"
	if format == "zip" {
		binaryName = "ccs.exe"
	}
	return testutil.CreateTestArchive(t, format, map[string][]byte{
		binaryName:  content,
		"README.md": []byte("readme"),
	})
}

func renameWithCopy(oldPath, newPath string) error {
	if err := copyFile(oldPath, newPath); err != nil {
		return err
	}
	return os.Remove(oldPath)
}

func TestCleanupOldUpdateDirs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) func(t *testing.T)
	}{
		{
			name: "read dir error",
			setup: func(t *testing.T) func(t *testing.T) {
				resetVersionHooks(t)
				called := false
				tempDirFunc = func() string { return t.TempDir() }
				readDirFunc = func(string) ([]os.DirEntry, error) {
					return nil, errors.New("read error")
				}
				removeAllFunc = func(string) error {
					called = true
					return nil
				}
				return func(t *testing.T) {
					if called {
						t.Fatalf("readDir 失败时不应触发删除")
					}
				}
			},
		},
		{
			name: "remove matching dirs",
			setup: func(t *testing.T) func(t *testing.T) {
				resetVersionHooks(t)
				tempDir := t.TempDir()
				tempDirFunc = func() string { return tempDir }
				readDirFunc = os.ReadDir
				removeAllFunc = os.RemoveAll

				dirs := []string{
					filepath.Join(tempDir, "ccs-update-123"),
					filepath.Join(tempDir, "ccs-install-456"),
					filepath.Join(tempDir, "keep-dir"),
				}
				for _, dir := range dirs {
					if err := os.MkdirAll(dir, 0755); err != nil {
						t.Fatalf("创建目录失败: %v", err)
					}
				}
				filePath := filepath.Join(tempDir, "ccs-update-file")
				if err := os.WriteFile(filePath, []byte("noop"), 0600); err != nil {
					t.Fatalf("创建文件失败: %v", err)
				}

				return func(t *testing.T) {
					if _, err := os.Stat(dirs[0]); !os.IsNotExist(err) {
						t.Fatalf("ccs-update 目录未删除")
					}
					if _, err := os.Stat(dirs[1]); !os.IsNotExist(err) {
						t.Fatalf("ccs-install 目录未删除")
					}
					if _, err := os.Stat(dirs[2]); err != nil {
						t.Fatalf("非匹配目录被误删: %v", err)
					}
					if _, err := os.Stat(filePath); err != nil {
						t.Fatalf("非目录条目不应被删除: %v", err)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := tt.setup(t)
			CleanupOldUpdateDirs()
			check(t)
		})
	}
}

func TestVersionMetadata(t *testing.T) {
	tests := []struct {
		name     string
		got      string
		expected string
		nonEmpty bool
	}{
		{name: "GetVersion", got: GetVersion(), expected: Version, nonEmpty: true},
		{name: "GetBuildDate", got: GetBuildDate(), nonEmpty: true},
		{name: "GetGitCommit", got: GetGitCommit(), nonEmpty: true},
		{name: "GetReleasePageURL", got: GetReleasePageURL(), expected: githubReleaseURL, nonEmpty: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.nonEmpty && tt.got == "" {
				t.Fatalf("返回值不应为空")
			}
			if tt.expected != "" && tt.got != tt.expected {
				t.Fatalf("返回值不匹配: %s != %s", tt.got, tt.expected)
			}
		})
	}
}

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{name: "newer patch", latest: "1.2.4", current: "1.2.3", want: true},
		{name: "same version", latest: "1.2.3", current: "1.2.3", want: false},
		{name: "older minor", latest: "1.2.3", current: "1.3.0", want: false},
		{name: "v prefix", latest: "v2.0.0", current: "1.9.9", want: true},
		{name: "release newer than pre-release", latest: "1.2.3", current: "1.2.3-beta.1", want: true},
		{name: "pre-release not newer than release", latest: "1.2.3-beta.1", current: "1.2.3", want: false},
		{name: "pre-release numeric compare", latest: "1.2.3-beta.2", current: "1.2.3-beta.1", want: true},
		{name: "pre-release lexical compare", latest: "1.2.3-beta", current: "1.2.3-alpha", want: true},
		{name: "pre-release longer newer", latest: "1.2.3-alpha.1", current: "1.2.3-alpha", want: true},
		{name: "pre-release numeric lower than non-numeric", latest: "1.2.3-alpha", current: "1.2.3-1", want: true},
		{name: "build metadata ignored", latest: "1.2.3+build.2", current: "1.2.3+build.1", want: false},
		{name: "invalid fallback", latest: "invalid", current: "1.0.0", want: true},
		{name: "invalid equal", latest: "invalid", current: "invalid", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
				t.Fatalf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

func TestCheckForUpdate(t *testing.T) {
	current := normalizeVersion(Version)
	newer := nextVersion(current)
	older := prevVersion(current)

	tests := []struct {
		name        string
		client      func(t *testing.T) *http.Client
		wantUpdate  bool
		wantErr     bool
		errContains string
	}{
		{
			name: "update available",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					githubAPIURL: {
						StatusCode: http.StatusOK,
						Body:       releaseJSON(t, "v"+newer),
					},
				})
			},
			wantUpdate: true,
		},
		{
			name: "no update for same",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					githubAPIURL: {
						StatusCode: http.StatusOK,
						Body:       releaseJSON(t, "v"+current),
					},
				})
			},
			wantUpdate: false,
		},
		{
			name: "no update for older",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					githubAPIURL: {
						StatusCode: http.StatusOK,
						Body:       releaseJSON(t, "v"+older),
					},
				})
			},
			wantUpdate: false,
		},
		{
			name: "api status error",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					githubAPIURL: {
						StatusCode: http.StatusInternalServerError,
						Body:       "oops",
					},
				})
			},
			wantErr:     true,
			errContains: "GitHub API 返回错误",
		},
		{
			name: "invalid json",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					githubAPIURL: {
						StatusCode: http.StatusOK,
						Body:       "not-json",
					},
				})
			},
			wantErr:     true,
			errContains: "解析响应失败",
		},
		{
			name: "read error",
			client: func(t *testing.T) *http.Client {
				return &http.Client{
					Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       errorReadCloser{err: errors.New("read fail")},
							Header:     make(http.Header),
							Request:    req,
						}, nil
					}),
				}
			},
			wantErr:     true,
			errContains: "读取响应失败",
		},
		{
			name: "timeout error",
			client: func(t *testing.T) *http.Client {
				return &http.Client{
					Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
						return nil, timeoutError{}
					}),
				}
			},
			wantErr:     true,
			errContains: "请求失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetVersionHooks(t)
			httpClientForUpdate = func() *http.Client { return tt.client(t) }

			_, gotUpdate, err := CheckForUpdate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不匹配: err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("错误信息不匹配: %v", err)
				}
				return
			}
			if gotUpdate != tt.wantUpdate {
				t.Fatalf("更新判断不匹配: %v != %v", gotUpdate, tt.wantUpdate)
			}
		})
	}
}

func TestGetArchiveNameForPlatform(t *testing.T) {
	tests := []struct {
		name    string
		tagName string
	}{
		{name: "with v prefix", tagName: "v1.2.3"},
		{name: "without v prefix", tagName: "1.2.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getArchiveNameForPlatform(tt.tagName)
			if !strings.Contains(got, runtime.GOOS) || !strings.Contains(got, runtime.GOARCH) {
				t.Fatalf("未包含平台信息: %s", got)
			}
			if !strings.Contains(got, normalizeVersion(tt.tagName)) {
				t.Fatalf("未包含版本号: %s", got)
			}
			if runtime.GOOS == "windows" {
				if !strings.HasSuffix(got, ".zip") {
					t.Fatalf("Windows 应使用 .zip: %s", got)
				}
			} else if !strings.HasSuffix(got, ".tar.gz") {
				t.Fatalf("非 Windows 应使用 .tar.gz: %s", got)
			}
		})
	}
}

func TestIsArchive(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{name: "tar.gz", filename: "file.tar.gz", want: true},
		{name: "zip", filename: "file.zip", want: true},
		{name: "tar", filename: "file.tar", want: false},
		{name: "tgz", filename: "file.tgz", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isArchive(tt.filename); got != tt.want {
				t.Fatalf("isArchive(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

func TestValidatePlatformFromFilename(t *testing.T) {
	otherOS := "windows"
	if runtime.GOOS == "windows" {
		otherOS = "linux"
	}
	otherArch := "arm64"
	if runtime.GOARCH == "arm64" {
		otherArch = "amd64"
	}

	tests := []struct {
		name        string
		filename    string
		wantErr     bool
		errContains string
	}{
		{
			name:     "valid platform",
			filename: fmt.Sprintf("cc-switch-cli-1.0.0-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH),
		},
		{
			name:        "os mismatch",
			filename:    fmt.Sprintf("cc-switch-cli-1.0.0-%s-%s.tar.gz", otherOS, runtime.GOARCH),
			wantErr:     true,
			errContains: "平台不匹配",
		},
		{
			name:        "arch mismatch",
			filename:    fmt.Sprintf("cc-switch-cli-1.0.0-%s-%s.tar.gz", runtime.GOOS, otherArch),
			wantErr:     true,
			errContains: "架构不匹配",
		},
		{
			name:        "invalid format",
			filename:    "invalid.tar.gz",
			wantErr:     true,
			errContains: "无法从文件名推断平台信息",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePlatformFromFilename(tt.filename)
			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不匹配: err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("错误信息不匹配: %v", err)
			}
		})
	}
}

func TestExtractBinary(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{name: "tar.gz", format: "tar.gz"},
		{name: "zip", format: "zip"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir := t.TempDir()
			archivePath := createArchiveWithBinary(t, tt.format, []byte("binary-"+tt.format))

			extractedPath, err := extractBinary(archivePath, destDir)
			if err != nil {
				t.Fatalf("解压失败: %v", err)
			}
			data, err := os.ReadFile(extractedPath)
			if err != nil {
				t.Fatalf("读取文件失败: %v", err)
			}
			if string(data) != "binary-"+tt.format {
				t.Fatalf("内容不匹配: %s", string(data))
			}
		})
	}
}

func TestExtractBinaryErrors(t *testing.T) {
	tests := []struct {
		name        string
		archivePath string
		wantErr     bool
		errContains string
		writeBad    bool
	}{
		{
			name:        "unsupported format",
			archivePath: filepath.Join(t.TempDir(), "test.tar"),
			wantErr:     true,
			errContains: "不支持的压缩格式",
		},
		{
			name:        "invalid zip",
			archivePath: filepath.Join(t.TempDir(), "bad.zip"),
			wantErr:     true,
			writeBad:    true,
		},
		{
			name:        "missing binary in tar.gz",
			archivePath: testutil.CreateTestArchive(t, "tar.gz", map[string][]byte{"README.md": []byte("readme")}),
			wantErr:     true,
			errContains: "未找到二进制文件",
		},
		{
			name:        "missing binary in zip",
			archivePath: testutil.CreateTestArchive(t, "zip", map[string][]byte{"README.md": []byte("readme")}),
			wantErr:     true,
			errContains: "未找到二进制文件",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.writeBad {
				if err := os.WriteFile(tt.archivePath, []byte("bad"), 0600); err != nil {
					t.Fatalf("创建坏包失败: %v", err)
				}
			}
			_, err := extractBinary(tt.archivePath, t.TempDir())
			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不匹配: err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("错误信息不匹配: %v", err)
			}
		})
	}
}

func TestDownloadFile(t *testing.T) {
	tests := []struct {
		name        string
		client      func(t *testing.T) *http.Client
		createErr   error
		wantErr     bool
		errContains string
		wantBody    string
	}{
		{
			name: "success",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					"https://example.com/file": {
						StatusCode: http.StatusOK,
						Body:       "payload",
					},
				})
			},
			wantBody: "payload",
		},
		{
			name: "status error",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					"https://example.com/file": {
						StatusCode: http.StatusBadGateway,
						Body:       "bad",
					},
				})
			},
			wantErr:     true,
			errContains: "下载失败",
		},
		{
			name: "timeout error",
			client: func(t *testing.T) *http.Client {
				return &http.Client{
					Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
						return nil, timeoutError{}
					}),
				}
			},
			wantErr:     true,
			errContains: "下载请求失败",
		},
		{
			name: "create file error",
			client: func(t *testing.T) *http.Client {
				return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
					"https://example.com/file": {
						StatusCode: http.StatusOK,
						Body:       "payload",
					},
				})
			},
			createErr:   errors.New("permission denied"),
			wantErr:     true,
			errContains: "创建文件失败",
		},
		{
			name: "read error",
			client: func(t *testing.T) *http.Client {
				return &http.Client{
					Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       errorReadCloser{err: errors.New("read fail")},
							Header:     make(http.Header),
							Request:    req,
						}, nil
					}),
				}
			},
			wantErr:     true,
			errContains: "写入文件失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetVersionHooks(t)
			httpClientForDownload = func() *http.Client { return tt.client(t) }
			if tt.createErr != nil {
				createFileFunc = func(string) (*os.File, error) {
					return nil, tt.createErr
				}
			}

			destPath := filepath.Join(t.TempDir(), "download.bin")
			err := downloadFile(destPath, "https://example.com/file")
			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不匹配: err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("错误信息不匹配: %v", err)
				}
				return
			}
			data, err := os.ReadFile(destPath)
			if err != nil {
				t.Fatalf("读取文件失败: %v", err)
			}
			if string(data) != tt.wantBody {
				t.Fatalf("内容不匹配: %s", string(data))
			}
		})
	}
}

func TestDownloadUpdate(t *testing.T) {
	tagName := "v9.9.9"
	archiveName := getArchiveNameForPlatform(tagName)
	downloadURL := "https://example.com/archive"
	archiveFormat := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveFormat = "zip"
	}

	tests := []struct {
		name        string
		setup       func(t *testing.T) (*ReleaseInfo, func(t *testing.T))
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			setup: func(t *testing.T) (*ReleaseInfo, func(t *testing.T)) {
				resetVersionHooks(t)
				exePath := fakeExecutable(t, "old")
				mkdirTempFunc = func(string, string) (string, error) {
					return os.MkdirTemp(filepath.Dir(exePath), "ccs-temp-*")
				}
				renameFunc = renameWithCopy
				archivePath := createArchiveWithBinary(t, archiveFormat, []byte("new"))
				archiveData, err := os.ReadFile(archivePath)
				if err != nil {
					t.Fatalf("读取归档失败: %v", err)
				}
				httpClientForDownload = func() *http.Client {
					return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
						downloadURL: {
							StatusCode: http.StatusOK,
							Body:       string(archiveData),
						},
					})
				}
				release := &ReleaseInfo{
					TagName: tagName,
					Assets: []Asset{
						{Name: archiveName, BrowserDownloadURL: downloadURL},
					},
				}
				return release, func(t *testing.T) {
					data, err := os.ReadFile(exePath)
					if err != nil {
						t.Fatalf("读取更新后可执行文件失败: %v", err)
					}
					if string(data) != "new" {
						t.Fatalf("可执行文件未更新: %s", string(data))
					}
					if _, err := os.Stat(exePath + ".old"); !os.IsNotExist(err) {
						t.Fatalf("备份文件应被删除")
					}
				}
			},
		},
		{
			name: "missing asset",
			setup: func(t *testing.T) (*ReleaseInfo, func(t *testing.T)) {
				resetVersionHooks(t)
				release := &ReleaseInfo{TagName: tagName}
				return release, func(t *testing.T) {}
			},
			wantErr:     true,
			errContains: "未找到适合当前平台",
		},
		{
			name: "download error",
			setup: func(t *testing.T) (*ReleaseInfo, func(t *testing.T)) {
				resetVersionHooks(t)
				httpClientForDownload = func() *http.Client {
					return testutil.MockHTTPClient(t, map[string]testutil.MockResponse{
						downloadURL: {
							StatusCode: http.StatusServiceUnavailable,
							Body:       "down",
						},
					})
				}
				release := &ReleaseInfo{
					TagName: tagName,
					Assets: []Asset{
						{Name: archiveName, BrowserDownloadURL: downloadURL},
					},
				}
				return release, func(t *testing.T) {}
			},
			wantErr:     true,
			errContains: "下载失败",
		},
		{
			name: "temp dir error",
			setup: func(t *testing.T) (*ReleaseInfo, func(t *testing.T)) {
				resetVersionHooks(t)
				mkdirTempFunc = func(string, string) (string, error) {
					return "", errors.New("disk full")
				}
				release := &ReleaseInfo{
					TagName: tagName,
					Assets: []Asset{
						{Name: archiveName, BrowserDownloadURL: downloadURL},
					},
				}
				return release, func(t *testing.T) {}
			},
			wantErr:     true,
			errContains: "创建临时目录失败",
		},
		{
			name: "network timeout",
			setup: func(t *testing.T) (*ReleaseInfo, func(t *testing.T)) {
				resetVersionHooks(t)
				httpClientForDownload = func() *http.Client {
					return &http.Client{
						Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
							return nil, timeoutError{}
						}),
					}
				}
				release := &ReleaseInfo{
					TagName: tagName,
					Assets: []Asset{
						{Name: archiveName, BrowserDownloadURL: downloadURL},
					},
				}
				return release, func(t *testing.T) {}
			},
			wantErr:     true,
			errContains: "请检查网络连接",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			release, check := tt.setup(t)
			err := DownloadUpdate(release)
			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不匹配: err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("错误信息不匹配: %v", err)
				}
				return
			}
			check(t)
		})
	}
}

func TestInstallBinarySuccess(t *testing.T) {
	archiveFormat := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveFormat = "zip"
	}

	tests := []struct {
		name  string
		setup func(t *testing.T) (string, bool, func(t *testing.T))
	}{
		{
			name: "raw binary",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				exePath := fakeExecutable(t, "old")
				mkdirTempFunc = func(string, string) (string, error) {
					return os.MkdirTemp(filepath.Dir(exePath), "ccs-temp-*")
				}
				renameFunc = renameWithCopy
				sourcePath := filepath.Join(t.TempDir(), "ccs-bin")
				if err := os.WriteFile(sourcePath, []byte("new-raw"), 0600); err != nil {
					t.Fatalf("创建源文件失败: %v", err)
				}
				return sourcePath, false, func(t *testing.T) {
					data, err := os.ReadFile(exePath)
					if err != nil {
						t.Fatalf("读取可执行文件失败: %v", err)
					}
					if string(data) != "new-raw" {
						t.Fatalf("内容不匹配: %s", string(data))
					}
				}
			},
		},
		{
			name: "archive with platform check",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				exePath := fakeExecutable(t, "old-archive")
				mkdirTempFunc = func(string, string) (string, error) {
					return os.MkdirTemp(filepath.Dir(exePath), "ccs-temp-*")
				}
				renameFunc = renameWithCopy
				archivePath := createArchiveWithBinary(t, archiveFormat, []byte("new-archive"))
				targetName := fmt.Sprintf("cc-switch-cli-1.0.0-%s-%s", runtime.GOOS, runtime.GOARCH)
				if archiveFormat == "zip" {
					targetName += ".zip"
				} else {
					targetName += ".tar.gz"
				}
				targetPath := filepath.Join(filepath.Dir(archivePath), targetName)
				if err := os.Rename(archivePath, targetPath); err != nil {
					t.Fatalf("重命名归档失败: %v", err)
				}
				return targetPath, false, func(t *testing.T) {
					data, err := os.ReadFile(exePath)
					if err != nil {
						t.Fatalf("读取可执行文件失败: %v", err)
					}
					if string(data) != "new-archive" {
						t.Fatalf("内容不匹配: %s", string(data))
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourcePath, skipCheck, check := tt.setup(t)
			err := InstallBinary(sourcePath, skipCheck)
			if err != nil {
				t.Fatalf("安装失败: %v", err)
			}
			check(t)
		})
	}
}

func TestInstallBinaryErrors(t *testing.T) {
	archiveFormat := "tar.gz"
	if runtime.GOOS == "windows" {
		archiveFormat = "zip"
	}

	tests := []struct {
		name        string
		setup       func(t *testing.T) (string, bool, func(t *testing.T))
		wantErr     string
		skipWindows bool
	}{
		{
			name: "executable error",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				executableFunc = func() (string, error) {
					return "", errors.New("no exe")
				}
				return "ccs.bin", false, func(t *testing.T) {}
			},
			wantErr: "获取可执行文件路径失败",
		},
		{
			name: "eval symlink error",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				exePath := fakeExecutable(t, "old")
				executableFunc = func() (string, error) { return exePath, nil }
				evalSymlinksFunc = func(string) (string, error) {
					return "", errors.New("bad link")
				}
				return "ccs.bin", false, func(t *testing.T) {}
			},
			wantErr: "解析符号链接失败",
		},
		{
			name: "temp dir error",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				fakeExecutable(t, "old")
				mkdirTempFunc = func(string, string) (string, error) {
					return "", errors.New("disk full")
				}
				sourcePath := filepath.Join(t.TempDir(), "ccs.bin")
				if err := os.WriteFile(sourcePath, []byte("bin"), 0600); err != nil {
					t.Fatalf("创建源文件失败: %v", err)
				}
				return sourcePath, false, func(t *testing.T) {}
			},
			wantErr: "创建临时目录失败",
		},
		{
			name: "platform mismatch",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				fakeExecutable(t, "old")
				otherOS := "windows"
				if runtime.GOOS == "windows" {
					otherOS = "linux"
				}
				fileName := fmt.Sprintf("cc-switch-cli-1.0.0-%s-%s.tar.gz", otherOS, runtime.GOARCH)
				sourcePath := filepath.Join(t.TempDir(), fileName)
				if err := os.WriteFile(sourcePath, []byte("bad"), 0600); err != nil {
					t.Fatalf("创建文件失败: %v", err)
				}
				return sourcePath, false, func(t *testing.T) {}
			},
			wantErr: "平台不匹配",
		},
		{
			name: "extract error",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				fakeExecutable(t, "old")
				targetName := fmt.Sprintf("cc-switch-cli-1.0.0-%s-%s", runtime.GOOS, runtime.GOARCH)
				if archiveFormat == "zip" {
					targetName += ".zip"
				} else {
					targetName += ".tar.gz"
				}
				sourcePath := filepath.Join(t.TempDir(), targetName)
				if err := os.WriteFile(sourcePath, []byte("bad-archive"), 0600); err != nil {
					t.Fatalf("创建归档失败: %v", err)
				}
				return sourcePath, true, func(t *testing.T) {}
			},
			wantErr: "解压失败",
		},
		{
			name:        "chmod error",
			skipWindows: runtime.GOOS == "windows",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				fakeExecutable(t, "old")
				sourcePath := filepath.Join(t.TempDir(), "ccs.bin")
				if err := os.WriteFile(sourcePath, []byte("bin"), 0600); err != nil {
					t.Fatalf("创建源文件失败: %v", err)
				}
				chmodFunc = func(string, os.FileMode) error {
					return errors.New("chmod denied")
				}
				return sourcePath, false, func(t *testing.T) {}
			},
			wantErr: "设置可执行权限失败",
		},
		{
			name: "backup rename error",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				fakeExecutable(t, "old")
				sourcePath := filepath.Join(t.TempDir(), "ccs.bin")
				if err := os.WriteFile(sourcePath, []byte("bin"), 0600); err != nil {
					t.Fatalf("创建源文件失败: %v", err)
				}
				renameFunc = func(string, string) error {
					return os.ErrPermission
				}
				return sourcePath, false, func(t *testing.T) {}
			},
			wantErr: "备份当前版本失败",
		},
		{
			name: "install rename error",
			setup: func(t *testing.T) (string, bool, func(t *testing.T)) {
				resetVersionHooks(t)
				exePath := fakeExecutable(t, "old")
				sourcePath := filepath.Join(t.TempDir(), "ccs.bin")
				if err := os.WriteFile(sourcePath, []byte("bin"), 0600); err != nil {
					t.Fatalf("创建源文件失败: %v", err)
				}
				callCount := 0
				renameFunc = func(oldPath, newPath string) error {
					callCount++
					if callCount == 2 {
						return errors.New("rename failed")
					}
					return os.Rename(oldPath, newPath)
				}
				return sourcePath, false, func(t *testing.T) {
					data, err := os.ReadFile(exePath)
					if err != nil {
						t.Fatalf("读取可执行文件失败: %v", err)
					}
					if string(data) != "old" {
						t.Fatalf("备份恢复失败: %s", string(data))
					}
				}
			},
			wantErr: "安装新版本失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipWindows {
				t.Skip("Windows 不执行 chmod 场景")
			}
			sourcePath, skipCheck, check := tt.setup(t)
			err := InstallBinary(sourcePath, skipCheck)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("错误不匹配: %v", err)
			}
			check(t)
		})
	}
}

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(t *testing.T) (string, string)
		wantErr     bool
		errContains string
	}{
		{
			name: "success",
			setup: func(t *testing.T) (string, string) {
				src := filepath.Join(t.TempDir(), "src.txt")
				if err := os.WriteFile(src, []byte("copy"), 0600); err != nil {
					t.Fatalf("创建源文件失败: %v", err)
				}
				dst := filepath.Join(t.TempDir(), "dst.txt")
				return src, dst
			},
		},
		{
			name: "source missing",
			setup: func(t *testing.T) (string, string) {
				return filepath.Join(t.TempDir(), "missing.txt"), filepath.Join(t.TempDir(), "dst.txt")
			},
			wantErr:     true,
			errContains: "no such file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := tt.setup(t)
			err := copyFile(src, dst)
			if (err != nil) != tt.wantErr {
				t.Fatalf("错误状态不匹配: err=%v wantErr=%v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && err != nil && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("错误信息不匹配: %v", err)
				}
				return
			}
			data, err := os.ReadFile(dst)
			if err != nil {
				t.Fatalf("读取目标文件失败: %v", err)
			}
			if string(data) != "copy" {
				t.Fatalf("内容不匹配: %s", string(data))
			}
		})
	}
}
