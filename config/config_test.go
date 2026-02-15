package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Check refresh interval
	if cfg.RefreshInterval.Duration() != 3*time.Second {
		t.Errorf("RefreshInterval = %v, want 3s", cfg.RefreshInterval.Duration())
	}

	// Alerts
	if !cfg.Alerts.Enabled {
		t.Error("Alerts should be enabled by default")
	}
	if cfg.Alerts.CPUWarning != 80 {
		t.Errorf("CPUWarning = %f, want 80", cfg.Alerts.CPUWarning)
	}
	if cfg.Alerts.CPUCritical != 95 {
		t.Errorf("CPUCritical = %f, want 95", cfg.Alerts.CPUCritical)
	}
	if cfg.Alerts.MemoryWarning != 500 {
		t.Errorf("MemoryWarning = %f, want 500", cfg.Alerts.MemoryWarning)
	}
	if cfg.Alerts.MemoryCritical != 1000 {
		t.Errorf("MemoryCritical = %f, want 1000", cfg.Alerts.MemoryCritical)
	}
	if cfg.Alerts.MaxAlerts != 100 {
		t.Errorf("MaxAlerts = %d, want 100", cfg.Alerts.MaxAlerts)
	}
	if cfg.Alerts.DailyBudgetUSD != 0 {
		t.Errorf("DailyBudgetUSD = %f, want 0", cfg.Alerts.DailyBudgetUSD)
	}
	if cfg.Alerts.MonthlyBudgetUSD != 0 {
		t.Errorf("MonthlyBudgetUSD = %f, want 0", cfg.Alerts.MonthlyBudgetUSD)
	}
	if cfg.Alerts.BudgetWarnPercent != 80 {
		t.Errorf("BudgetWarnPercent = %f, want 80", cfg.Alerts.BudgetWarnPercent)
	}
	if cfg.Alerts.BurnRateWarning != 2.0 {
		t.Errorf("BurnRateWarning = %f, want 2.0", cfg.Alerts.BurnRateWarning)
	}
	if cfg.Alerts.BurnRateCritical != 3.0 {
		t.Errorf("BurnRateCritical = %f, want 3.0", cfg.Alerts.BurnRateCritical)
	}

	// Security
	if !cfg.Security.Enabled {
		t.Error("Security should be enabled by default")
	}
	if len(cfg.Security.DangerousCommands) == 0 {
		t.Error("Security.DangerousCommands should not be empty")
	}
	if len(cfg.Security.SensitiveFiles) == 0 {
		t.Error("Security.SensitiveFiles should not be empty")
	}
	if cfg.Security.MassDeletionThreshold != 10 {
		t.Errorf("MassDeletionThreshold = %d, want 10", cfg.Security.MassDeletionThreshold)
	}

	// Detection
	if !cfg.Detection.SkipSystemProcesses {
		t.Error("SkipSystemProcesses should be true by default")
	}
	if len(cfg.Detection.IgnoreProcessPatterns) == 0 {
		t.Error("IgnoreProcessPatterns should not be empty")
	}

	// Display
	if !cfg.Display.ShowTokens {
		t.Error("ShowTokens should be true")
	}
	if !cfg.Display.ShowAlerts {
		t.Error("ShowAlerts should be true")
	}
	if !cfg.Display.ShowSecurity {
		t.Error("ShowSecurity should be true")
	}

	// LocalModels
	if !cfg.LocalModels.Enabled {
		t.Error("LocalModels should be enabled by default")
	}
}

func TestDuration_MarshalJSON(t *testing.T) {
	d := Duration(3 * time.Second)
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	if string(data) != `"3s"` {
		t.Errorf("MarshalJSON = %s, want \"3s\"", data)
	}
}

func TestDuration_MarshalJSON_Complex(t *testing.T) {
	d := Duration(1*time.Hour + 30*time.Minute + 15*time.Second)
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	if string(data) != `"1h30m15s"` {
		t.Errorf("MarshalJSON = %s, want \"1h30m15s\"", data)
	}
}

func TestDuration_UnmarshalJSON_String(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`"5s"`), &d)
	if err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if d.Duration() != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", d.Duration())
	}
}

func TestDuration_UnmarshalJSON_Float(t *testing.T) {
	var d Duration
	err := json.Unmarshal([]byte(`1000000000`), &d)
	if err != nil {
		t.Fatalf("UnmarshalJSON error: %v", err)
	}
	if d.Duration() != 1*time.Second {
		t.Errorf("Duration = %v, want 1s", d.Duration())
	}
}

func TestDuration_UnmarshalJSON_Invalid(t *testing.T) {
	var d Duration
	// Invalid string
	err := json.Unmarshal([]byte(`"not-a-duration"`), &d)
	if err == nil {
		t.Error("expected error for invalid duration string")
	}

	// Invalid type (bool)
	err = json.Unmarshal([]byte(`true`), &d)
	if err == nil {
		t.Error("expected error for bool value")
	}
}

func TestDuration_RoundTrip(t *testing.T) {
	original := Duration(2*time.Minute + 30*time.Second)
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var restored Duration
	err = json.Unmarshal(data, &restored)
	if err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if original != restored {
		t.Errorf("round-trip failed: original=%v, restored=%v", original.Duration(), restored.Duration())
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")
	cfg := DefaultConfig()
	cfg.Alerts.CPUWarning = 42
	cfg.Security.MaxEvents = 999

	// Save
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent: %v", err)
	}
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Load
	loaded := DefaultConfig()
	fileData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if err := json.Unmarshal(fileData, loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if loaded.Alerts.CPUWarning != 42 {
		t.Errorf("loaded CPUWarning = %f, want 42", loaded.Alerts.CPUWarning)
	}
	if loaded.Security.MaxEvents != 999 {
		t.Errorf("loaded MaxEvents = %d, want 999", loaded.Security.MaxEvents)
	}
}

func TestShouldIgnoreProcess(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name    string
		cmdline string
		want    bool
	}{
		{"apple process", "com.apple.WindowServer", true},
		{"system library", "/System/Library/CoreServices/Finder.app", true},
		{"libexec", "/usr/libexec/some_daemon", true},
		{"normal process", "/usr/local/bin/claude", false},
		{"user process", "node /home/user/app.js", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ShouldIgnoreProcess(tt.cmdline)
			if got != tt.want {
				t.Errorf("ShouldIgnoreProcess(%q) = %v, want %v", tt.cmdline, got, tt.want)
			}
		})
	}
}

func TestShouldIgnoreProcess_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Detection.SkipSystemProcesses = false
	// With skip disabled, nothing should be ignored
	got := cfg.ShouldIgnoreProcess("com.apple.WindowServer")
	if got != false {
		t.Error("ShouldIgnoreProcess should return false when SkipSystemProcesses is false")
	}
}

func TestShouldIgnorePath(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name string
		path string
		want bool
	}{
		{"Library", "/Library/Frameworks/something", true},
		{"System", "/System/Library/CoreServices", true},
		{"private", "/private/var/folders", true},
		{"usr", "/usr/local/bin", true},
		{"home", "/Users/dev/projects", false},
		{"root project", "/opt/myapp", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ShouldIgnorePath(tt.path)
			if got != tt.want {
				t.Errorf("ShouldIgnorePath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsSystemProcess(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name    string
		cmdline string
		want    bool
	}{
		{"System prefix", "/System/Library/CoreServices/launchd", true},
		{"libexec", "/usr/libexec/some_daemon", true},
		{"sbin", "/usr/sbin/sshd", true},
		{"Apple lib", "/Library/Apple/usr/bin/something", true},
		{"user binary", "/usr/local/bin/node", false},
		{"home", "/Users/dev/bin/myapp", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.IsSystemProcess(tt.cmdline)
			if got != tt.want {
				t.Errorf("IsSystemProcess(%q) = %v, want %v", tt.cmdline, got, tt.want)
			}
		})
	}
}

func TestIsAgentDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Detection.DisabledAgents = []string{"copilot", "Aider"}

	tests := []struct {
		agentID string
		want    bool
	}{
		{"copilot", true},
		{"Copilot", true}, // case-insensitive
		{"COPILOT", true}, // case-insensitive
		{"aider", true},   // case-insensitive match
		{"Aider", true},
		{"claude-code", false},
		{"cursor", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.agentID, func(t *testing.T) {
			got := cfg.IsAgentDisabled(tt.agentID)
			if got != tt.want {
				t.Errorf("IsAgentDisabled(%q) = %v, want %v", tt.agentID, got, tt.want)
			}
		})
	}
}

func TestIsAgentDisabled_EmptyList(t *testing.T) {
	cfg := DefaultConfig()
	// Default has empty disabled list
	if cfg.IsAgentDisabled("copilot") {
		t.Error("no agent should be disabled with empty list")
	}
}

func TestConfigPath(t *testing.T) {
	path := ConfigPath()
	if path == "" {
		t.Fatal("ConfigPath() returned empty")
	}
	// Should end with config.json
	if filepath.Base(path) != "config.json" {
		t.Errorf("ConfigPath() base = %q, want config.json", filepath.Base(path))
	}
	// Should contain .agentmetrics
	dir := filepath.Dir(path)
	if filepath.Base(dir) != ".agentmetrics" {
		t.Errorf("ConfigPath() parent dir = %q, want .agentmetrics", filepath.Base(dir))
	}
}

func TestConfigJSON_RoundTrip(t *testing.T) {
	cfg := DefaultConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var cfg2 Config
	if err := json.Unmarshal(data, &cfg2); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if cfg2.RefreshInterval.Duration() != cfg.RefreshInterval.Duration() {
		t.Errorf("RefreshInterval mismatch: %v vs %v", cfg2.RefreshInterval.Duration(), cfg.RefreshInterval.Duration())
	}
	if cfg2.Alerts.CPUWarning != cfg.Alerts.CPUWarning {
		t.Errorf("CPUWarning mismatch")
	}
	if cfg2.Security.Enabled != cfg.Security.Enabled {
		t.Errorf("Security.Enabled mismatch")
	}
	if cfg2.Display.ShowTokens != cfg.Display.ShowTokens {
		t.Errorf("Display.ShowTokens mismatch")
	}
}
