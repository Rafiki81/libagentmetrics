package monitor

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestNewTokenMonitor(t *testing.T) {
	tm := NewTokenMonitor()
	if tm == nil {
		t.Fatal("NewTokenMonitor returned nil")
	}
	if tm.data == nil {
		t.Error("data map should be initialized")
	}
	if tm.prevBytes == nil {
		t.Error("prevBytes map should be initialized")
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		count int64
		want  string
	}{
		{0, "—"},
		{-1, "—"},
		{1, "1"},
		{999, "999"},
		{1000, "1.0k"},
		{1500, "1.5k"},
		{10000, "10.0k"},
		{500000, "500.0k"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
		{10000000, "10.0M"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatTokenCount(tt.count)
			if got != tt.want {
				t.Errorf("FormatTokenCount(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestFormatTokensPerSec(t *testing.T) {
	tests := []struct {
		tps  float64
		want string
	}{
		{0, "—"},
		{-1, "—"},
		{1, "1/s"},
		{50, "50/s"},
		{100, "100/s"},
		{999, "999/s"},
		{1000, "1.0k/s"},
		{2500, "2.5k/s"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatTokensPerSec(tt.tps)
			if got != tt.want {
				t.Errorf("FormatTokensPerSec(%f) = %q, want %q", tt.tps, got, tt.want)
			}
		})
	}
}

func TestParseTokenCount(t *testing.T) {
	tests := []struct {
		s    string
		want int64
	}{
		{"100", 100},
		{"1.5k", 1500},
		{"2.5k", 2500},
		{"1.0M", 1000000},
		{"0", 0},
		{"abc", 0},
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			got := parseTokenCount(tt.s)
			if got != tt.want {
				t.Errorf("parseTokenCount(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestGetMetrics_Unknown(t *testing.T) {
	tm := NewTokenMonitor()
	m := tm.GetMetrics("nonexistent")
	if m.TotalTokens != 0 {
		t.Errorf("TotalTokens = %d, want 0 for unknown agent", m.TotalTokens)
	}
	if m.RequestCount != 0 {
		t.Errorf("RequestCount = %d, want 0 for unknown agent", m.RequestCount)
	}
}

func TestParseCursorDBLines(t *testing.T) {
	lines := []string{
		`{"usageData":{"inputTokens":100,"outputTokens":50},"modelConfig":{"modelName":"gpt-4.1"},"conversationMap":{"a":{},"b":{}}}`,
		`{"usageData":{"inputTokens":20,"outputTokens":10},"modelConfig":{"modelName":"default,default,default,default"},"conversationMap":{"c":{}}}`,
		`invalid-json`,
		"",
	}

	got := parseCursorDBLines(lines)
	want := cursorDBParseResult{
		InputTokens:  120,
		OutputTokens: 60,
		RequestCount: 3,
		LastModel:    "gpt-4.1",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCursorDBLines() = %+v, want %+v", got, want)
	}
}

func TestTokenMonitorPruneState(t *testing.T) {
	tm := NewTokenMonitor()
	now := time.Now()

	tm.prevBytes[111] = 1000
	tm.prevBytesSeen[111] = now.Add(-25 * time.Hour)
	tm.prevBytes[222] = 2000
	tm.prevBytesSeen[222] = now.Add(-25 * time.Hour)

	tm.copilotLogOffsets["old.log"] = 10
	tm.copilotLogSeen["old.log"] = now.Add(-25 * time.Hour)
	tm.copilotLogOffsets["new.log"] = 20
	tm.copilotLogSeen["new.log"] = now.Add(-1 * time.Hour)

	agents := []agent.Instance{{PID: 222}}
	tm.pruneState(agents, now)

	if _, ok := tm.prevBytes[111]; ok {
		t.Fatal("expected stale inactive PID 111 to be pruned")
	}
	if _, ok := tm.prevBytesSeen[111]; ok {
		t.Fatal("expected stale inactive PID 111 seen-state to be pruned")
	}
	if _, ok := tm.prevBytes[222]; !ok {
		t.Fatal("expected active PID 222 to be kept")
	}
	if _, ok := tm.copilotLogOffsets["old.log"]; ok {
		t.Fatal("expected stale old.log offset to be pruned")
	}
	if _, ok := tm.copilotLogSeen["old.log"]; ok {
		t.Fatal("expected stale old.log seen-state to be pruned")
	}
	if _, ok := tm.copilotLogOffsets["new.log"]; !ok {
		t.Fatal("expected recent new.log offset to be kept")
	}
}

func TestTokenMonitorErrorStats(t *testing.T) {
	tm := NewTokenMonitor()
	tm.recordError(tokenErrCursorDB, errors.New("sqlite failed"))
	tm.recordError(tokenErrCursorDB, errors.New("sqlite timeout"))

	stats := tm.GetErrorStats()
	cursor, ok := stats[tokenErrCursorDB]
	if !ok {
		t.Fatal("expected cursor_db stats to exist")
	}
	if cursor.Count != 2 {
		t.Fatalf("expected count 2, got %d", cursor.Count)
	}
	if cursor.LastError != "sqlite timeout" {
		t.Fatalf("expected last error sqlite timeout, got %q", cursor.LastError)
	}
	if cursor.LastAt.IsZero() {
		t.Fatal("expected non-zero LastAt timestamp")
	}

	stats[tokenErrCursorDB] = MonitorErrorStats{}
	stats2 := tm.GetErrorStats()
	if stats2[tokenErrCursorDB].Count != 2 {
		t.Fatal("expected internal stats to be immutable from snapshot")
	}
}

func TestTokenMonitorZeroValueSafe(t *testing.T) {
	var tm TokenMonitor
	agents := []agent.Instance{{Info: agent.Info{ID: "unknown"}, PID: -1}}
	tm.Collect(agents)
	_ = tm.GetMetrics("unknown")
	_ = tm.GetErrorStats()
}

func TestTokenConfidence(t *testing.T) {
	tests := []struct {
		source agent.TokenSource
		want   float64
	}{
		{agent.TokenSourceLog, 0.95},
		{agent.TokenSourceDB, 0.95},
		{agent.TokenSourceLocalAPI, 0.95},
		{agent.TokenSourceEstimated, 0.70},
		{agent.TokenSourceNetwork, 0.60},
		{agent.TokenSourceNone, 0.0},
	}

	for _, tt := range tests {
		if got := tokenConfidence(tt.source); got != tt.want {
			t.Fatalf("tokenConfidence(%q) = %.2f, want %.2f", tt.source, got, tt.want)
		}
	}
}
