package monitor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// TokenMonitor collects token usage from multiple sources per agent.
type TokenMonitor struct {
	mu sync.Mutex
	// Accumulated token data per agent ID
	data map[string]*agent.TokenMetrics
	// Network bytes tracking per PID for estimation
	prevBytes map[int]int64
	// Copilot log: last read offset per file
	copilotLogOffsets map[string]int64
	// Claude: last read offset per JSONL file
	claudeLogOffsets map[string]int64
	// Aider: last read offset per history file
	aiderLogOffsets map[string]int64
}

// NewTokenMonitor creates a new token monitor.
func NewTokenMonitor() *TokenMonitor {
	return &TokenMonitor{
		data:              make(map[string]*agent.TokenMetrics),
		prevBytes:         make(map[int]int64),
		copilotLogOffsets: make(map[string]int64),
		claudeLogOffsets:  make(map[string]int64),
		aiderLogOffsets:   make(map[string]int64),
	}
}

// Collect gathers token metrics for all detected agents. It dispatches to
// agent-specific collectors (Copilot logs, Claude JSONL, Cursor DB, Aider
// history) and falls back to network-based estimation for unknown agents.
func (tm *TokenMonitor) Collect(agents []agent.Instance) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for i := range agents {
		a := &agents[i]
		id := a.Info.ID

		// Initialize if new agent
		if _, ok := tm.data[id]; !ok {
			tm.data[id] = &agent.TokenMetrics{}
		}

		switch id {
		case "copilot":
			tm.collectCopilot(a)
		case "claude-code":
			tm.collectClaude(a)
		case "cursor":
			tm.collectCursor(a)
		case "aider":
			tm.collectAider(a)
		default:
			tm.collectFromNetwork(a)
		}

		// Calculate cost based on model and tokens
		m := tm.data[id]
		m.EstCost = EstimateCost(m.LastModel, m.InputTokens, m.OutputTokens)

		// Copy metrics to agent instance
		a.Tokens = *m
	}
}

// GetMetrics returns a copy of metrics for a specific agent.
func (tm *TokenMonitor) GetMetrics(agentID string) agent.TokenMetrics {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if m, ok := tm.data[agentID]; ok {
		return *m
	}
	return agent.TokenMetrics{}
}

// ---------- Copilot: parse VS Code extension logs ----------

var copilotReqRe = regexp.MustCompile(
	`ccreq:\w+\.copilotmd \| (success|error) \| (\S+)\s*->\s*(\S+) \| (\d+)ms`,
)

func (tm *TokenMonitor) collectCopilot(a *agent.Instance) {
	home, _ := os.UserHomeDir()
	m := tm.data[a.Info.ID]

	logsBase := filepath.Join(home, "Library", "Application Support", "Code", "logs")
	logDirs, _ := filepath.Glob(filepath.Join(logsBase, "*"))
	if len(logDirs) == 0 {
		tm.collectFromNetwork(a)
		return
	}

	sort.Strings(logDirs)
	latestDir := logDirs[len(logDirs)-1]

	chatLogs, _ := filepath.Glob(filepath.Join(latestDir, "window*", "exthost", "GitHub.copilot-chat", "GitHub Copilot Chat.log"))

	if len(chatLogs) == 0 {
		tm.collectFromNetwork(a)
		return
	}

	foundRequests := false
	for _, logPath := range chatLogs {
		count := tm.parseCopilotLog(logPath, m)
		if count > 0 {
			foundRequests = true
		}
	}

	if foundRequests {
		m.Source = agent.TokenSourceLog
	} else if m.Source == "" {
		tm.collectFromNetwork(a)
	}
}

func (tm *TokenMonitor) parseCopilotLog(logPath string, m *agent.TokenMetrics) int {
	f, err := os.Open(logPath)
	if err != nil {
		return 0
	}
	defer f.Close()

	offset, exists := tm.copilotLogOffsets[logPath]
	if exists {
		f.Seek(offset, 0)
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	newRequests := 0

	for scanner.Scan() {
		line := scanner.Text()

		match := copilotReqRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		model := match[2]
		latencyStr := match[4]
		latency, _ := strconv.Atoi(latencyStr)

		m.RequestCount++
		m.LastModel = model
		m.LastRequestAt = time.Now()
		newRequests++

		if latency > 0 {
			if m.AvgLatencyMs == 0 {
				m.AvgLatencyMs = int64(latency)
			} else {
				m.AvgLatencyMs = (m.AvgLatencyMs*int64(m.RequestCount-1) + int64(latency)) / int64(m.RequestCount)
			}
		}

		estimatedInput := int64(300)
		estimatedOutput := int64(200)
		if strings.Contains(model, "gpt-4") || strings.Contains(model, "claude") {
			estimatedInput = 800
			estimatedOutput = 400
		}

		m.InputTokens += estimatedInput
		m.OutputTokens += estimatedOutput
		m.TotalTokens = m.InputTokens + m.OutputTokens
	}

	pos, _ := f.Seek(0, 1)
	tm.copilotLogOffsets[logPath] = pos

	if m.RequestCount > 0 && !m.LastRequestAt.IsZero() {
		elapsed := time.Since(m.LastRequestAt).Seconds()
		if elapsed < 60 && elapsed > 0 {
			m.TokensPerSec = float64(m.OutputTokens) / float64(m.RequestCount) / (elapsed + 0.5)
		} else {
			m.TokensPerSec = 0
		}
	}

	return newRequests
}

// ---------- Claude Code: parse conversation JSONL files ----------

func (tm *TokenMonitor) collectClaude(a *agent.Instance) {
	home, _ := os.UserHomeDir()
	m := tm.data[a.Info.ID]

	claudeDir := filepath.Join(home, ".claude")
	if _, err := os.Stat(claudeDir); os.IsNotExist(err) {
		tm.collectFromNetwork(a)
		return
	}

	files, _ := filepath.Glob(filepath.Join(claudeDir, "projects", "*", "conversations", "*.jsonl"))
	if len(files) == 0 {
		files, _ = filepath.Glob(filepath.Join(claudeDir, "conversations", "*.jsonl"))
	}

	if len(files) == 0 {
		tm.collectFromNetwork(a)
		return
	}

	foundTokens := false
	for _, f := range files {
		count := tm.parseClaudeJSONL(f, m)
		if count > 0 {
			foundTokens = true
		}
	}

	if foundTokens {
		m.Source = agent.TokenSourceLog
	} else if m.Source == "" {
		tm.collectFromNetwork(a)
	}
}

type claudeMessage struct {
	Type    string `json:"type"`
	Message struct {
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
		Model string `json:"model"`
	} `json:"message"`
}

func (tm *TokenMonitor) parseClaudeJSONL(path string, m *agent.TokenMetrics) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()

	offset, exists := tm.claudeLogOffsets[path]
	if exists {
		f.Seek(offset, 0)
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	count := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var msg claudeMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		if msg.Type == "assistant" && msg.Message.Usage.InputTokens > 0 {
			m.InputTokens += msg.Message.Usage.InputTokens
			m.OutputTokens += msg.Message.Usage.OutputTokens
			m.TotalTokens = m.InputTokens + m.OutputTokens
			m.RequestCount++
			m.LastRequestAt = time.Now()
			if msg.Message.Model != "" {
				m.LastModel = msg.Message.Model
			}
			count++
		}
	}

	pos, _ := f.Seek(0, 1)
	tm.claudeLogOffsets[path] = pos

	if m.RequestCount > 0 && !m.LastRequestAt.IsZero() {
		elapsed := time.Since(m.LastRequestAt).Seconds()
		if elapsed < 60 && elapsed > 0 {
			m.TokensPerSec = float64(m.OutputTokens) / float64(m.RequestCount) / (elapsed + 0.5)
		} else {
			m.TokensPerSec = 0
		}
	}

	return count
}

// ---------- Cursor: parse SQLite DB ----------

func (tm *TokenMonitor) collectCursor(a *agent.Instance) {
	home, _ := os.UserHomeDir()
	m := tm.data[a.Info.ID]

	dbPath := filepath.Join(home, "Library", "Application Support", "Cursor", "User", "globalStorage", "state.vscdb")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		tm.collectFromNetwork(a)
		return
	}

	if tm.parseCursorDB(dbPath, m) {
		m.Source = agent.TokenSourceDB
		return
	}

	logsBase := filepath.Join(home, "Library", "Application Support", "Cursor", "logs")
	logDirs, _ := filepath.Glob(filepath.Join(logsBase, "*"))
	if len(logDirs) > 0 {
		sort.Strings(logDirs)
		latestDir := logDirs[len(logDirs)-1]
		chatLogs, _ := filepath.Glob(filepath.Join(latestDir, "window*", "exthost", "*", "*.log"))
		for _, logPath := range chatLogs {
			tm.parseCopilotLog(logPath, m)
		}
	}

	if m.RequestCount == 0 {
		tm.collectFromNetwork(a)
	}
}

func (tm *TokenMonitor) parseCursorDB(dbPath string, m *agent.TokenMetrics) bool {
	cmd := exec.Command("sqlite3", dbPath,
		"SELECT value FROM cursorDiskKV WHERE key LIKE 'composerData:%' ORDER BY length(value) DESC LIMIT 10")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(out), "\n")
	totalRequests := 0
	var lastModel string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}

		if usage, ok := data["usageData"]; ok {
			if usageMap, ok := usage.(map[string]interface{}); ok && len(usageMap) > 0 {
				if input, ok := usageMap["inputTokens"]; ok {
					if v, ok := input.(float64); ok {
						m.InputTokens += int64(v)
					}
				}
				if output, ok := usageMap["outputTokens"]; ok {
					if v, ok := output.(float64); ok {
						m.OutputTokens += int64(v)
					}
				}
			}
		}

		if mc, ok := data["modelConfig"]; ok {
			if mcMap, ok := mc.(map[string]interface{}); ok {
				if mn, ok := mcMap["modelName"]; ok {
					if name, ok := mn.(string); ok && name != "" && name != "default,default,default,default" {
						lastModel = name
					}
				}
			}
		}

		if convMap, ok := data["conversationMap"]; ok {
			if cm, ok := convMap.(map[string]interface{}); ok {
				totalRequests += len(cm)
			}
		}
	}

	if totalRequests > 0 || m.InputTokens > 0 {
		m.RequestCount = totalRequests
		m.TotalTokens = m.InputTokens + m.OutputTokens
		if lastModel != "" {
			m.LastModel = lastModel
		} else {
			m.LastModel = "cursor"
		}
		m.LastRequestAt = time.Now()

		if m.InputTokens == 0 && totalRequests > 0 {
			m.InputTokens = int64(totalRequests) * 500
			m.OutputTokens = int64(totalRequests) * 300
			m.TotalTokens = m.InputTokens + m.OutputTokens
			m.Source = agent.TokenSourceEstimated
		}
		return true
	}

	return false
}

// ---------- Aider: parse chat history ----------

var aiderTokenRe = regexp.MustCompile(
	`Tokens:\s*([\d.]+)k?\s*sent,\s*([\d.]+)k?\s*received`,
)

func (tm *TokenMonitor) collectAider(a *agent.Instance) {
	m := tm.data[a.Info.ID]

	searchPaths := []string{}
	if a.WorkDir != "" {
		searchPaths = append(searchPaths,
			filepath.Join(a.WorkDir, ".aider.chat.history.md"),
			filepath.Join(a.WorkDir, ".aider.logs", "aider.log"),
		)
	}

	home, _ := os.UserHomeDir()
	searchPaths = append(searchPaths,
		filepath.Join(home, ".aider.chat.history.md"),
		filepath.Join(home, ".aider.logs", "aider.log"),
	)

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			if tm.parseAiderHistory(path, m) {
				m.Source = agent.TokenSourceLog
				return
			}
		}
	}

	tm.collectFromNetwork(a)
}

func (tm *TokenMonitor) parseAiderHistory(path string, m *agent.TokenMetrics) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	offset, exists := tm.aiderLogOffsets[path]
	if exists {
		f.Seek(offset, 0)
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	found := false

	for scanner.Scan() {
		line := scanner.Text()
		match := aiderTokenRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		sent := parseTokenCount(match[1])
		recv := parseTokenCount(match[2])

		m.InputTokens += sent
		m.OutputTokens += recv
		m.TotalTokens = m.InputTokens + m.OutputTokens
		m.RequestCount++
		m.LastRequestAt = time.Now()
		m.LastModel = "aider"
		found = true
	}

	pos, _ := f.Seek(0, 1)
	tm.aiderLogOffsets[path] = pos

	return found
}

func parseTokenCount(s string) int64 {
	s = strings.TrimSpace(s)
	multiplier := int64(1)
	if strings.HasSuffix(s, "k") {
		multiplier = 1000
		s = strings.TrimSuffix(s, "k")
	} else if strings.HasSuffix(s, "M") {
		multiplier = 1000000
		s = strings.TrimSuffix(s, "M")
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return int64(f * float64(multiplier))
}

// ---------- Network-based estimation ----------

func (tm *TokenMonitor) collectFromNetwork(a *agent.Instance) {
	m := tm.data[a.Info.ID]

	bytes := getNetworkBytesForPID(a.PID)

	if bytes <= 0 {
		return
	}

	prevBytes := tm.prevBytes[a.PID]
	delta := bytes - prevBytes
	tm.prevBytes[a.PID] = bytes

	if delta <= 0 || prevBytes == 0 {
		return
	}

	estimatedTokens := delta / 4

	m.OutputTokens += estimatedTokens
	m.TotalTokens = m.InputTokens + m.OutputTokens
	m.LastRequestAt = time.Now()

	if m.Source == "" {
		m.Source = agent.TokenSourceNetwork
	}

	m.TokensPerSec = float64(estimatedTokens) / 2.0
}

func getNetworkBytesForPID(pid int) int64 {
	cmd := exec.Command("nettop", "-p", strconv.Itoa(pid), "-L", "1", "-J", "bytes_in,bytes_out", "-x")
	cmd.Env = append(os.Environ(), "TERM=dumb")
	out, err := cmd.Output()
	if err != nil {
		return estimateFromLsof(pid)
	}

	lines := strings.Split(string(out), "\n")
	var totalBytes int64
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		for _, field := range fields {
			if n, err := strconv.ParseInt(field, 10, 64); err == nil && n > 0 {
				totalBytes += n
			}
		}
	}

	return totalBytes
}

func estimateFromLsof(pid int) int64 {
	cmd := exec.Command("lsof", "-i", "-n", "-P", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	established := 0
	for _, line := range lines {
		if strings.Contains(line, "ESTABLISHED") {
			established++
		}
	}

	return int64(established * 500)
}

// FormatTokenCount formats a token count for display.
// Returns "—" for zero/negative, "X.Xk" for thousands,
// "X.XM" for millions, or the raw number for smaller values.
func FormatTokenCount(count int64) string {
	if count <= 0 {
		return "—"
	}
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fk", float64(count)/1000)
	}
	return strconv.FormatInt(count, 10)
}

// FormatTokensPerSec formats a tokens-per-second rate for display.
// Returns "—" for zero/negative, "X.Xk/s" for >= 1000, or "X/s" otherwise.
func FormatTokensPerSec(tps float64) string {
	if tps <= 0 {
		return "—"
	}
	if tps >= 1000 {
		return fmt.Sprintf("%.1fk/s", tps/1000)
	}
	return fmt.Sprintf("%.0f/s", tps)
}
