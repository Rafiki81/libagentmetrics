package monitor

import (
	"errors"
	"testing"
)

func TestBuildHealthReport_EmptyMonitors(t *testing.T) {
	report := BuildHealthReport(nil, nil, nil, nil)

	if !report.OverallHealthy {
		t.Fatal("expected empty report to be healthy")
	}
	if report.TotalErrors != 0 {
		t.Fatalf("expected total errors 0, got %d", report.TotalErrors)
	}
	if len(report.Monitors) != 0 {
		t.Fatalf("expected 0 monitors, got %d", len(report.Monitors))
	}
	if report.GeneratedAt.IsZero() {
		t.Fatal("expected GeneratedAt to be set")
	}
}

func TestBuildHealthReport_AggregatesErrors(t *testing.T) {
	tm := NewTokenMonitor()
	pm := NewProcessMonitor(nil)
	nm := NewNetworkMonitor()
	gm := NewGitMonitor()

	tm.recordError(tokenErrCursorDB, errors.New("cursor timeout"))
	pm.recordError(processErrPS, errors.New("ps failed"))
	pm.recordError(processErrPS, errors.New("ps failed again"))
	nm.recordError(networkErrLsofConnections, errors.New("lsof failed"))
	gm.recordError(gitErrRepo, errors.New("not a repo"))

	report := BuildHealthReport(tm, pm, nm, gm)

	if report.OverallHealthy {
		t.Fatal("expected report to be unhealthy")
	}
	if report.TotalErrors != 5 {
		t.Fatalf("expected total errors 5, got %d", report.TotalErrors)
	}

	if report.Monitors["tokens"].TotalErrors != 1 {
		t.Fatalf("expected tokens total errors 1, got %d", report.Monitors["tokens"].TotalErrors)
	}
	if report.Monitors["process"].TotalErrors != 2 {
		t.Fatalf("expected process total errors 2, got %d", report.Monitors["process"].TotalErrors)
	}
	if report.Monitors["network"].TotalErrors != 1 {
		t.Fatalf("expected network total errors 1, got %d", report.Monitors["network"].TotalErrors)
	}
	if report.Monitors["git"].TotalErrors != 1 {
		t.Fatalf("expected git total errors 1, got %d", report.Monitors["git"].TotalErrors)
	}
	if report.LastErrorAt.IsZero() {
		t.Fatal("expected LastErrorAt to be set")
	}
}

func TestBuildHealthReport_SingleHealthyMonitor(t *testing.T) {
	tm := NewTokenMonitor()
	report := BuildHealthReport(tm, nil, nil, nil)

	if !report.OverallHealthy {
		t.Fatal("expected overall healthy for monitor without errors")
	}
	if report.TotalErrors != 0 {
		t.Fatalf("expected total errors 0, got %d", report.TotalErrors)
	}
	m, ok := report.Monitors["tokens"]
	if !ok {
		t.Fatal("expected tokens monitor to be present")
	}
	if !m.Healthy {
		t.Fatal("expected tokens monitor to be healthy")
	}
}
