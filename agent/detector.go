package agent

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Rafiki81/libagentmetrics/config"
)

// Detector scans for running AI agent processes.
type Detector struct {
	Registry *Registry
	Config   *config.Config
}

// NewDetector creates a new agent detector.
func NewDetector(registry *Registry, cfg *config.Config) *Detector {
	return &Detector{Registry: registry, Config: cfg}
}

type processInfo struct {
	PID     int
	CPU     float64
	Mem     float64
	Command string
	CmdFull string
}

// Scan lists running processes via "ps aux", matches them against the
// registry, and returns one Instance per detected agent. Multiple processes
// for the same agent are merged (highest CPU, summed memory).
func (d *Detector) Scan() ([]Instance, error) {
	procs, err := d.listProcesses()
	if err != nil {
		return nil, fmt.Errorf("listing processes: %w", err)
	}

	seen := make(map[string]*Instance)

	for _, proc := range procs {
		if d.Config.ShouldIgnoreProcess(proc.CmdFull) {
			continue
		}
		if d.Config.Detection.SkipSystemProcesses && d.Config.IsSystemProcess(proc.CmdFull) {
			continue
		}

		agentInfo := d.matchProcess(proc)
		if agentInfo == nil {
			continue
		}

		existing, exists := seen[agentInfo.ID]
		if exists {
			if proc.CPU > existing.CPU {
				existing.CPU = proc.CPU
			}
			existing.Memory += proc.Mem
			continue
		}

		workDir := ""
		if !d.Config.Detection.SkipLsofForDetection {
			workDir = d.getWorkingDir(proc.PID)
			if workDir != "" && d.Config.ShouldIgnorePath(workDir) {
				workDir = ""
			}
		}

		instance := &Instance{
			Info:      *agentInfo,
			PID:       proc.PID,
			Status:    StatusRunning,
			StartTime: time.Now(),
			LastSeen:  time.Now(),
			CPU:       proc.CPU,
			Memory:    proc.Mem,
			CmdLine:   proc.CmdFull,
			WorkDir:   workDir,
		}

		seen[agentInfo.ID] = instance
	}

	result := make([]Instance, 0, len(seen))
	for _, inst := range seen {
		result = append(result, *inst)
	}
	return result, nil
}

func (d *Detector) listProcesses() ([]processInfo, error) {
	cmd := exec.Command("ps", "aux")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var procs []processInfo

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}
		proc, err := parsePSLine(line)
		if err != nil {
			continue
		}
		procs = append(procs, proc)
	}
	return procs, nil
}

func parsePSLine(line string) (processInfo, error) {
	fields := strings.Fields(line)
	if len(fields) < 11 {
		return processInfo{}, fmt.Errorf("not enough fields")
	}

	pid, err := strconv.Atoi(fields[1])
	if err != nil {
		return processInfo{}, err
	}

	cpu, _ := strconv.ParseFloat(fields[2], 64)
	mem, _ := strconv.ParseFloat(fields[3], 64)
	command := fields[10]
	cmdFull := strings.Join(fields[10:], " ")

	return processInfo{
		PID:     pid,
		CPU:     cpu,
		Mem:     mem,
		Command: command,
		CmdFull: cmdFull,
	}, nil
}

func (d *Detector) matchProcess(proc processInfo) *Info {
	cmdBase := extractBaseName(proc.Command)
	if agentInfo := d.Registry.FindByProcess(cmdBase); agentInfo != nil {
		return agentInfo
	}
	if d.Config.Detection.OnlyExactProcessMatch {
		return nil
	}
	if agentInfo := d.Registry.FindByCmdLine(proc.CmdFull); agentInfo != nil {
		return agentInfo
	}
	return nil
}

func extractBaseName(cmd string) string {
	parts := strings.Split(cmd, "/")
	return parts[len(parts)-1]
}

func (d *Detector) getWorkingDir(pid int) string {
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-Fn")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "fcwd") {
			if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "n") {
				return lines[i+1][1:]
			}
		}
	}
	return ""
}
