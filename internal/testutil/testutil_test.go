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
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCreateTempDirAndFile(t *testing.T) {
	tests := []struct {
		name    string
		subPath string
		content string
	}{
		{name: "simple file", subPath: "file.txt", content: "hello"},
		{name: "nested file", subPath: filepath.Join("nested", "file.txt"), content: "nested"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := CreateTempDir(t)
			path := CreateTempFile(t, dir, tt.subPath, tt.content)
			AssertFileExists(t, path)
			AssertFileContent(t, path, tt.content)
		})
	}
}

func TestAssertHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "exists.txt")
	if err := os.WriteFile(filePath, []byte("data"), 0644); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	tests := []struct {
		name       string
		run        func(t *testing.T)
		shouldPass bool
	}{
		{
			name: "AssertFileExists pass",
			run: func(t *testing.T) {
				AssertFileExists(t, filePath)
			},
			shouldPass: true,
		},
		{
			name: "AssertFileNotExists pass",
			run: func(t *testing.T) {
				AssertFileNotExists(t, filepath.Join(tmpDir, "missing.txt"))
			},
			shouldPass: true,
		},
		{
			name: "AssertFileContent pass",
			run: func(t *testing.T) {
				AssertFileContent(t, filePath, "data")
			},
			shouldPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok := t.Run("assert", tt.run)
			if ok != tt.shouldPass {
				t.Fatalf("expected pass=%v, got %v", tt.shouldPass, ok)
			}
		})
	}
}

func TestAssertFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("跳过 Windows 权限测试")
	}

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "perm.txt")
	if err := os.WriteFile(filePath, []byte("perm"), 0600); err != nil {
		t.Fatalf("创建文件失败: %v", err)
	}

	AssertFileMode(t, filePath, 0600)
}

func TestWithTempHome(t *testing.T) {
	WithTempHome(t, func(home string) {
		if got := os.Getenv("HOME"); got != home {
			t.Fatalf("HOME 未设置为临时目录: %s", got)
		}
		if got := os.Getenv("USERPROFILE"); got != home {
			t.Fatalf("USERPROFILE 未设置为临时目录: %s", got)
		}

		path := filepath.Join(home, "marker.txt")
		if err := os.WriteFile(path, []byte("ok"), 0644); err != nil {
			t.Fatalf("写入文件失败: %v", err)
		}
		AssertFileExists(t, path)
	})
}

func TestWithTempCWD(t *testing.T) {
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}

	ok := t.Run("temp cwd", func(t *testing.T) {
		WithTempCWD(t, func(cwd string) {
			current, err := os.Getwd()
			if err != nil {
				t.Fatalf("获取当前工作目录失败: %v", err)
			}
			if current != cwd {
				t.Fatalf("当前目录不匹配: %s != %s", current, cwd)
			}
		})
	})
	if !ok {
		t.Fatalf("子测试失败")
	}

	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	if current != original {
		t.Fatalf("工作目录未恢复: %s != %s", current, original)
	}
}

func TestCaptureOutput(t *testing.T) {
	stdout, stderr := CaptureOutput(t, func() {
		_, _ = io.WriteString(os.Stdout, "hello")
		_, _ = io.WriteString(os.Stderr, "oops")
	})

	if stdout != "hello" {
		t.Fatalf("stdout 不匹配: %s", stdout)
	}
	if stderr != "oops" {
		t.Fatalf("stderr 不匹配: %s", stderr)
	}
}

func TestMockHTTPClient(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "known response",
			url:        "https://example.com/test",
			wantStatus: http.StatusCreated,
			wantBody:   "created",
		},
		{
			name:       "unknown response",
			url:        "https://example.com/missing",
			wantStatus: http.StatusNotFound,
			wantBody:   "not found",
		},
	}

	client := MockHTTPClient(t, map[string]MockResponse{
		"https://example.com/test": {
			StatusCode: http.StatusCreated,
			Body:       "created",
			Headers: map[string]string{
				"X-Test": "ok",
			},
		},
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatalf("创建请求失败: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("请求失败: %v", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("读取响应失败: %v", err)
			}

			if resp.StatusCode != tt.wantStatus {
				t.Fatalf("状态码不匹配: %d != %d", resp.StatusCode, tt.wantStatus)
			}
			if string(body) != tt.wantBody {
				t.Fatalf("响应内容不匹配: %s != %s", string(body), tt.wantBody)
			}
		})
	}
}

func TestCreateTestArchive(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{name: "zip archive", format: "zip"},
		{name: "tar.gz archive", format: "tar.gz"},
	}

	files := map[string][]byte{
		"bin/ccs":     []byte("binary"),
		"README.txt":  []byte("readme"),
		"config.json": []byte(`{"ok":true}`),
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archivePath := CreateTestArchive(t, tt.format, files)
			switch tt.format {
			case "zip":
				verifyZipArchive(t, archivePath, files)
			case "tar.gz":
				verifyTarGzArchive(t, archivePath, files)
			}
		})
	}
}

func TestBubbleTeaTestHelper(t *testing.T) {
	model := simpleModel{}
	keys := []string{"a", "enter", "left"}
	final := BubbleTeaTestHelper(t, model, keys)

	got, ok := final.(simpleModel)
	if !ok {
		t.Fatalf("返回模型类型不匹配")
	}

	if got.last != "left" {
		t.Fatalf("最后按键不匹配: %s", got.last)
	}
	if len(got.history) != len(keys) {
		t.Fatalf("按键历史长度不匹配: %d", len(got.history))
	}
}

type simpleModel struct {
	last    string
	history []string
}

func (m simpleModel) Init() tea.Cmd {
	return nil
}

func (m simpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		key := keyMsg.String()
		if keyMsg.Type == tea.KeyRunes {
			key = string(keyMsg.Runes)
		}
		m.last = key
		m.history = append(m.history, key)
	}
	return m, nil
}

func (m simpleModel) View() string {
	return ""
}

func verifyZipArchive(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("打开 zip 失败: %v", err)
	}
	defer reader.Close()

	found := make(map[string][]byte)
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("读取 zip 文件失败: %v", err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("读取 zip 内容失败: %v", err)
		}
		found[file.Name] = data
	}

	assertArchiveContents(t, found, files)
}

func verifyTarGzArchive(t *testing.T, path string, files map[string][]byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取 tar.gz 失败: %v", err)
	}

	gzipReader, err := gzipReaderFromBytes(data)
	if err != nil {
		t.Fatalf("解压 gzip 失败: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	found := make(map[string][]byte)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("读取 tar 内容失败: %v", err)
		}
		data, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("读取 tar 文件失败: %v", err)
		}
		found[header.Name] = data
	}

	assertArchiveContents(t, found, files)
}

func gzipReaderFromBytes(data []byte) (io.ReadCloser, error) {
	reader := bytes.NewReader(data)
	return gzip.NewReader(reader)
}

func assertArchiveContents(t *testing.T, got map[string][]byte, want map[string][]byte) {
	t.Helper()
	for name, content := range want {
		key := filepath.ToSlash(name)
		gotContent, ok := got[key]
		if !ok {
			t.Fatalf("归档缺少文件: %s", key)
		}
		if !bytes.Equal(gotContent, content) {
			t.Fatalf("文件内容不匹配: %s", key)
		}
	}
}
