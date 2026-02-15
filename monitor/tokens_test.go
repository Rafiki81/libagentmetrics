package monitor

import (
	"testing"
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
