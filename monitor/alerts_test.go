package monitor

import (
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestDefaultThresholds(t *testing.T) {
	th := DefaultThresholds()
	if th.CPUWarning != 80 {
		t.Errorf("CPUWarning = %f, want 80", th.CPUWarning)
	}
	if th.CPUCritical != 95 {
		t.Errorf("CPUCritical = %f, want 95", th.CPUCritical)
	}
	if th.MemoryWarning != 500 {
		t.Errorf("MemoryWarning = %f, want 500", th.MemoryWarning)
	}
	if th.MemoryCritical != 1000 {
		t.Errorf("MemoryCritical = %f, want 1000", th.MemoryCritical)
	}
	if th.TokenWarning != 500000 {
		t.Errorf("TokenWarning = %d, want 500000", th.TokenWarning)
	}
	if th.TokenCritical != 2000000 {
		t.Errorf("TokenCritical = %d, want 2000000", th.TokenCritical)
	}
	if th.CostWarning != 1.0 {
		t.Errorf("CostWarning = %f, want 1.0", th.CostWarning)
	}
	if th.CostCritical != 5.0 {
		t.Errorf("CostCritical = %f, want 5.0", th.CostCritical)
	}
	if th.IdleMinutes != 10 {
		t.Errorf("IdleMinutes = %d, want 10", th.IdleMinutes)
	}
	if th.CooldownMinutes != 5 {
		t.Errorf("CooldownMinutes = %d, want 5", th.CooldownMinutes)
	}
	if th.MaxAlerts != 100 {
		t.Errorf("MaxAlerts = %d, want 100", th.MaxAlerts)
	}
}

func TestNewAlertMonitor(t *testing.T) {
	th := DefaultThresholds()
	am := NewAlertMonitor(th)
	if am == nil {
		t.Fatal("NewAlertMonitor returned nil")
	}
	if am.maxAlerts != 100 {
		t.Errorf("maxAlerts = %d, want 100", am.maxAlerts)
	}
}

func TestNewAlertMonitor_ZeroMaxAlerts(t *testing.T) {
	th := DefaultThresholds()
	th.MaxAlerts = 0
	am := NewAlertMonitor(th)
	if am.maxAlerts != 100 {
		t.Errorf("maxAlerts = %d, want 100 (default)", am.maxAlerts)
	}
}

func TestCheck_CPUWarning(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  85.0, // Above 80 (warning), below 95 (critical)
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertWarning {
		t.Errorf("alert level = %q, want WARNING", alerts[0].Level)
	}
	if alerts[0].AgentID != "test" {
		t.Errorf("alert agentID = %q, want test", alerts[0].AgentID)
	}
}

func TestCheck_CPUCritical(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  96.0,
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertCritical {
		t.Errorf("alert level = %q, want CRITICAL", alerts[0].Level)
	}
}

func TestCheck_MemoryWarning(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		Memory: 600.0, // Above 500 warning
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertWarning {
		t.Errorf("alert level = %q, want WARNING", alerts[0].Level)
	}
}

func TestCheck_MemoryCritical(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		Memory: 1200.0,
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertCritical {
		t.Errorf("alert level = %q, want CRITICAL", alerts[0].Level)
	}
}

func TestCheck_TokenWarning(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		Tokens: agent.TokenMetrics{TotalTokens: 600000},
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertWarning {
		t.Errorf("alert level = %q, want WARNING", alerts[0].Level)
	}
}

func TestCheck_TokenCritical(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		Tokens: agent.TokenMetrics{TotalTokens: 3000000},
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertCritical {
		t.Errorf("alert level = %q, want CRITICAL", alerts[0].Level)
	}
}

func TestCheck_CostWarning(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		Tokens: agent.TokenMetrics{EstCost: 2.0},
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertWarning {
		t.Errorf("alert level = %q, want WARNING", alerts[0].Level)
	}
}

func TestCheck_CostCritical(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		Tokens: agent.TokenMetrics{EstCost: 10.0},
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertCritical {
		t.Errorf("alert level = %q, want CRITICAL", alerts[0].Level)
	}
}

func TestCheck_IdleAlert(t *testing.T) {
	th := DefaultThresholds()
	th.IdleMinutes = 1
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		Session: agent.SessionMetrics{
			LastActiveAt: time.Now().Add(-5 * time.Minute),
		},
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertInfo {
		t.Errorf("alert level = %q, want INFO", alerts[0].Level)
	}
}

func TestCheck_NoAlerts(t *testing.T) {
	th := DefaultThresholds()
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		CPU:    10.0,
		Memory: 100.0,
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) != 0 {
		t.Errorf("got %d alerts, want 0", len(alerts))
	}
}

func TestCheck_Cooldown(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 60 // Long cooldown
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  96.0,
	}
	am.Check(inst)
	am.Check(inst) // Second call should be within cooldown
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Errorf("got %d alerts, want 1 (cooldown should prevent duplicate)", len(alerts))
	}
}

func TestCheck_MultipleAlertTypes(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info:   agent.Info{ID: "test", Name: "Test Agent"},
		CPU:    96.0,
		Memory: 1200.0,
		Tokens: agent.TokenMetrics{TotalTokens: 3000000, EstCost: 10.0},
	}
	am.Check(inst)
	alerts := am.GetAlerts()
	if len(alerts) < 4 {
		t.Errorf("got %d alerts, want at least 4 (cpu, mem, tokens, cost)", len(alerts))
	}
}

func TestGetRecentAlerts(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	inst := &agent.Instance{
		Info: agent.Info{ID: "test", Name: "Test Agent"},
		CPU:  96.0,
	}
	am.Check(inst)
	recent := am.GetRecentAlerts(5) // Last 5 minutes
	if len(recent) != 1 {
		t.Errorf("got %d recent alerts, want 1", len(recent))
	}
	old := am.GetRecentAlerts(0) // 0 minutes ago
	if len(old) != 0 {
		t.Errorf("got %d alerts for 0 min window, want 0", len(old))
	}
}

func TestAlertCount(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)

	inst1 := &agent.Instance{
		Info: agent.Info{ID: "a1", Name: "Agent 1"},
		CPU:  85.0, // Warning
	}
	inst2 := &agent.Instance{
		Info: agent.Info{ID: "a2", Name: "Agent 2"},
		CPU:  96.0, // Critical
	}
	inst3 := &agent.Instance{
		Info: agent.Info{ID: "a3", Name: "Agent 3"},
		Session: agent.SessionMetrics{
			LastActiveAt: time.Now().Add(-15 * time.Minute),
		},
	}

	am.Check(inst1)
	am.Check(inst2)
	am.Check(inst3)

	info, warning, critical := am.AlertCount()
	if warning != 1 {
		t.Errorf("warning = %d, want 1", warning)
	}
	if critical != 1 {
		t.Errorf("critical = %d, want 1", critical)
	}
	if info != 1 {
		t.Errorf("info = %d, want 1", info)
	}
}

func TestCheckFleet_DailyBudgetWarning(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	th.DailyBudgetUSD = 10
	th.BudgetWarnPercent = 80
	am := NewAlertMonitor(th)

	agents := []agent.Instance{
		{Info: agent.Info{ID: "a1", Name: "A1"}, Tokens: agent.TokenMetrics{EstCost: 4}},
		{Info: agent.Info{ID: "a2", Name: "A2"}, Tokens: agent.TokenMetrics{EstCost: 4.5}},
	}

	am.CheckFleet(agents)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertWarning {
		t.Fatalf("level = %s, want WARNING", alerts[0].Level)
	}
	if alerts[0].AgentID != "fleet" {
		t.Fatalf("agent id = %s, want fleet", alerts[0].AgentID)
	}
}

func TestCheckFleet_MonthlyBudgetCritical(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	th.MonthlyBudgetUSD = 5
	am := NewAlertMonitor(th)

	agents := []agent.Instance{
		{Info: agent.Info{ID: "a1", Name: "A1"}, Tokens: agent.TokenMetrics{EstCost: 6}},
	}

	am.CheckFleet(agents)
	alerts := am.GetAlerts()
	if len(alerts) != 1 {
		t.Fatalf("got %d alerts, want 1", len(alerts))
	}
	if alerts[0].Level != agent.AlertCritical {
		t.Fatalf("level = %s, want CRITICAL", alerts[0].Level)
	}
}

func TestCheckFleet_NoBudgetsNoAlert(t *testing.T) {
	th := DefaultThresholds()
	th.CooldownMinutes = 0
	th.DailyBudgetUSD = 0
	th.MonthlyBudgetUSD = 0
	am := NewAlertMonitor(th)

	agents := []agent.Instance{
		{Info: agent.Info{ID: "a1", Name: "A1"}, Tokens: agent.TokenMetrics{EstCost: 999}},
	}

	am.CheckFleet(agents)
	if got := len(am.GetAlerts()); got != 0 {
		t.Fatalf("got %d alerts, want 0", got)
	}
}

func TestMaxAlerts_Truncation(t *testing.T) {
	th := DefaultThresholds()
	th.MaxAlerts = 5
	th.CooldownMinutes = 0
	am := NewAlertMonitor(th)
	// Generate more than maxAlerts
	for i := 0; i < 10; i++ {
		inst := &agent.Instance{
			Info: agent.Info{ID: "test", Name: "Test Agent"},
			CPU:  96.0,
		}
		// Reset cooldown by removing the key
		am.mu.Lock()
		am.alerted = make(map[string]time.Time)
		am.mu.Unlock()
		am.Check(inst)
	}
	alerts := am.GetAlerts()
	if len(alerts) > 5 {
		t.Errorf("got %d alerts, want <= 5 (maxAlerts)", len(alerts))
	}
}
