package monitor

import "testing"

func TestNewProcessMonitor(t *testing.T) {
	pm := NewProcessMonitor([]int{1, 2, 3})
	if pm == nil {
		t.Fatal("NewProcessMonitor returned nil")
	}
	if len(pm.pids) != 3 {
		t.Errorf("pids length = %d, want 3", len(pm.pids))
	}
}

func TestNewProcessMonitor_Nil(t *testing.T) {
	pm := NewProcessMonitor(nil)
	if pm == nil {
		t.Fatal("NewProcessMonitor returned nil for nil input")
	}
}

func TestSetPIDs(t *testing.T) {
	pm := NewProcessMonitor(nil)
	pm.SetPIDs([]int{10, 20})
	if len(pm.pids) != 2 {
		t.Errorf("pids length = %d, want 2", len(pm.pids))
	}
}

func TestCollect_EmptyPIDs(t *testing.T) {
	pm := NewProcessMonitor(nil)
	metrics, err := pm.Collect()
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}
	if metrics != nil {
		t.Errorf("Collect() = %v, want nil for empty pids", metrics)
	}
}

func TestCollect_InvalidPID(t *testing.T) {
	pm := NewProcessMonitor([]int{999999999}) // Very unlikely to exist
	metrics, err := pm.Collect()
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}
	// Invalid PID should be skipped
	if len(metrics) != 0 {
		t.Errorf("got %d metrics for invalid PID, want 0", len(metrics))
	}
}

func TestCollect_CurrentProcess(t *testing.T) {
	// Use PID 1 (launchd/init) always exists
	pm := NewProcessMonitor([]int{1})
	metrics, err := pm.Collect()
	if err != nil {
		t.Fatalf("Collect() error: %v", err)
	}
	if len(metrics) == 0 {
		t.Skip("PID 1 not accessible (may need elevated permissions)")
	}
	m := metrics[0]
	if m.PID != 1 {
		t.Errorf("PID = %d, want 1", m.PID)
	}
	if m.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestProcessMetrics_Fields(t *testing.T) {
	m := ProcessMetrics{
		PID:       42,
		CPU:       5.5,
		MemoryMB:  100.0,
		Threads:   4,
		OpenFiles: 20,
	}
	if m.PID != 42 {
		t.Errorf("PID = %d, want 42", m.PID)
	}
	if m.CPU != 5.5 {
		t.Errorf("CPU = %f, want 5.5", m.CPU)
	}
	if m.MemoryMB != 100.0 {
		t.Errorf("MemoryMB = %f, want 100.0", m.MemoryMB)
	}
}
