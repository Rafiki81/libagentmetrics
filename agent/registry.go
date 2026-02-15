package agent

import (
	"os"
	"path/filepath"
)

// Registry holds all known agent definitions.
type Registry struct {
	Agents []Info
}

// NewRegistry creates a registry populated with known AI coding agents.
func NewRegistry() *Registry {
	home, _ := os.UserHomeDir()
	return &Registry{
		Agents: []Info{
			{
				Name:           "Claude Code",
				ID:             "claude-code",
				ProcessNames:   []string{"claude"},
				LogPaths:       []string{filepath.Join(home, ".claude", "logs"), filepath.Join(home, ".claude.log")},
				Ports:          []int{},
				Description:    "Anthropic's Claude AI coding agent",
				DetectPatterns: []string{"claude", "claude-code"},
			},
			{
				Name:           "GitHub Copilot",
				ID:             "copilot",
				ProcessNames:   []string{"copilot-agent", "copilot"},
				LogPaths:       []string{filepath.Join(home, ".vscode", "extensions")},
				Ports:          []int{},
				Description:    "GitHub Copilot AI pair programmer",
				DetectPatterns: []string{"copilot", "github.copilot"},
			},
			{
				Name:           "OpenAI Codex CLI",
				ID:             "codex-cli",
				ProcessNames:   []string{"codex"},
				LogPaths:       []string{},
				Ports:          []int{},
				Description:    "OpenAI Codex CLI agent",
				DetectPatterns: []string{"codex", "openai-codex"},
			},
			{
				Name:           "Open Codex",
				ID:             "open-codex",
				ProcessNames:   []string{"open-codex", "ocodex"},
				LogPaths:       []string{},
				Ports:          []int{},
				Description:    "Open-source Codex alternative",
				DetectPatterns: []string{"open-codex", "ocodex"},
			},
			{
				Name:           "Aider",
				ID:             "aider",
				ProcessNames:   []string{"aider"},
				LogPaths:       []string{filepath.Join(home, ".aider.logs")},
				Ports:          []int{},
				Description:    "AI pair programming in your terminal",
				DetectPatterns: []string{"aider"},
			},
			{
				Name:           "Cody (Sourcegraph)",
				ID:             "cody",
				ProcessNames:   []string{"cody"},
				LogPaths:       []string{},
				Ports:          []int{},
				Description:    "Sourcegraph Cody AI assistant",
				DetectPatterns: []string{"cody", "sourcegraph.cody"},
			},
			{
				Name:           "Cursor",
				ID:             "cursor",
				ProcessNames:   []string{"Cursor", "cursor"},
				LogPaths:       []string{filepath.Join(home, ".cursor", "logs")},
				Ports:          []int{},
				Description:    "Cursor AI-powered code editor",
				DetectPatterns: []string{"cursor", "Cursor"},
			},
			{
				Name:           "Continue.dev",
				ID:             "continue",
				ProcessNames:   []string{"continue"},
				LogPaths:       []string{},
				Ports:          []int{65432},
				Description:    "Continue.dev open-source AI assistant",
				DetectPatterns: []string{"continue"},
			},
			{
				Name:           "Codel",
				ID:             "codel",
				ProcessNames:   []string{"codel"},
				LogPaths:       []string{},
				Ports:          []int{3000},
				Description:    "Autonomous AI coding agent",
				DetectPatterns: []string{"codel"},
			},
			{
				Name:           "MoltBot",
				ID:             "moltbot",
				ProcessNames:   []string{"moltbot", "molt"},
				LogPaths:       []string{},
				Ports:          []int{},
				Description:    "MoltBot AI coding assistant",
				DetectPatterns: []string{"moltbot", "molt"},
			},
			{
				Name:           "Windsurf (Codeium)",
				ID:             "windsurf",
				ProcessNames:   []string{"windsurf", "Windsurf"},
				LogPaths:       []string{filepath.Join(home, ".windsurf", "logs")},
				Ports:          []int{},
				Description:    "Codeium's Windsurf AI editor",
				DetectPatterns: []string{"windsurf", "Windsurf"},
			},
			{
				Name:           "Gemini CLI",
				ID:             "gemini-cli",
				ProcessNames:   []string{"gemini"},
				LogPaths:       []string{},
				Ports:          []int{},
				Description:    "Google Gemini CLI agent",
				DetectPatterns: []string{"gemini"},
			},
		},
	}
}

// FindByProcess returns the Info for the agent whose ProcessNames list contains
// processName, or nil if no match is found.
func (r *Registry) FindByProcess(processName string) *Info {
	for i, a := range r.Agents {
		for _, pname := range a.ProcessNames {
			if pname == processName {
				return &r.Agents[i]
			}
		}
	}
	return nil
}

// FindByCmdLine returns the Info for the first agent whose DetectPatterns
// appear as a word in cmdline, or nil if no match is found.
func (r *Registry) FindByCmdLine(cmdline string) *Info {
	for i, a := range r.Agents {
		for _, pattern := range a.DetectPatterns {
			if containsWord(cmdline, pattern) {
				return &r.Agents[i]
			}
		}
	}
	return nil
}

func containsWord(s, word string) bool {
	return len(s) > 0 && len(word) > 0 && stringContains(s, word)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
