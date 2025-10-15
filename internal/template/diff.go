package template

import (
	"fmt"
	"strings"

	"github.com/aymanbagabas/go-udiff"
)

// GenerateDiff 生成两个文本之间的 unified diff
func GenerateDiff(oldText, newText, oldLabel, newLabel string) string {
	// 检查是否完全相同
	if oldText == newText {
		return "未发现差异"
	}

	// 使用 go-udiff 生成 unified diff，它原生支持 UTF-8 和中文
	edits := udiff.Strings(oldText, newText)
	unified := fmt.Sprint(udiff.ToUnified(oldLabel, newLabel, oldText, edits, 3))

	if unified == "" || len(edits) == 0 {
		return "未发现差异"
	}

	return unified
}

// FormatDiffForCLI 为 CLI 输出格式化 diff（带颜色）
func FormatDiffForCLI(diff string) string {
	lines := strings.Split(diff, "\n")
	var result strings.Builder

	for _, line := range lines {
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			result.WriteString("\033[1m" + line + "\033[0m\n") // Bold
		} else if strings.HasPrefix(line, "-") {
			result.WriteString("\033[31m" + line + "\033[0m\n") // Red
		} else if strings.HasPrefix(line, "+") {
			result.WriteString("\033[32m" + line + "\033[0m\n") // Green
		} else if strings.HasPrefix(line, "@@") {
			result.WriteString("\033[36m" + line + "\033[0m\n") // Cyan
		} else {
			result.WriteString(line + "\n")
		}
	}

	return result.String()
}
