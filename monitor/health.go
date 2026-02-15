package monitor

import "time"

// MonitorHealth represents health and error summary for a specific monitor.
type MonitorHealth struct {
	Name        string                       `json:"name"`
	Healthy     bool                         `json:"healthy"`
	TotalErrors int                          `json:"total_errors"`
	LastErrorAt time.Time                    `json:"last_error_at"`
	Errors      map[string]MonitorErrorStats `json:"errors"`
}

// HealthReport is an aggregated view of monitor health for observability.
type HealthReport struct {
	GeneratedAt    time.Time                `json:"generated_at"`
	OverallHealthy bool                     `json:"overall_healthy"`
	TotalErrors    int                      `json:"total_errors"`
	LastErrorAt    time.Time                `json:"last_error_at"`
	Monitors       map[string]MonitorHealth `json:"monitors"`
}

// BuildHealthReport returns a consolidated health snapshot for available monitors.
func BuildHealthReport(
	tokenMonitor *TokenMonitor,
	processMonitor *ProcessMonitor,
	networkMonitor *NetworkMonitor,
	gitMonitor *GitMonitor,
) HealthReport {
	report := HealthReport{
		GeneratedAt:    time.Now(),
		OverallHealthy: true,
		Monitors:       make(map[string]MonitorHealth, 4),
	}

	if tokenMonitor != nil {
		report.Monitors["tokens"] = buildMonitorHealth("tokens", tokenMonitor.GetErrorStats())
	}
	if processMonitor != nil {
		report.Monitors["process"] = buildMonitorHealth("process", processMonitor.GetErrorStats())
	}
	if networkMonitor != nil {
		report.Monitors["network"] = buildMonitorHealth("network", networkMonitor.GetErrorStats())
	}
	if gitMonitor != nil {
		report.Monitors["git"] = buildMonitorHealth("git", gitMonitor.GetErrorStats())
	}

	for _, mh := range report.Monitors {
		report.TotalErrors += mh.TotalErrors
		if mh.TotalErrors > 0 {
			report.OverallHealthy = false
		}
		if mh.LastErrorAt.After(report.LastErrorAt) {
			report.LastErrorAt = mh.LastErrorAt
		}
	}

	return report
}

func buildMonitorHealth(name string, stats map[string]MonitorErrorStats) MonitorHealth {
	health := MonitorHealth{
		Name:    name,
		Healthy: true,
		Errors:  make(map[string]MonitorErrorStats, len(stats)),
	}

	for source, stat := range stats {
		health.Errors[source] = stat
		health.TotalErrors += stat.Count
		if stat.Count > 0 {
			health.Healthy = false
		}
		if stat.LastAt.After(health.LastErrorAt) {
			health.LastErrorAt = stat.LastAt
		}
	}

	return health
}
