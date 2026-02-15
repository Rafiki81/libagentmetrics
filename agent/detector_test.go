package agent

import (
	"testing"

	"github.com/Rafiki81/libagentmetrics/config"
)

func TestNewDetector(t *testing.T) {
	r := NewRegistry()
	cfg := config.DefaultConfig()
	d := NewDetector(r, cfg)

	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
	if d.Registry != r {
		t.Error("Detector.Registry not set correctly")
	}
	if d.Config != cfg {
		t.Error("Detector.Config not set correctly")
	}
}

func TestParsePSLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantPID int
		wantCPU float64
		wantCmd string
		wantErr bool
	}{
		{
			name:    "normal line",
			line:    "user   1234  5.3  1.2  123456  78900   ??  S    10:00AM   0:01.23 /usr/bin/claude --config test",
			wantPID: 1234,
			wantCPU: 5.3,
			wantCmd: "/usr/bin/claude",
			wantErr: false,
		},
		{
			name:    "zero cpu",
			line:    "root     42  0.0  0.1   12345   6789   ??  S    09:00AM   0:00.01 /sbin/launchd",
			wantPID: 42,
			wantCPU: 0.0,
			wantCmd: "/sbin/launchd",
			wantErr: false,
		},
		{
			name:    "too few fields",
			line:    "user 123 0.0",
			wantErr: true,
		},
		{
			name:    "invalid pid",
			line:    "user   abc  5.3  1.2  123456  78900   ??  S    10:00AM   0:01.23 /usr/bin/test",
			wantErr: true,
		},
		{
			name:    "empty line",
			line:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proc, err := parsePSLine(tt.line)
			if tt.wantErr {
				if err == nil {
					t.Error("parsePSLine() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePSLine() unexpected error: %v", err)
			}
			if proc.PID != tt.wantPID {
				t.Errorf("PID = %d, want %d", proc.PID, tt.wantPID)
			}
			if proc.CPU != tt.wantCPU {
				t.Errorf("CPU = %f, want %f", proc.CPU, tt.wantCPU)
			}
			if proc.Command != tt.wantCmd {
				t.Errorf("Command = %q, want %q", proc.Command, tt.wantCmd)
			}
		})
	}
}

func TestParsePSLine_CmdFull(t *testing.T) {
	line := "user   999  1.0  0.5  123456  78900   ??  S    10:00AM   0:01.23 /usr/bin/claude --config /path/to/config"
	proc, err := parsePSLine(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "/usr/bin/claude --config /path/to/config"
	if proc.CmdFull != expected {
		t.Errorf("CmdFull = %q, want %q", proc.CmdFull, expected)
	}
}

func TestExtractBaseName(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"/usr/local/bin/claude", "claude"},
		{"/usr/bin/node", "node"},
		{"claude", "claude"},
		{"/a/b/c/d/e", "e"},
		{"", ""},
	}

	for _, tt := range tests {
		got := extractBaseName(tt.cmd)
		if got != tt.want {
			t.Errorf("extractBaseName(%q) = %q, want %q", tt.cmd, got, tt.want)
		}
	}
}

func TestMatchProcess(t *testing.T) {
	r := NewRegistry()
	cfg := config.DefaultConfig()
	d := NewDetector(r, cfg)

	tests := []struct {
		name    string
		proc    processInfo
		wantID  string
		wantNil bool
	}{
		{
			name:   "exact process name match",
			proc:   processInfo{PID: 1, Command: "claude", CmdFull: "/usr/bin/claude"},
			wantID: "claude-code",
		},
		{
			name:   "basename extraction",
			proc:   processInfo{PID: 2, Command: "/usr/local/bin/claude", CmdFull: "/usr/local/bin/claude --help"},
			wantID: "claude-code",
		},
		{
			name:   "cmdline fallback",
			proc:   processInfo{PID: 3, Command: "node", CmdFull: "node /path/to/copilot-agent"},
			wantID: "copilot",
		},
		{
			name:    "no match",
			proc:    processInfo{PID: 4, Command: "vim", CmdFull: "vim main.go"},
			wantNil: true,
		},
		{
			name:   "aider match",
			proc:   processInfo{PID: 5, Command: "aider", CmdFull: "aider --model gpt-4"},
			wantID: "aider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := d.matchProcess(tt.proc)
			if tt.wantNil {
				if result != nil {
					t.Errorf("matchProcess() = %q, want nil", result.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("matchProcess() = nil, want %q", tt.wantID)
			}
			if result.ID != tt.wantID {
				t.Errorf("matchProcess().ID = %q, want %q", result.ID, tt.wantID)
			}
		})
	}
}

func TestMatchProcess_OnlyExactMatch(t *testing.T) {
	r := NewRegistry()
	cfg := config.DefaultConfig()
	cfg.Detection.OnlyExactProcessMatch = true
	d := NewDetector(r, cfg)

	// With exact mode, cmdline fallback should be disabled
	proc := processInfo{PID: 1, Command: "node", CmdFull: "node /path/to/copilot-agent"}
	result := d.matchProcess(proc)
	if result != nil {
		t.Errorf("matchProcess with OnlyExactProcessMatch should return nil for cmdline-only match, got %q", result.ID)
	}

	// But exact process name should still work
	proc2 := processInfo{PID: 2, Command: "claude", CmdFull: "claude"}
	result2 := d.matchProcess(proc2)
	if result2 == nil {
		t.Fatal("matchProcess with OnlyExactProcessMatch returned nil for exact match")
	}
	if result2.ID != "claude-code" {
		t.Errorf("got ID %q, want 'claude-code'", result2.ID)
	}
}

func TestScan_ReturnsResult(t *testing.T) {
	// Scan talks to real processes — verify it doesn't error
	r := NewRegistry()
	cfg := config.DefaultConfig()
	d := NewDetector(r, cfg)

	agents, err := d.Scan()
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}

	// We can't guarantee any agents are running, but the slice should not be nil on success
	_ = agents
}
