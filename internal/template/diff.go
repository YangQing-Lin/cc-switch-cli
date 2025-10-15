package template

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// GenerateDiff 生成两个文本之间的 unified diff
func GenerateDiff(oldText, newText, oldLabel, newLabel string) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(oldText, newText, false)

	if len(diffs) == 0 || (len(diffs) == 1 && diffs[0].Type == diffmatchpatch.DiffEqual) {
		return "No differences found."
	}

	// 生成 unified diff 格式
	patches := dmp.PatchMake(oldText, diffs)
	unified := dmp.PatchToText(patches)

	if unified == "" {
		return "No differences found."
	}

	// 添加文件头
	var result strings.Builder
	result.WriteString(fmt.Sprintf("--- %s\n", oldLabel))
	result.WriteString(fmt.Sprintf("+++ %s\n", newLabel))
	result.WriteString(unified)

	return result.String()
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
