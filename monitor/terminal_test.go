package monitor

import "testing"

func TestCategorizeCommand(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		// Build
		{"make build", "build"},
		{"go build ./...", "build"},
		{"npm run build", "build"},
		{"cargo build --release", "build"},
		{"tsc --watch", "build"},
		{"webpack --mode=production", "build"},
		{"gcc main.c -o main", "build"},

		// Test
		{"go test ./...", "test"},
		{"npm test", "test"},
		{"pytest tests/", "test"},
		{"jest --coverage", "test"},
		{"cargo test", "test"},

		// Install
		{"npm install express", "install"},
		{"pip install requests", "install"},
		{"go get github.com/pkg/errors", "install"},
		{"brew install jq", "install"},
		{"go mod tidy", "install"},

		// Git
		{"git commit -m 'fix'", "git"},
		{"git push origin main", "git"},
		{"git status", "git"},

		// Run
		{"go run main.go", "run"},
		{"node server.js", "run"},
		{"python app.py", "run"},
		{"npm start", "run"},
		{"cargo run", "run"},

		// Lint
		{"eslint src/", "lint"},
		{"prettier --write .", "lint"},
		{"golangci-lint run", "lint"},

		// File
		{"cat README.md", "file"},
		{"grep -r 'TODO' .", "file"},
		{"find . -name '*.go'", "file"},
		{"ls -la", "file"},
		{"mkdir -p src/utils", "file"},
		{"rm old_file.txt", "file"},

		// Other
		{"echo 'hello'", "other"},
		{"date", "other"},
		{"whoami", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := CategorizeCommand(tt.cmd)
			if got != tt.want {
				t.Errorf("CategorizeCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestMatchesAny(t *testing.T) {
	tests := []struct {
		s        string
		patterns []string
		want     bool
	}{
		{"go test ./...", []string{"go test", "npm test"}, true},
		{"npm test", []string{"go test", "npm test"}, true},
		{"echo hello", []string{"go test", "npm test"}, false},
		{"", []string{"test"}, false},
		{"test", []string{}, false},
	}

	for _, tt := range tests {
		got := matchesAny(tt.s, tt.patterns...)
		if got != tt.want {
			t.Errorf("matchesAny(%q, %v) = %v, want %v", tt.s, tt.patterns, got, tt.want)
		}
	}
}

func TestIsIgnoredProcess(t *testing.T) {
	tests := []struct {
		cmd  string
		want bool
	}{
		{"ps -p 123 -o command=", true},
		{"pgrep -P 456", true},
		{"/bin/sh -c something", true},
		{"/bin/zsh", true},
		{"/bin/bash", true},
		{"(zsh)", true},
		{"go test ./...", false},
		{"node server.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			got := isIgnoredProcess(tt.cmd)
			if got != tt.want {
				t.Errorf("isIgnoredProcess(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestNewTerminalMonitor(t *testing.T) {
	tm := NewTerminalMonitor(50)
	if tm == nil {
		t.Fatal("NewTerminalMonitor returned nil")
	}
	if tm.maxHistory != 50 {
		t.Errorf("maxHistory = %d, want 50", tm.maxHistory)
	}
}

func TestNewTerminalMonitor_DefaultMaxHistory(t *testing.T) {
	tm := NewTerminalMonitor(0)
	if tm.maxHistory != 50 {
		t.Errorf("maxHistory = %d, want 50 (default)", tm.maxHistory)
	}

	tm2 := NewTerminalMonitor(-5)
	if tm2.maxHistory != 50 {
		t.Errorf("maxHistory = %d, want 50 (default for negative)", tm2.maxHistory)
	}
}

func TestCategorizeCommand_CaseInsensitive(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"GO TEST ./...", "test"},
		{"NPM TEST", "test"},
		{"Git Status", "git"},
		{"MAKE build", "build"},
	}

	for _, tt := range tests {
		got := CategorizeCommand(tt.cmd)
		if got != tt.want {
			t.Errorf("CategorizeCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}
