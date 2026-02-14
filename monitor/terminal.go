package monitor

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// TerminalMonitor detects terminal commands spawned by agent processes.
type TerminalMonitor struct {
	mu         sync.Mutex
	history    map[string][]agent.TerminalCommand // agentID -> commands
	seenPIDs   map[int]bool                       // PIDs we've already seen
	maxHistory int
}

// NewTerminalMonitor creates a new terminal monitor.
func NewTerminalMonitor(maxHistory int) *TerminalMonitor {
	if maxHistory <= 0 {
		maxHistory = 50
	}
	return &TerminalMonitor{
		history:    make(map[string][]agent.TerminalCommand),
		seenPIDs:   make(map[int]bool),
		maxHistory: maxHistory,
	}
}

// Collect detects terminal commands spawned by an agent process.
func (tm *TerminalMonitor) Collect(a *agent.Instance) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if a.PID == 0 {
		return
	}

	// Find child processes that look like terminal commands
	children := getChildProcesses(a.PID)
	for _, child := range children {
		if tm.seenPIDs[child.pid] {
			continue
		}
		tm.seenPIDs[child.pid] = true

		cmd := agent.TerminalCommand{
			Command:   child.cmd,
			Timestamp: time.Now(),
			Category:  categorizeCommand(child.cmd),
		}

		tm.history[a.Info.ID] = append(tm.history[a.Info.ID], cmd)

		// Trim history
		if len(tm.history[a.Info.ID]) > tm.maxHistory {
			tm.history[a.Info.ID] = tm.history[a.Info.ID][len(tm.history[a.Info.ID])-tm.maxHistory:]
		}
	}

	// Populate the agent's terminal activity
	cmds := tm.history[a.Info.ID]
	a.Terminal.RecentCommands = cmds
	a.Terminal.TotalCommands = len(cmds)
}

type childProcess struct {
	pid int
	cmd string
}

// getChildProcesses finds child processes of a given PID.
func getChildProcesses(parentPID int) []childProcess {
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(parentPID))
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var children []childProcess
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		childPID, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue
		}

		cmdExec := exec.Command("ps", "-p", strconv.Itoa(childPID), "-o", "command=")
		cmdOut, err := cmdExec.Output()
		if err != nil {
			continue
		}

		cmdLine := strings.TrimSpace(string(cmdOut))
		if cmdLine == "" || isIgnoredProcess(cmdLine) {
			continue
		}

		children = append(children, childProcess{pid: childPID, cmd: cmdLine})

		// Recursively find grandchildren
		grandchildren := getChildProcesses(childPID)
		children = append(children, grandchildren...)
	}

	return children
}

// CategorizeCommand assigns a category to a terminal command.
func CategorizeCommand(cmd string) string {
	return categorizeCommand(cmd)
}

func categorizeCommand(cmd string) string {
	lower := strings.ToLower(cmd)

	if matchesAny(lower, "make", "go build", "npm run build", "cargo build",
		"mvn", "gradle", "cmake", "gcc", "g++", "clang", "rustc", "tsc",
		"webpack", "vite", "esbuild") {
		return "build"
	}

	if matchesAny(lower, "go test", "npm test", "pytest", "jest", "cargo test",
		"mvn test", "mocha", "vitest", "rspec", "phpunit") {
		return "test"
	}

	if matchesAny(lower, "npm install", "pip install", "go get", "cargo add",
		"brew install", "apt install", "yarn add", "pnpm add", "gem install",
		"go mod tidy") {
		return "install"
	}

	if matchesAny(lower, "git ") {
		return "git"
	}

	if matchesAny(lower, "go run", "node ", "python", "ruby ", "java ",
		"npm start", "npm run", "cargo run", "deno run") {
		return "run"
	}

	if matchesAny(lower, "eslint", "prettier", "gofmt", "black ", "ruff",
		"clippy", "golangci-lint", "rubocop") {
		return "lint"
	}

	if matchesAny(lower, "cat ", "less ", "grep ", "find ", "ls ", "mkdir ",
		"cp ", "mv ", "rm ", "touch ", "sed ", "awk ") {
		return "file"
	}

	return "other"
}

func matchesAny(s string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func isIgnoredProcess(cmd string) bool {
	ignored := []string{
		"ps -p", "pgrep", "/bin/sh", "/bin/zsh", "/bin/bash",
		"(zsh)", "(bash)", "(sh)", "fish",
	}
	lower := strings.ToLower(cmd)
	for _, ig := range ignored {
		if strings.Contains(lower, ig) {
			return true
		}
	}
	return false
}
