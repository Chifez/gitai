package git

import (
	"strings"
	"testing"
)

func TestSanitizeDiff(t *testing.T) {
	tests := []struct {
		name      string
		diff      string
		want      string
		truncated bool
	}{
		{
			name:      "short normal diff",
			diff:      "--- a/file.go\n+++ b/file.go",
			want:      "--- a/file.go\n+++ b/file.go",
			truncated: false,
		},
		{
			name:      "binary diff replace",
			diff:      "Binary files a/bin.exe and b/bin.exe differ",
			want:      "Binary file modified: bin.exe",
			truncated: false,
		},
		{
			name:      "long diff truncates",
			diff:      strings.Repeat("a", maxChars+100),
			want:      strings.Repeat("a", maxChars),
			truncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, trunc := SanitizeDiff(tt.diff)
			if got != tt.want {
				t.Errorf("SanitizeDiff() got = %v, want = %v", got, tt.want)
			}
			if trunc != tt.truncated {
				t.Errorf("SanitizeDiff() trunc = %v, want = %v", trunc, tt.truncated)
			}
		})
	}
}
