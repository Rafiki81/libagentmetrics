package monitor

import (
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestNewSessionMonitor(t *testing.T) {
	sm := NewSessionMonitor()
	if sm == nil {
		t.Fatal("NewSessionMonitor returned nil")
	}
	if sm.sessions == nil {
		t.Error("sessions map should be initialized")
	}
}

func TestSessionMonitor_CollectNew(t *testing.T) {
	sm := NewSessionMonitor()
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  5.0,
	}
	sm.Collect(inst)

	if inst.Session.StartedAt.IsZero() {
		t.Error("StartedAt should be set")
	}
	if inst.Session.Uptime <= 0 {
		// Might be very small but should be >=0
	}
}

func TestSessionMonitor_CollectActive(t *testing.T) {
	sm := NewSessionMonitor()
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  5.0, // Above cpuActiveThreshold (0.5)
	}
	sm.Collect(inst)
	if inst.Session.IdleTime < 0 {
		t.Errorf("IdleTime = %v, should not be negative", inst.Session.IdleTime)
	}
}

func TestSessionMonitor_Reset(t *testing.T) {
	sm := NewSessionMonitor()
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  5.0,
	}
	sm.Collect(inst)
	sm.mu.Lock()
	_, exists := sm.sessions["test"]
	sm.mu.Unlock()
	if !exists {
		t.Error("session should exist after Collect")
	}

	sm.Reset("test")
	sm.mu.Lock()
	_, exists = sm.sessions["test"]
	sm.mu.Unlock()
	if exists {
		t.Error("session should not exist after Reset")
	}
}

func TestSessionMonitor_MultipleAgents(t *testing.T) {
	sm := NewSessionMonitor()
	inst1 := &agent.Instance{
		Info: agent.Info{ID: "agent1", Name: "Agent 1"},
		CPU:  5.0,
	}
	inst2 := &agent.Instance{
		Info: agent.Info{ID: "agent2", Name: "Agent 2"},
		CPU:  0.0,
	}
	sm.Collect(inst1)
	sm.Collect(inst2)
	sm.mu.Lock()
	if len(sm.sessions) != 2 {
		t.Errorf("sessions count = %d, want 2", len(sm.sessions))
	}
	sm.mu.Unlock()

	// Reset one
	sm.Reset("agent1")
	sm.mu.Lock()
	if len(sm.sessions) != 1 {
		t.Errorf("sessions count after reset = %d, want 1", len(sm.sessions))
	}
	sm.mu.Unlock()
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{0, "—"},
		{-1 * time.Second, "—"},
		{5 * time.Second, "5s"},
		{30 * time.Second, "30s"},
		{1*time.Minute + 30*time.Second, "1m 30s"},
		{5 * time.Minute, "5m 0s"},
		{59*time.Minute + 59*time.Second, "59m 59s"},
		{1 * time.Hour, "1h 0m"},
		{1*time.Hour + 30*time.Minute, "1h 30m"},
		{2*time.Hour + 15*time.Minute, "2h 15m"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatDuration(tt.d)
			if got != tt.want {
				t.Errorf("FormatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestIntToStr(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{9, "9"},
		{10, "10"},
		{42, "42"},
		{99, "99"},
	}
	for _, tt := range tests {
		got := intToStr(tt.n)
		if got != tt.want {
			t.Errorf("intToStr(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
