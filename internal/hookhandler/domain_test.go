package hookhandler

import "testing"

func TestDetectDomain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		prompt string
		want   string
	}{
		{"auth", "fix the login token validation", "auth"},
		{"database", "optimize the SQL query for users table", "database"},
		{"ui", "fix the button layout in modal", "ui"},
		{"api", "add a new REST endpoint for users", "api"},
		{"config", "update the environment settings", "config"},
		{"infra", "fix the docker deployment pipeline", "infra"},
		{"test", "add test coverage for parser", "test"},
		{"general", "make this better", "general"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := detectDomain(tt.prompt)
			if got != tt.want {
				t.Errorf("detectDomain(%q) = %q, want %q", tt.prompt, got, tt.want)
			}
		})
	}
}
