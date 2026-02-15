package agent

import "testing"

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry() returned nil")
	}
	if len(r.Agents) != 12 {
		t.Errorf("NewRegistry() has %d agents, want 12", len(r.Agents))
	}
}

func TestNewRegistry_AgentIDs(t *testing.T) {
	r := NewRegistry()
	expectedIDs := []string{
		"claude-code", "copilot", "codex-cli", "open-codex",
		"aider", "cody", "cursor", "continue",
		"codel", "moltbot", "windsurf", "gemini-cli",
	}

	ids := make(map[string]bool)
	for _, a := range r.Agents {
		ids[a.ID] = true
	}

	for _, id := range expectedIDs {
		if !ids[id] {
			t.Errorf("registry missing agent ID %q", id)
		}
	}
}

func TestNewRegistry_AgentFields(t *testing.T) {
	r := NewRegistry()
	for _, a := range r.Agents {
		if a.Name == "" {
			t.Errorf("agent %q has empty Name", a.ID)
		}
		if a.ID == "" {
			t.Error("agent has empty ID")
		}
		if len(a.ProcessNames) == 0 {
			t.Errorf("agent %q has no ProcessNames", a.ID)
		}
		if a.Description == "" {
			t.Errorf("agent %q has empty Description", a.ID)
		}
		if len(a.DetectPatterns) == 0 {
			t.Errorf("agent %q has no DetectPatterns", a.ID)
		}
	}
}

func TestFindByProcess(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		process  string
		wantID   string
		wantNil  bool
	}{
		{"claude", "claude", "claude-code", false},
		{"copilot-agent", "copilot-agent", "copilot", false},
		{"codex", "codex", "codex-cli", false},
		{"aider", "aider", "aider", false},
		{"Cursor uppercase", "Cursor", "cursor", false},
		{"cursor lowercase", "cursor", "cursor", false},
		{"windsurf", "windsurf", "windsurf", false},
		{"gemini", "gemini", "gemini-cli", false},
		{"moltbot", "moltbot", "moltbot", false},
		{"molt", "molt", "moltbot", false},
		{"cody", "cody", "cody", false},
		{"continue", "continue", "continue", false},
		{"codel", "codel", "codel", false},
		{"unknown", "unknown-process", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.FindByProcess(tt.process)
			if tt.wantNil {
				if result != nil {
					t.Errorf("FindByProcess(%q) = %q, want nil", tt.process, result.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("FindByProcess(%q) = nil, want %q", tt.process, tt.wantID)
			}
			if result.ID != tt.wantID {
				t.Errorf("FindByProcess(%q).ID = %q, want %q", tt.process, result.ID, tt.wantID)
			}
		})
	}
}

func TestFindByCmdLine(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name    string
		cmdLine string
		wantID  string
		wantNil bool
	}{
		{"claude in path", "/usr/local/bin/claude --config foo", "claude-code", false},
		{"copilot in args", "--extension github.copilot --port 8080", "copilot", false},
		{"aider command", "python3 -m aider --model gpt-4", "aider", false},
		{"cursor pattern", "Cursor Helper (Renderer)", "cursor", false},
		{"gemini cmd", "gemini generate code", "gemini-cli", false},
		{"no match", "vim main.go", "", true},
		{"empty", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := r.FindByCmdLine(tt.cmdLine)
			if tt.wantNil {
				if result != nil {
					t.Errorf("FindByCmdLine(%q) = %q, want nil", tt.cmdLine, result.ID)
				}
				return
			}
			if result == nil {
				t.Fatalf("FindByCmdLine(%q) = nil, want %q", tt.cmdLine, tt.wantID)
			}
			if result.ID != tt.wantID {
				t.Errorf("FindByCmdLine(%q).ID = %q, want %q", tt.cmdLine, result.ID, tt.wantID)
			}
		})
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		s, word string
		want    bool
	}{
		{"hello world", "world", true},
		{"hello world", "worl", true},
		{"hello", "hello", true},
		{"", "hello", false},
		{"hello", "", false},
		{"", "", false},
		{"abc", "abcdef", false},
	}

	for _, tt := range tests {
		got := containsWord(tt.s, tt.word)
		if got != tt.want {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.s, tt.word, got, tt.want)
		}
	}
}

func TestStringContains(t *testing.T) {
	tests := []struct {
		s, substr string
		want      bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello", "hello", true},
		{"hello", "hellox", false},
		{"abc", "abcdef", false},
		{"test string", "st s", true},
		{"a", "a", true},
		{"a", "b", false},
	}

	for _, tt := range tests {
		got := stringContains(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("stringContains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}

func TestFindByProcess_ReturnsPointerToRegistrySlice(t *testing.T) {
	r := NewRegistry()
	a1 := r.FindByProcess("claude")
	a2 := r.FindByProcess("claude")

	if a1 != a2 {
		t.Error("FindByProcess should return pointer to same Info in registry")
	}
}

func TestFindByCmdLine_ReturnsPointerToRegistrySlice(t *testing.T) {
	r := NewRegistry()
	a1 := r.FindByCmdLine("/path/to/claude --help")
	a2 := r.FindByCmdLine("running claude-code here")

	if a1 == nil || a2 == nil {
		t.Fatal("FindByCmdLine returned nil for claude pattern")
	}
	// Both should point to the same agent entry
	if a1.ID != a2.ID {
		t.Errorf("expected same agent ID, got %q and %q", a1.ID, a2.ID)
	}
}

func TestRegistryWithCustomAgents(t *testing.T) {
	r := &Registry{
		Agents: []Info{
			{
				Name:           "Test Agent",
				ID:             "test-agent",
				ProcessNames:   []string{"testagent", "ta"},
				DetectPatterns: []string{"testagent", "test-agent"},
			},
		},
	}

	found := r.FindByProcess("ta")
	if found == nil {
		t.Fatal("FindByProcess('ta') returned nil")
	}
	if found.ID != "test-agent" {
		t.Errorf("found.ID = %q, want 'test-agent'", found.ID)
	}

	found = r.FindByCmdLine("run testagent --flag")
	if found == nil {
		t.Fatal("FindByCmdLine returned nil")
	}
	if found.ID != "test-agent" {
		t.Errorf("found.ID = %q, want 'test-agent'", found.ID)
	}
}
