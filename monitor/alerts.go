package monitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// AlertThresholds defines configurable alert thresholds.
type AlertThresholds struct {
	CPUWarning        float64
	CPUCritical       float64
	MemoryWarning     float64
	MemoryCritical    float64
	TokenWarning      int64
	TokenCritical     int64
	CostWarning       float64
	CostCritical      float64
	DailyBudgetUSD    float64
	MonthlyBudgetUSD  float64
	BudgetWarnPercent float64
	IdleMinutes       int
	CooldownMinutes   int
	MaxAlerts         int
	CPUPercent        float64
	MemoryMB          float64
	TokensPerMin      int
	CostPerHour       float64
	ErrorRate         float64
}

// DefaultThresholds returns default alert thresholds.
func DefaultThresholds() AlertThresholds {
	return AlertThresholds{
		CPUWarning:        80,
		CPUCritical:       95,
		MemoryWarning:     500,
		MemoryCritical:    1000,
		TokenWarning:      500000,
		TokenCritical:     2000000,
		CostWarning:       1.0,
		CostCritical:      5.0,
		DailyBudgetUSD:    0,
		MonthlyBudgetUSD:  0,
		BudgetWarnPercent: 80,
		IdleMinutes:       10,
		CooldownMinutes:   5,
		MaxAlerts:         100,
	}
}

// AlertMonitor checks agents against thresholds and generates alerts.
type AlertMonitor struct {
	mu         sync.Mutex
	thresholds AlertThresholds
	alerts     []agent.Alert
	maxAlerts  int
	alerted    map[string]time.Time
}

// NewAlertMonitor creates a new alert monitor.
func NewAlertMonitor(thresholds AlertThresholds) *AlertMonitor {
	maxAlerts := thresholds.MaxAlerts
	if maxAlerts <= 0 {
		maxAlerts = 100
	}
	return &AlertMonitor{
		thresholds: thresholds,
		alerts:     make([]agent.Alert, 0),
		maxAlerts:  maxAlerts,
		alerted:    make(map[string]time.Time),
	}
}

// Check evaluates an agent's CPU, memory, token count, cost, and idle time
// against the configured thresholds. Alerts are deduplicated using a
// per-agent cooldown window.
func (am *AlertMonitor) Check(a *agent.Instance) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if a.CPU >= am.thresholds.CPUCritical {
		am.addAlert(a, agent.AlertCritical, fmt.Sprintf("Critical CPU: %.1f%%", a.CPU), "cpu")
	} else if a.CPU >= am.thresholds.CPUWarning {
		am.addAlert(a, agent.AlertWarning, fmt.Sprintf("High CPU: %.1f%%", a.CPU), "cpu")
	}

	if a.Memory >= am.thresholds.MemoryCritical {
		am.addAlert(a, agent.AlertCritical, fmt.Sprintf("Critical memory: %.1f MB", a.Memory), "mem")
	} else if a.Memory >= am.thresholds.MemoryWarning {
		am.addAlert(a, agent.AlertWarning, fmt.Sprintf("High memory: %.1f MB", a.Memory), "mem")
	}

	if a.Tokens.TotalTokens >= am.thresholds.TokenCritical {
		am.addAlert(a, agent.AlertCritical,
			fmt.Sprintf("Critical tokens: %s", FormatTokenCount(a.Tokens.TotalTokens)), "tokens")
	} else if a.Tokens.TotalTokens >= am.thresholds.TokenWarning {
		am.addAlert(a, agent.AlertWarning,
			fmt.Sprintf("High tokens: %s", FormatTokenCount(a.Tokens.TotalTokens)), "tokens")
	}

	if a.Tokens.EstCost >= am.thresholds.CostCritical {
		am.addAlert(a, agent.AlertCritical,
			fmt.Sprintf("Critical cost: %s", FormatCost(a.Tokens.EstCost)), "cost")
	} else if a.Tokens.EstCost >= am.thresholds.CostWarning {
		am.addAlert(a, agent.AlertWarning,
			fmt.Sprintf("High cost: %s", FormatCost(a.Tokens.EstCost)), "cost")
	}

	if am.thresholds.IdleMinutes > 0 && !a.Session.LastActiveAt.IsZero() {
		idleDur := time.Since(a.Session.LastActiveAt).Minutes()
		if idleDur >= float64(am.thresholds.IdleMinutes) {
			am.addAlert(a, agent.AlertInfo,
				fmt.Sprintf("Agent idle for %.0f min", idleDur), "idle")
		}
	}
}

// CheckFleet evaluates aggregated token/cost usage for all agents against
// optional budget thresholds. This is O(n) over agent slice and intended to be
// called at the same cadence as other monitor checks.
func (am *AlertMonitor) CheckFleet(agents []agent.Instance) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if len(agents) == 0 {
		return
	}

	if am.thresholds.DailyBudgetUSD <= 0 && am.thresholds.MonthlyBudgetUSD <= 0 {
		return
	}

	warnPercent := am.thresholds.BudgetWarnPercent
	if warnPercent <= 0 || warnPercent >= 100 {
		warnPercent = 80
	}

	var totalCost float64
	var totalTokens int64
	for _, a := range agents {
		totalCost += a.Tokens.EstCost
		totalTokens += a.Tokens.TotalTokens
	}

	fleet := &agent.Instance{Info: agent.Info{ID: "fleet", Name: "Fleet"}}

	if am.thresholds.DailyBudgetUSD > 0 {
		usagePct := (totalCost / am.thresholds.DailyBudgetUSD) * 100
		if usagePct >= 100 {
			am.addAlert(fleet, agent.AlertCritical,
				fmt.Sprintf("Daily budget exceeded: %s / %s (%.0f%%, %s tokens)",
					FormatCost(totalCost), FormatCost(am.thresholds.DailyBudgetUSD), usagePct, FormatTokenCount(totalTokens)),
				"budget_daily")
		} else if usagePct >= warnPercent {
			am.addAlert(fleet, agent.AlertWarning,
				fmt.Sprintf("Daily budget high usage: %s / %s (%.0f%%, %s tokens)",
					FormatCost(totalCost), FormatCost(am.thresholds.DailyBudgetUSD), usagePct, FormatTokenCount(totalTokens)),
				"budget_daily")
		}
	}

	if am.thresholds.MonthlyBudgetUSD > 0 {
		usagePct := (totalCost / am.thresholds.MonthlyBudgetUSD) * 100
		if usagePct >= 100 {
			am.addAlert(fleet, agent.AlertCritical,
				fmt.Sprintf("Monthly budget exceeded: %s / %s (%.0f%%, %s tokens)",
					FormatCost(totalCost), FormatCost(am.thresholds.MonthlyBudgetUSD), usagePct, FormatTokenCount(totalTokens)),
				"budget_monthly")
		} else if usagePct >= warnPercent {
			am.addAlert(fleet, agent.AlertWarning,
				fmt.Sprintf("Monthly budget high usage: %s / %s (%.0f%%, %s tokens)",
					FormatCost(totalCost), FormatCost(am.thresholds.MonthlyBudgetUSD), usagePct, FormatTokenCount(totalTokens)),
				"budget_monthly")
		}
	}
}

func (am *AlertMonitor) addAlert(a *agent.Instance, level agent.AlertLevel, msg, alertType string) {
	cooldown := time.Duration(am.thresholds.CooldownMinutes) * time.Minute
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}

	key := a.Info.ID + ":" + alertType
	if last, ok := am.alerted[key]; ok {
		if time.Since(last) < cooldown {
			return
		}
	}

	alert := agent.Alert{
		Timestamp: time.Now(),
		Level:     level,
		AgentID:   a.Info.ID,
		AgentName: a.Info.Name,
		Message:   msg,
	}
	am.alerts = append(am.alerts, alert)
	am.alerted[key] = time.Now()

	if len(am.alerts) > am.maxAlerts {
		am.alerts = am.alerts[len(am.alerts)-am.maxAlerts:]
	}
}

// GetAlerts returns all alerts.
func (am *AlertMonitor) GetAlerts() []agent.Alert {
	am.mu.Lock()
	defer am.mu.Unlock()
	result := make([]agent.Alert, len(am.alerts))
	copy(result, am.alerts)
	return result
}

// GetRecentAlerts returns alerts from the last N minutes.
func (am *AlertMonitor) GetRecentAlerts(minutes int) []agent.Alert {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-time.Duration(minutes) * time.Minute)
	var result []agent.Alert
	for _, a := range am.alerts {
		if a.Timestamp.After(cutoff) {
			result = append(result, a)
		}
	}
	return result
}

// AlertCount returns counts by level.
func (am *AlertMonitor) AlertCount() (info, warning, critical int) {
	am.mu.Lock()
	defer am.mu.Unlock()
	for _, a := range am.alerts {
		switch a.Level {
		case agent.AlertInfo:
			info++
		case agent.AlertWarning:
			warning++
		case agent.AlertCritical:
			critical++
		}
	}
	return
}
