package prompt

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	diff := "--- a/file.txt\n+++ b/file.txt\n+ hello"
	
	tests := []struct {
		name     string
		opts     Options
		mustHave []string
		mustNot  []string
	}{
		{
			name: "conventional default",
			opts: Options{Style: "conventional", IncludeBody: false},
			mustHave: []string{"type(scope): description", "Do NOT include a body"},
			mustNot:  []string{"Emoji mapping", "concise sentence"},
		},
		{
			name: "simple style",
			opts: Options{Style: "simple", IncludeBody: true},
			mustHave: []string{"single concise sentence"},
			mustNot:  []string{"type(scope): description", "Include a brief multi-line body"},
		},
		{
			name: "emoji style with hints",
			opts: Options{
				Style:       "emoji",
				MaxLength:   50,
				Lang:        "french",
				Context:     "fix login bug",
				IncludeBody: true,
			},
			mustHave: []string{
				"Emoji mapping",
				"under 50 characters",
				"in french",
				"context from the developer: fix login bug",
				"Include a brief multi-line body",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildPrompt(diff, tt.opts)
			
			for _, check := range tt.mustHave {
				if !strings.Contains(got, check) {
					t.Errorf("prompt missing expected substring: %s", check)
				}
			}
			for _, check := range tt.mustNot {
				if strings.Contains(got, check) {
					t.Errorf("prompt contains forbidden substring: %s", check)
				}
			}
			if !strings.Contains(got, diff) {
				t.Errorf("prompt missing the diff")
			}
		})
	}
}
