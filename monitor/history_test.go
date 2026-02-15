package monitor

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestNewHistoryStore(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)
	if hs == nil {
		t.Fatal("NewHistoryStore returned nil")
	}
	if hs.maxSize != 1000 {
		t.Errorf("maxSize = %d, want 1000", hs.maxSize)
	}
	if hs.dataDir != tmpDir {
		t.Errorf("dataDir = %q, want %q", hs.dataDir, tmpDir)
	}
}

func TestNewHistoryStore_Defaults(t *testing.T) {
	hs := NewHistoryStore("", 0)
	if hs.maxSize != 10000 {
		t.Errorf("maxSize = %d, want 10000 (default)", hs.maxSize)
	}
	if hs.dataDir == "" {
		t.Error("dataDir should have a default value")
	}
}

func TestHistoryStore_Record(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	agents := []agent.Instance{
		{
			Info:   agent.Info{ID: "claude-code", Name: "Claude Code"},
			PID:    1234,
			Status: agent.StatusRunning,
			CPU:    15.5,
			Memory: 250.0,
			Tokens: agent.TokenMetrics{
				TotalTokens:  50000,
				InputTokens:  30000,
				OutputTokens: 20000,
				TokensPerSec: 10.5,
				EstCost:      0.50,
				RequestCount: 5,
				LastModel:    "claude-sonnet-4",
			},
			Git: agent.GitActivity{Branch: "main"},
			LOC: agent.LOCMetrics{Added: 100, Removed: 50, Files: 3},
			Terminal: agent.TerminalActivity{TotalCommands: 10},
			Session:  agent.SessionMetrics{Uptime: 30 * time.Minute},
		},
	}

	hs.Record(agents)

	records := hs.GetRecords()
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	r := records[0]
	if r.AgentID != "claude-code" {
		t.Errorf("AgentID = %q, want claude-code", r.AgentID)
	}
	if r.CPU != 15.5 {
		t.Errorf("CPU = %f, want 15.5", r.CPU)
	}
	if r.TotalTokens != 50000 {
		t.Errorf("TotalTokens = %d, want 50000", r.TotalTokens)
	}
	if r.Status != "RUNNING" {
		t.Errorf("Status = %q, want RUNNING", r.Status)
	}
	if r.Branch != "main" {
		t.Errorf("Branch = %q, want main", r.Branch)
	}
	if r.LOCAdded != 100 {
		t.Errorf("LOCAdded = %d, want 100", r.LOCAdded)
	}
}

func TestHistoryStore_MaxSize(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 5)

	for i := 0; i < 10; i++ {
		hs.Record([]agent.Instance{
			{Info: agent.Info{ID: "test", Name: "Test"}, PID: i + 1},
		})
	}

	records := hs.GetRecords()
	if len(records) > 5 {
		t.Errorf("got %d records, want <= 5 (maxSize)", len(records))
	}
}

func TestHistoryStore_GetRecordsForAgent(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	hs.Record([]agent.Instance{
		{Info: agent.Info{ID: "agent1", Name: "Agent 1"}, PID: 1},
		{Info: agent.Info{ID: "agent2", Name: "Agent 2"}, PID: 2},
	})
	hs.Record([]agent.Instance{
		{Info: agent.Info{ID: "agent1", Name: "Agent 1"}, PID: 1},
	})

	records := hs.GetRecordsForAgent("agent1")
	if len(records) != 2 {
		t.Errorf("got %d records for agent1, want 2", len(records))
	}

	records2 := hs.GetRecordsForAgent("agent2")
	if len(records2) != 1 {
		t.Errorf("got %d records for agent2, want 1", len(records2))
	}

	none := hs.GetRecordsForAgent("nonexistent")
	if len(none) != 0 {
		t.Errorf("got %d records for nonexistent, want 0", len(none))
	}
}

func TestHistoryStore_ExportJSON(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	hs.Record([]agent.Instance{
		{
			Info:   agent.Info{ID: "test", Name: "Test"},
			PID:    1,
			Status: agent.StatusRunning,
			CPU:    5.0,
		},
	})

	exportPath := filepath.Join(tmpDir, "export.json")
	err := hs.ExportJSON(exportPath)
	if err != nil {
		t.Fatalf("ExportJSON error: %v", err)
	}

	data, err := os.ReadFile(exportPath)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}

	var records []HistoryRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("got %d records in JSON, want 1", len(records))
	}
	if records[0].AgentID != "test" {
		t.Errorf("AgentID = %q, want test", records[0].AgentID)
	}
}

func TestHistoryStore_ExportJSON_AutoPath(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	hs.Record([]agent.Instance{
		{Info: agent.Info{ID: "test", Name: "Test"}, PID: 1},
	})

	err := hs.ExportJSON("")
	if err != nil {
		t.Fatalf("ExportJSON with empty path error: %v", err)
	}

	// Check that a file was created in dataDir
	files, _ := filepath.Glob(filepath.Join(tmpDir, "agentmetrics_*.json"))
	if len(files) == 0 {
		t.Error("expected auto-generated JSON file in dataDir")
	}
}

func TestHistoryStore_ExportCSV(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	hs.Record([]agent.Instance{
		{
			Info:   agent.Info{ID: "test", Name: "Test"},
			PID:    1,
			Status: agent.StatusRunning,
			CPU:    5.0,
			Memory: 100.0,
		},
	})

	exportPath := filepath.Join(tmpDir, "export.csv")
	err := hs.ExportCSV(exportPath)
	if err != nil {
		t.Fatalf("ExportCSV error: %v", err)
	}

	f, err := os.Open(exportPath)
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll error: %v", err)
	}

	if len(records) < 2 {
		t.Fatalf("got %d CSV rows, want at least 2 (header + data)", len(records))
	}

	// Header should have 20 columns
	if len(records[0]) != 20 {
		t.Errorf("header has %d columns, want 20", len(records[0]))
	}

	// First data row
	if records[1][1] != "test" {
		t.Errorf("agent_id = %q, want test", records[1][1])
	}
}

func TestHistoryStore_ExportCSV_AutoPath(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	hs.Record([]agent.Instance{
		{Info: agent.Info{ID: "test", Name: "Test"}, PID: 1},
	})

	err := hs.ExportCSV("")
	if err != nil {
		t.Fatalf("ExportCSV with empty path error: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(tmpDir, "agentmetrics_*.csv"))
	if len(files) == 0 {
		t.Error("expected auto-generated CSV file in dataDir")
	}
}

func TestHistoryStore_DataDir(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)
	if hs.DataDir() != tmpDir {
		t.Errorf("DataDir() = %q, want %q", hs.DataDir(), tmpDir)
	}
}

func TestHistoryStore_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	hs := NewHistoryStore(tmpDir, 1000)

	records := hs.GetRecords()
	if len(records) != 0 {
		t.Errorf("empty store has %d records, want 0", len(records))
	}

	err := hs.ExportJSON(filepath.Join(tmpDir, "empty.json"))
	if err != nil {
		t.Fatalf("ExportJSON of empty store error: %v", err)
	}

	err = hs.ExportCSV(filepath.Join(tmpDir, "empty.csv"))
	if err != nil {
		t.Fatalf("ExportCSV of empty store error: %v", err)
	}
}
