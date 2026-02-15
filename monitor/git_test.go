package monitor

import (
	"errors"
	"testing"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestNewGitMonitor(t *testing.T) {
	gm := NewGitMonitor()
	if gm == nil {
		t.Fatal("NewGitMonitor returned nil")
	}
	if gm.lastCommitHash == nil {
		t.Fatal("lastCommitHash should be initialized")
	}
}

func TestGitMonitorErrorStats(t *testing.T) {
	gm := NewGitMonitor()
	gm.recordError(gitErrRepo, errors.New("repo check failed"))
	gm.recordError(gitErrRepo, errors.New("repo timeout"))

	stats := gm.GetErrorStats()
	repo, ok := stats[gitErrRepo]
	if !ok {
		t.Fatal("expected repo error stats")
	}
	if repo.Count != 2 {
		t.Fatalf("expected count 2, got %d", repo.Count)
	}
	if repo.LastError != "repo timeout" {
		t.Fatalf("expected last error repo timeout, got %q", repo.LastError)
	}

	stats[gitErrRepo] = MonitorErrorStats{}
	stats2 := gm.GetErrorStats()
	if stats2[gitErrRepo].Count != 2 {
		t.Fatal("expected internal stats unchanged after mutating snapshot")
	}
}

func TestGitMonitorZeroValueSafe(t *testing.T) {
	var gm GitMonitor
	a := &agent.Instance{}
	gm.Collect(a)
	_ = gm.GetErrorStats()
}
