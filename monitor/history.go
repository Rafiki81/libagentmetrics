package monitor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// HistoryRecord is a flattened snapshot record for storage.
type HistoryRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	AgentID      string    `json:"agent_id"`
	AgentName    string    `json:"agent_name"`
	PID          int       `json:"pid"`
	Status       string    `json:"status"`
	CPU          float64   `json:"cpu"`
	Memory       float64   `json:"memory"`
	TotalTokens  int64     `json:"total_tokens"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TokensPerSec float64   `json:"tokens_per_sec"`
	EstCost      float64   `json:"est_cost"`
	RequestCount int       `json:"request_count"`
	Model        string    `json:"model"`
	Branch       string    `json:"branch"`
	LOCAdded     int       `json:"loc_added"`
	LOCRemoved   int       `json:"loc_removed"`
	FilesChanged int       `json:"files_changed"`
	TermCmds     int       `json:"terminal_commands"`
	Uptime       string    `json:"uptime"`
}

// HistoryStore manages historical metric recording.
type HistoryStore struct {
	mu      sync.Mutex
	records []HistoryRecord
	maxSize int
	dataDir string
}

// NewHistoryStore creates a history store.
func NewHistoryStore(dataDir string, maxSize int) *HistoryStore {
	if maxSize <= 0 {
		maxSize = 10000
	}
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".agentmetrics", "history")
	}
	os.MkdirAll(dataDir, 0755)

	return &HistoryStore{
		records: make([]HistoryRecord, 0),
		maxSize: maxSize,
		dataDir: dataDir,
	}
}

// Record takes a snapshot of all agents and adds to history.
func (hs *HistoryStore) Record(agents []agent.Instance) {
	hs.mu.Lock()
	defer hs.mu.Unlock()

	now := time.Now()
	for _, a := range agents {
		rec := HistoryRecord{
			Timestamp:    now,
			AgentID:      a.Info.ID,
			AgentName:    a.Info.Name,
			PID:          a.PID,
			Status:       a.Status.String(),
			CPU:          a.CPU,
			Memory:       a.Memory,
			TotalTokens:  a.Tokens.TotalTokens,
			InputTokens:  a.Tokens.InputTokens,
			OutputTokens: a.Tokens.OutputTokens,
			TokensPerSec: a.Tokens.TokensPerSec,
			EstCost:      a.Tokens.EstCost,
			RequestCount: a.Tokens.RequestCount,
			Model:        a.Tokens.LastModel,
			Branch:       a.Git.Branch,
			LOCAdded:     a.LOC.Added,
			LOCRemoved:   a.LOC.Removed,
			FilesChanged: a.LOC.Files,
			TermCmds:     a.Terminal.TotalCommands,
			Uptime:       FormatDuration(a.Session.Uptime),
		}
		hs.records = append(hs.records, rec)
	}

	if len(hs.records) > hs.maxSize {
		hs.records = hs.records[len(hs.records)-hs.maxSize:]
	}
}

// GetRecords returns all historical records.
func (hs *HistoryStore) GetRecords() []HistoryRecord {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	result := make([]HistoryRecord, len(hs.records))
	copy(result, hs.records)
	return result
}

// GetRecordsForAgent returns records for a specific agent.
func (hs *HistoryStore) GetRecordsForAgent(agentID string) []HistoryRecord {
	hs.mu.Lock()
	defer hs.mu.Unlock()
	var result []HistoryRecord
	for _, r := range hs.records {
		if r.AgentID == agentID {
			result = append(result, r)
		}
	}
	return result
}

// ExportJSON exports history to a JSON file.
func (hs *HistoryStore) ExportJSON(path string) error {
	hs.mu.Lock()
	records := make([]HistoryRecord, len(hs.records))
	copy(records, hs.records)
	hs.mu.Unlock()

	if path == "" {
		path = filepath.Join(hs.dataDir, fmt.Sprintf("agentmetrics_%s.json",
			time.Now().Format("20060102_150405")))
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// ExportCSV exports history to a CSV file.
func (hs *HistoryStore) ExportCSV(path string) error {
	hs.mu.Lock()
	records := make([]HistoryRecord, len(hs.records))
	copy(records, hs.records)
	hs.mu.Unlock()

	if path == "" {
		path = filepath.Join(hs.dataDir, fmt.Sprintf("agentmetrics_%s.csv",
			time.Now().Format("20060102_150405")))
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	header := []string{
		"timestamp", "agent_id", "agent_name", "pid", "status",
		"cpu", "memory_mb", "total_tokens", "input_tokens", "output_tokens",
		"tokens_per_sec", "est_cost_usd", "request_count", "model",
		"branch", "loc_added", "loc_removed", "files_changed",
		"terminal_commands", "uptime",
	}
	if err := w.Write(header); err != nil {
		return err
	}

	for _, r := range records {
		row := []string{
			r.Timestamp.Format(time.RFC3339),
			r.AgentID,
			r.AgentName,
			fmt.Sprintf("%d", r.PID),
			r.Status,
			fmt.Sprintf("%.1f", r.CPU),
			fmt.Sprintf("%.1f", r.Memory),
			fmt.Sprintf("%d", r.TotalTokens),
			fmt.Sprintf("%d", r.InputTokens),
			fmt.Sprintf("%d", r.OutputTokens),
			fmt.Sprintf("%.1f", r.TokensPerSec),
			fmt.Sprintf("%.4f", r.EstCost),
			fmt.Sprintf("%d", r.RequestCount),
			r.Model,
			r.Branch,
			fmt.Sprintf("%d", r.LOCAdded),
			fmt.Sprintf("%d", r.LOCRemoved),
			fmt.Sprintf("%d", r.FilesChanged),
			fmt.Sprintf("%d", r.TermCmds),
			r.Uptime,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// DataDir returns the data directory path.
func (hs *HistoryStore) DataDir() string {
	return hs.dataDir
}
