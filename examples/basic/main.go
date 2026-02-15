// Example: basic usage of libagentmetrics
//
// This program scans for running AI coding agents, collects metrics
// (CPU, memory, tokens, git, terminal, session, security), and prints
// a summary to stdout.
//
// Run with:
//
//	go run main.go
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
	"github.com/Rafiki81/libagentmetrics/config"
	"github.com/Rafiki81/libagentmetrics/monitor"
)

func main() {
	// 1. Load configuration (creates default if missing)
	cfg := config.DefaultConfig()

	// 2. Create detector and monitor subsystems
	registry := agent.NewRegistry()
	detector := agent.NewDetector(registry, cfg)
	sessMon := monitor.NewSessionMonitor()
	termMon := monitor.NewTerminalMonitor(50)
	tokenMon := monitor.NewTokenMonitor()
	gitMon := monitor.NewGitMonitor()
	netMon := monitor.NewNetworkMonitor()
	secMon := monitor.NewSecurityMonitor(cfg.Security)
	alertMon := monitor.NewAlertMonitor(monitor.AlertThresholds{
		CPUWarning:        cfg.Alerts.CPUWarning,
		CPUCritical:       cfg.Alerts.CPUCritical,
		MemoryWarning:     cfg.Alerts.MemoryWarning,
		MemoryCritical:    cfg.Alerts.MemoryCritical,
		TokenWarning:      cfg.Alerts.TokenWarning,
		TokenCritical:     cfg.Alerts.TokenCritical,
		CostWarning:       cfg.Alerts.CostWarning,
		CostCritical:      cfg.Alerts.CostCritical,
		DailyBudgetUSD:    cfg.Alerts.DailyBudgetUSD,
		MonthlyBudgetUSD:  cfg.Alerts.MonthlyBudgetUSD,
		BudgetWarnPercent: cfg.Alerts.BudgetWarnPercent,
		IdleMinutes:       cfg.Alerts.IdleMinutes,
		CooldownMinutes:   cfg.Alerts.CooldownMinutes,
		MaxAlerts:         cfg.Alerts.MaxAlerts,
	})
	localMon := monitor.NewLocalModelMonitor(cfg.LocalModels)

	fmt.Println("=== libagentmetrics - scan example ===")
	fmt.Println()

	// 3. Detect running agents
	agents, err := detector.Scan()
	if err != nil {
		fmt.Printf("Error scanning: %v\n", err)
		return
	}
	if len(agents) == 0 {
		fmt.Println("No AI coding agents detected.")
		fmt.Println("Start an agent (e.g. Claude Code, Copilot, Aider, Cursor) and run again.")
		return
	}

	fmt.Printf("Detected %d agent(s):\n\n", len(agents))

	// 4. Collect metrics for each agent
	// Build PID list for process monitor
	var pids []int
	for _, a := range agents {
		pids = append(pids, a.PID)
	}
	procMon := monitor.NewProcessMonitor(pids)
	procMetrics, _ := procMon.Collect()

	// Apply process metrics and collect other data
	for i := range agents {
		a := &agents[i]
		for _, pm := range procMetrics {
			if pm.PID == a.PID {
				a.CPU = pm.CPU
				a.Memory = pm.MemoryMB
			}
		}
		sessMon.Collect(a)
		termMon.Collect(a)
		gitMon.Collect(a)
		a.NetConns = netMon.GetConnections(a.PID)
		secMon.CheckAgent(a)
		alertMon.Check(a)
	}

	tokenMon.Collect(agents)
	alertMon.CheckFleet(agents)

	localModels := localMon.GetModels()

	// 5. Print results
	for _, a := range agents {
		printAgent(a)
	}

	alerts := alertMon.GetAlerts()
	if len(alerts) > 0 {
		fmt.Println("-- Alerts --")
		for _, al := range alerts {
			fmt.Printf("  [%s] %s - %s\n", al.Level, al.AgentName, al.Message)
		}
		fmt.Println()
	}

	events := secMon.GetRecentEvents(60)
	if len(events) > 0 {
		fmt.Println("-- Security Events --")
		for _, ev := range events {
			fmt.Printf("  [%s/%s] %s: %s\n", ev.Severity, ev.Category, ev.Description, ev.Detail)
		}
		fmt.Println()
	}

	if len(localModels) > 0 {
		fmt.Println("-- Local Models --")
		for _, lm := range localModels {
			fmt.Printf("  %s (%s) - %s - %d model(s)\n",
				lm.ServerName, lm.Status, lm.Endpoint, len(lm.Models))
		}
		fmt.Println()
	}

	health := monitor.BuildHealthReport(tokenMon, procMon, netMon, gitMon)
	if !health.OverallHealthy {
		fmt.Println("-- Monitor Health --")
		fmt.Printf("  total errors: %d\n", health.TotalErrors)
		for name, mh := range health.Monitors {
			if mh.TotalErrors == 0 {
				continue
			}
			fmt.Printf("  %s: %d error(s)\n", name, mh.TotalErrors)
		}
		fmt.Println()
	}
}

func printAgent(a agent.Instance) {
	fmt.Printf("-- %s (%s) --\n", a.Info.Name, a.Status)
	fmt.Printf("  PID:    %d\n", a.PID)

	if a.CPU > 0 || a.Memory > 0 {
		fmt.Printf("  CPU:    %.1f%%    Memory: %.1f MB\n", a.CPU, a.Memory)
	}

	if a.WorkDir != "" {
		fmt.Printf("  Dir:    %s\n", a.WorkDir)
	}

	if a.Tokens.TotalTokens > 0 {
		fmt.Printf("  Tokens: %s in / %s out  (cost ~ $%.4f)\n",
			monitor.FormatTokenCount(a.Tokens.InputTokens),
			monitor.FormatTokenCount(a.Tokens.OutputTokens),
			a.Tokens.EstCost)
		if a.Tokens.LastModel != "" {
			fmt.Printf("  Model:  %s\n", a.Tokens.LastModel)
		}
	}

	if a.Git.Branch != "" {
		fmt.Printf("  Git:    branch=%s  +%d/-%d (%d files)\n",
			a.Git.Branch, a.Git.LinesAdded, a.Git.LinesRemoved, a.Git.FilesChanged)
	}

	if a.Session.Uptime > 0 {
		fmt.Printf("  Up:     %s (active %s, idle %s)\n",
			monitor.FormatDuration(a.Session.Uptime),
			monitor.FormatDuration(a.Session.ActiveTime),
			monitor.FormatDuration(a.Session.IdleTime))
	}

	if a.Terminal.TotalCommands > 0 {
		fmt.Printf("  Cmds:   %d detected\n", a.Terminal.TotalCommands)
		max := 5
		if len(a.Terminal.RecentCommands) < max {
			max = len(a.Terminal.RecentCommands)
		}
		for _, cmd := range a.Terminal.RecentCommands[len(a.Terminal.RecentCommands)-max:] {
			fmt.Printf("          [%s] %s  (%s)\n",
				cmd.Category,
				truncate(cmd.Command, 60),
				cmd.Timestamp.Format(time.Kitchen))
		}
	}

	if len(a.NetConns) > 0 {
		fmt.Printf("  Net:    %d connection(s)\n", len(a.NetConns))
	}

	if len(a.SecurityEvents) > 0 {
		fmt.Printf("  Sec:    %d event(s)\n", len(a.SecurityEvents))
	}

	fmt.Println()
}

func truncate(s string, max int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > max {
		return s[:max-1] + "..."
	}
	return s
}
