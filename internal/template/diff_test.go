package template

import (
	"strings"
	"testing"
)

func TestGenerateDiff(t *testing.T) {
	cases := []struct {
		name      string
		oldText   string
		newText   string
		oldLabel  string
		newLabel  string
		contains  []string
		notEquals string
	}{
		{
			name:      "same_content",
			oldText:   "same",
			newText:   "same",
			oldLabel:  "Old",
			newLabel:  "New",
			contains:  []string{"未发现差异"},
			notEquals: "",
		},
		{
			name:      "different_content",
			oldText:   "old line",
			newText:   "new line",
			oldLabel:  "Old",
			newLabel:  "New",
			contains:  []string{"--- Old", "+++ New", "-old line", "+new line"},
			notEquals: "未发现差异",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diff := GenerateDiff(tc.oldText, tc.newText, tc.oldLabel, tc.newLabel)
			if tc.notEquals != "" && diff == tc.notEquals {
				t.Fatalf("unexpected diff: %s", diff)
			}
			for _, want := range tc.contains {
				if !strings.Contains(diff, want) {
					t.Fatalf("expected diff to contain %q, got %s", want, diff)
				}
			}
		})
	}
}

func TestFormatDiffForCLI(t *testing.T) {
	cases := []struct {
		name      string
		diff      string
		contains  []string
		notEquals string
	}{
		{
			name: "ansi_styles",
			diff: strings.Join([]string{
				"--- Old",
				"+++ New",
				"@@ -1 +1 @@",
				"-old",
				"+new",
				" unchanged",
			}, "\n"),
			contains: []string{"\033[1m--- Old\033[0m", "\033[32m+new\033[0m", "\033[31m-old\033[0m", "\033[36m@@ -1 +1 @@\033[0m"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDiffForCLI(tc.diff)
			for _, want := range tc.contains {
				if !strings.Contains(got, want) {
					t.Fatalf("expected formatted diff to contain %q, got %s", want, got)
				}
			}
		})
	}
}
