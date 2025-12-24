package testutil

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// CreateTempDir 创建临时测试目录
func CreateTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "cc-switch-test-*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// CreateTempFile 创建临时测试文件
func CreateTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	return path
}

// AssertFileExists 断言文件存在
func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("文件不存在: %s", path)
	}
}

// AssertFileNotExists 断言文件不存在
func AssertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("文件不应该存在: %s", path)
	}
}

// AssertFileContent 断言文件内容
func AssertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != expected {
		t.Errorf("文件内容不匹配\n期望: %s\n实际: %s", expected, string(content))
	}
}

// AssertFileMode 断言文件权限（仅在非Windows系统）
func AssertFileMode(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("获取文件信息失败: %v", err)
	}
	actual := info.Mode().Perm()
	if actual != expected {
		t.Errorf("文件权限不匹配\n期望: %o\n实际: %o", expected, actual)
	}
}

// WithTempHome 设置临时 HOME/USERPROFILE 环境变量
func WithTempHome(t *testing.T, setupFunc func(homeDir string)) {
	t.Helper()
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	if setupFunc != nil {
		setupFunc(homeDir)
	}
}

// WithTempCWD 临时切换当前工作目录。
//
// 注意：os.Chdir 是进程级全局副作用，调用方不得与 t.Parallel() 或并行测试一起使用。
func WithTempCWD(t *testing.T, setupFunc func(cwd string)) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(original)
	})

	if setupFunc != nil {
		setupFunc(tempDir)
	}
}

// CaptureOutput 捕获函数执行期间的 stdout 和 stderr 输出
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	origStdout := os.Stdout
	origStderr := os.Stderr

	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("创建 stdout pipe 失败: %v", err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		_ = stdoutReader.Close()
		_ = stdoutWriter.Close()
		t.Fatalf("创建 stderr pipe 失败: %v", err)
	}

	os.Stdout = stdoutWriter
	os.Stderr = stderrWriter

	var closeOnce sync.Once
	closePipes := func() {
		closeOnce.Do(func() {
			_ = stdoutWriter.Close()
			_ = stderrWriter.Close()
		})
	}
	defer closePipes()

	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	stdoutCh := make(chan []byte, 1)
	stderrCh := make(chan []byte, 1)

	go func() {
		defer stdoutReader.Close()
		data, _ := io.ReadAll(stdoutReader)
		stdoutCh <- data
	}()
	go func() {
		defer stderrReader.Close()
		data, _ := io.ReadAll(stderrReader)
		stderrCh <- data
	}()

	fn()

	closePipes()

	stdout = string(<-stdoutCh)
	stderr = string(<-stderrCh)
	return stdout, stderr
}

// MockResponse 定义 HTTP mock 响应
type MockResponse struct {
	StatusCode int
	Body       string
	Headers    map[string]string
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

// MockHTTPClient 返回一个使用自定义 RoundTripper 的 http.Client
func MockHTTPClient(t *testing.T, responses map[string]MockResponse) *http.Client {
	t.Helper()
	return &http.Client{
		Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
			resp, ok := responses[req.URL.String()]
			if !ok {
				resp = MockResponse{
					StatusCode: http.StatusNotFound,
					Body:       "not found",
				}
			}

			statusCode := resp.StatusCode
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			header := make(http.Header)
			for key, value := range resp.Headers {
				header.Set(key, value)
			}

			return &http.Response{
				StatusCode: statusCode,
				Body:       io.NopCloser(bytes.NewBufferString(resp.Body)),
				Header:     header,
				Request:    req,
			}, nil
		}),
	}
}

// CreateTestArchive 生成临时的 .zip 或 .tar.gz 归档文件
func CreateTestArchive(t *testing.T, format string, files map[string][]byte) string {
	t.Helper()

	tempDir := t.TempDir()
	var archivePath string

	switch format {
	case "zip":
		archivePath = filepath.Join(tempDir, "test.zip")
		if err := createZipArchive(archivePath, files); err != nil {
			t.Fatalf("创建 zip 归档失败: %v", err)
		}
	case "tar.gz":
		archivePath = filepath.Join(tempDir, "test.tar.gz")
		if err := createTarGzArchive(archivePath, files); err != nil {
			t.Fatalf("创建 tar.gz 归档失败: %v", err)
		}
	default:
		t.Fatalf("不支持的归档格式: %s", format)
	}

	return archivePath
}

// BubbleTeaTestHelper 构造 tea.KeyMsg 序列并依次调用 model.Update
func BubbleTeaTestHelper(t *testing.T, model tea.Model, keys []string) tea.Model {
	t.Helper()
	for _, key := range keys {
		msg := buildKeyMsg(key)
		updated, _ := model.Update(msg)
		model = updated
	}
	return model
}

func createZipArchive(path string, files map[string][]byte) error {
	archiveFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	writer := zip.NewWriter(archiveFile)
	defer writer.Close()

	for name, content := range files {
		headerName := filepath.ToSlash(name)
		entry, err := writer.Create(headerName)
		if err != nil {
			return err
		}
		if _, err := entry.Write(content); err != nil {
			return err
		}
	}

	return nil
}

func createTarGzArchive(path string, files map[string][]byte) error {
	archiveFile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer archiveFile.Close()

	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for name, content := range files {
		header := &tar.Header{
			Name: filepath.ToSlash(name),
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write(content); err != nil {
			return err
		}
	}

	return nil
}

func buildKeyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "delete":
		return tea.KeyMsg{Type: tea.KeyDelete}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}
