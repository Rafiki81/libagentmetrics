package monitor

import (
	"bufio"
	"context"
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

const (
	tokenCommandTimeout     = 3 * time.Second
	tokenStateTTL           = 24 * time.Hour
	tokenPruneCheckInterval = 5 * time.Minute
)

const (
	tokenErrHomeDir     = "home_dir"
	tokenErrCopilotLog  = "copilot_log"
	tokenErrClaudeJSONL = "claude_jsonl"
	tokenErrCursorDB    = "cursor_db"
	tokenErrAiderLog    = "aider_log"
	tokenErrNetwork     = "network"
)

// MonitorErrorStats represents aggregated operational errors for a monitor source.
type MonitorErrorStats struct {
	Count     int       `json:"count"`
	LastError string    `json:"last_error"`
	LastAt    time.Time `json:"last_at"`
}

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
	// Last seen timestamps for path-based offsets
	copilotLogSeen map[string]time.Time
	claudeLogSeen  map[string]time.Time
	aiderLogSeen   map[string]time.Time
	// Last seen timestamps for PID-based network state
	prevBytesSeen map[int]time.Time
	// Last state pruning time
	lastPruneAt time.Time
	// Error observability state per source
	errorStats map[string]MonitorErrorStats
}

func (tm *TokenMonitor) ensureInit() {
	if tm.data == nil {
		tm.data = make(map[string]*agent.TokenMetrics)
	}
	if tm.prevBytes == nil {
		tm.prevBytes = make(map[int]int64)
	}
	if tm.copilotLogOffsets == nil {
		tm.copilotLogOffsets = make(map[string]int64)
	}
	if tm.claudeLogOffsets == nil {
		tm.claudeLogOffsets = make(map[string]int64)
	}
	if tm.aiderLogOffsets == nil {
		tm.aiderLogOffsets = make(map[string]int64)
	}
	if tm.copilotLogSeen == nil {
		tm.copilotLogSeen = make(map[string]time.Time)
	}
	if tm.claudeLogSeen == nil {
		tm.claudeLogSeen = make(map[string]time.Time)
	}
	if tm.aiderLogSeen == nil {
		tm.aiderLogSeen = make(map[string]time.Time)
	}
	if tm.prevBytesSeen == nil {
		tm.prevBytesSeen = make(map[int]time.Time)
	}
	if tm.errorStats == nil {
		tm.errorStats = make(map[string]MonitorErrorStats)
	}
}

// NewTokenMonitor creates a new token monitor.
func NewTokenMonitor() *TokenMonitor {
	return &TokenMonitor{
		data:              make(map[string]*agent.TokenMetrics),
		prevBytes:         make(map[int]int64),
		copilotLogOffsets: make(map[string]int64),
		claudeLogOffsets:  make(map[string]int64),
		aiderLogOffsets:   make(map[string]int64),
		copilotLogSeen:    make(map[string]time.Time),
		claudeLogSeen:     make(map[string]time.Time),
		aiderLogSeen:      make(map[string]time.Time),
		prevBytesSeen:     make(map[int]time.Time),
		errorStats:        make(map[string]MonitorErrorStats),
	}
}

// Collect gathers token metrics for all detected agents. It dispatches to
// agent-specific collectors (Copilot logs, Claude JSONL, Cursor DB, Aider
// history) and falls back to network-based estimation for unknown agents.
func (tm *TokenMonitor) Collect(agents []agent.Instance) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.ensureInit()

	now := time.Now()
	if tm.lastPruneAt.IsZero() || now.Sub(tm.lastPruneAt) >= tokenPruneCheckInterval {
		tm.pruneState(agents, now)
		tm.lastPruneAt = now
	}

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
	tm.ensureInit()
	if m, ok := tm.data[agentID]; ok {
		return *m
	}
	return agent.TokenMetrics{}
}

// GetErrorStats returns a snapshot of operational errors per data source.
func (tm *TokenMonitor) GetErrorStats() map[string]MonitorErrorStats {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.ensureInit()

	stats := make(map[string]MonitorErrorStats, len(tm.errorStats))
	for k, v := range tm.errorStats {
		stats[k] = v
	}
	return stats
}

func (tm *TokenMonitor) recordError(source string, err error) {
	if err == nil {
		return
	}
	tm.ensureInit()

	stat := tm.errorStats[source]
	stat.Count++
	stat.LastError = err.Error()
	stat.LastAt = time.Now()
	tm.errorStats[source] = stat
}

// ---------- Copilot: parse VS Code extension logs ----------

var copilotReqRe = regexp.MustCompile(
	`ccreq:\w+\.copilotmd \| (success|error) \| (\S+)\s*->\s*(\S+) \| (\d+)ms`,
)

func (tm *TokenMonitor) collectCopilot(a *agent.Instance) {
	home, err := os.UserHomeDir()
	if err != nil {
		tm.recordError(tokenErrHomeDir, err)
		tm.collectFromNetwork(a)
		return
	}
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
		tm.recordError(tokenErrCopilotLog, err)
		return 0
	}
	defer f.Close()
	tm.copilotLogSeen[logPath] = time.Now()

	offset, exists := tm.copilotLogOffsets[logPath]
	if exists {
		if _, err := f.Seek(offset, 0); err != nil {
			tm.recordError(tokenErrCopilotLog, err)
		}
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

	pos, err := f.Seek(0, 1)
	if err != nil {
		tm.recordError(tokenErrCopilotLog, err)
	} else {
		tm.copilotLogOffsets[logPath] = pos
	}

	if err := scanner.Err(); err != nil {
		tm.recordError(tokenErrCopilotLog, err)
	}

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
	home, err := os.UserHomeDir()
	if err != nil {
		tm.recordError(tokenErrHomeDir, err)
		tm.collectFromNetwork(a)
		return
	}
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
		tm.recordError(tokenErrClaudeJSONL, err)
		return 0
	}
	defer f.Close()
	tm.claudeLogSeen[path] = time.Now()

	offset, exists := tm.claudeLogOffsets[path]
	if exists {
		if _, err := f.Seek(offset, 0); err != nil {
			tm.recordError(tokenErrClaudeJSONL, err)
		}
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

	pos, err := f.Seek(0, 1)
	if err != nil {
		tm.recordError(tokenErrClaudeJSONL, err)
	} else {
		tm.claudeLogOffsets[path] = pos
	}

	if err := scanner.Err(); err != nil {
		tm.recordError(tokenErrClaudeJSONL, err)
	}

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
	home, err := os.UserHomeDir()
	if err != nil {
		tm.recordError(tokenErrHomeDir, err)
		tm.collectFromNetwork(a)
		return
	}
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
	ctx, cancel := context.WithTimeout(context.Background(), tokenCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sqlite3", dbPath,
		"SELECT value FROM cursorDiskKV WHERE key LIKE 'composerData:%' ORDER BY length(value) DESC LIMIT 10")
	out, err := cmd.Output()
	if err != nil {
		tm.recordError(tokenErrCursorDB, err)
		return false
	}

	lines := strings.Split(string(out), "\n")
	parsed := parseCursorDBLines(lines)

	if parsed.RequestCount > 0 || parsed.InputTokens > 0 || parsed.OutputTokens > 0 {
		m.InputTokens = parsed.InputTokens
		m.OutputTokens = parsed.OutputTokens
		m.RequestCount = parsed.RequestCount
		m.TotalTokens = m.InputTokens + m.OutputTokens
		if parsed.LastModel != "" {
			m.LastModel = parsed.LastModel
		} else {
			m.LastModel = "cursor"
		}
		m.LastRequestAt = time.Now()

		if m.InputTokens == 0 && m.RequestCount > 0 {
			m.InputTokens = int64(m.RequestCount) * 500
			m.OutputTokens = int64(m.RequestCount) * 300
			m.TotalTokens = m.InputTokens + m.OutputTokens
			m.Source = agent.TokenSourceEstimated
		}
		return true
	}

	return false
}

type cursorDBParseResult struct {
	InputTokens  int64
	OutputTokens int64
	RequestCount int
	LastModel    string
}

func parseCursorDBLines(lines []string) cursorDBParseResult {
	result := cursorDBParseResult{}

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
						result.InputTokens += int64(v)
					}
				}
				if output, ok := usageMap["outputTokens"]; ok {
					if v, ok := output.(float64); ok {
						result.OutputTokens += int64(v)
					}
				}
			}
		}

		if mc, ok := data["modelConfig"]; ok {
			if mcMap, ok := mc.(map[string]interface{}); ok {
				if mn, ok := mcMap["modelName"]; ok {
					if name, ok := mn.(string); ok && name != "" && name != "default,default,default,default" {
						result.LastModel = name
					}
				}
			}
		}

		if convMap, ok := data["conversationMap"]; ok {
			if cm, ok := convMap.(map[string]interface{}); ok {
				result.RequestCount += len(cm)
			}
		}
	}

	return result
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

	home, err := os.UserHomeDir()
	if err != nil {
		tm.recordError(tokenErrHomeDir, err)
		tm.collectFromNetwork(a)
		return
	}
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
		tm.recordError(tokenErrAiderLog, err)
		return false
	}
	defer f.Close()
	tm.aiderLogSeen[path] = time.Now()

	offset, exists := tm.aiderLogOffsets[path]
	if exists {
		if _, err := f.Seek(offset, 0); err != nil {
			tm.recordError(tokenErrAiderLog, err)
		}
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

	pos, err := f.Seek(0, 1)
	if err != nil {
		tm.recordError(tokenErrAiderLog, err)
	} else {
		tm.aiderLogOffsets[path] = pos
	}

	if err := scanner.Err(); err != nil {
		tm.recordError(tokenErrAiderLog, err)
	}

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

	bytes, err := getNetworkBytesForPID(a.PID)
	if err != nil {
		tm.recordError(tokenErrNetwork, err)
	}

	if bytes <= 0 {
		return
	}

	prevBytes := tm.prevBytes[a.PID]
	delta := bytes - prevBytes
	tm.prevBytes[a.PID] = bytes
	tm.prevBytesSeen[a.PID] = time.Now()

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

func getNetworkBytesForPID(pid int) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tokenCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "nettop", "-p", strconv.Itoa(pid), "-L", "1", "-J", "bytes_in,bytes_out", "-x")
	cmd.Env = append(os.Environ(), "TERM=dumb")
	out, err := cmd.Output()
	if err != nil {
		bytes, fallbackErr := estimateFromLsof(pid)
		if fallbackErr != nil {
			return 0, fmt.Errorf("nettop failed: %w; lsof fallback failed: %v", err, fallbackErr)
		}
		return bytes, nil
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

	return totalBytes, nil
}

func estimateFromLsof(pid int) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), tokenCommandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsof", "-i", "-n", "-P", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(out), "\n")
	established := 0
	for _, line := range lines {
		if strings.Contains(line, "ESTABLISHED") {
			established++
		}
	}

	return int64(established * 500), nil
}

func (tm *TokenMonitor) pruneState(agents []agent.Instance, now time.Time) {
	activePIDs := make(map[int]struct{}, len(agents))
	for _, a := range agents {
		if a.PID > 0 {
			activePIDs[a.PID] = struct{}{}
		}
	}

	for pid, lastSeen := range tm.prevBytesSeen {
		if _, active := activePIDs[pid]; active {
			continue
		}
		if now.Sub(lastSeen) > tokenStateTTL {
			delete(tm.prevBytesSeen, pid)
			delete(tm.prevBytes, pid)
		}
	}

	prunePathOffsetMap(tm.copilotLogOffsets, tm.copilotLogSeen, now)
	prunePathOffsetMap(tm.claudeLogOffsets, tm.claudeLogSeen, now)
	prunePathOffsetMap(tm.aiderLogOffsets, tm.aiderLogSeen, now)
}

func prunePathOffsetMap(offsets map[string]int64, seen map[string]time.Time, now time.Time) {
	for path, lastSeen := range seen {
		if now.Sub(lastSeen) > tokenStateTTL {
			delete(seen, path)
			delete(offsets, path)
		}
	}

	for path := range offsets {
		if _, ok := seen[path]; !ok {
			delete(offsets, path)
		}
	}
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
