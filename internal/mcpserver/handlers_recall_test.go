package mcpserver

import (
	"testing"
)

func TestDetectScope(t *testing.T) {
	t.Parallel()
	cases := []struct {
		query string
		want  string
	}{
		{"internal/store/sessions.go", "file"},
		{"sessions.go", "file"},
		{"main.go", "file"},
		{"internal/store/", "directory"},
		{"internal/store", "directory"},
		{"cmd/serve", "directory"},
		{"authentication", "all"},
		{"how to test", "all"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			t.Parallel()
			got := detectScope(tc.query)
			if got != tc.want {
				t.Errorf("detectScope(%q) = %q, want %q", tc.query, got, tc.want)
			}
		})
	}
}
