package monitor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessMetrics holds CPU/memory metrics for a process.
type ProcessMetrics struct {
	PID       int
	CPU       float64
	MemoryMB  float64
	Threads   int
	OpenFiles int
	Timestamp time.Time
}

// ProcessMonitor monitors metrics of specific PIDs.
type ProcessMonitor struct {
	pids []int
}

// NewProcessMonitor creates a process monitor for given PIDs.
func NewProcessMonitor(pids []int) *ProcessMonitor {
	return &ProcessMonitor{pids: pids}
}

// SetPIDs updates the list of PIDs to monitor.
func (pm *ProcessMonitor) SetPIDs(pids []int) {
	pm.pids = pids
}

// Collect gathers metrics for all tracked PIDs.
func (pm *ProcessMonitor) Collect() ([]ProcessMetrics, error) {
	if len(pm.pids) == 0 {
		return nil, nil
	}
	var metrics []ProcessMetrics
	for _, pid := range pm.pids {
		m, err := pm.collectOne(pid)
		if err != nil {
			continue
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}

func (pm *ProcessMonitor) collectOne(pid int) (ProcessMetrics, error) {
	pidStr := strconv.Itoa(pid)
	cmd := exec.Command("ps", "-p", pidStr, "-o", "%cpu,%mem,rss")
	out, err := cmd.Output()
	if err != nil {
		return ProcessMetrics{}, fmt.Errorf("ps failed for pid %d: %w", pid, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return ProcessMetrics{}, fmt.Errorf("process %d not found", pid)
	}

	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return ProcessMetrics{}, fmt.Errorf("unexpected ps output for pid %d", pid)
	}

	cpu, _ := strconv.ParseFloat(fields[0], 64)
	rssKB, _ := strconv.ParseFloat(fields[2], 64)
	memMB := rssKB / 1024.0
	openFiles := countOpenFiles(pid)

	return ProcessMetrics{
		PID:       pid,
		CPU:       cpu,
		MemoryMB:  memMB,
		OpenFiles: openFiles,
		Timestamp: time.Now(),
	}, nil
}

func countOpenFiles(pid int) int {
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(string(out), "\n")
	count := len(lines) - 2
	if count < 0 {
		count = 0
	}
	return count
}

// IsRunning checks if a PID is still active.
func IsRunning(pid int) bool {
	cmd := exec.Command("kill", "-0", strconv.Itoa(pid))
	return cmd.Run() == nil
}

// GetChildPIDs returns child processes of a PID.
func GetChildPIDs(pid int) []int {
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	var children []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		childPID, err := strconv.Atoi(line)
		if err == nil {
			children = append(children, childPID)
		}
	}
	return children
}
