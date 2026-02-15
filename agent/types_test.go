package agent

import "testing"

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusUnknown, "UNKNOWN"},
		{StatusRunning, "RUNNING"},
		{StatusIdle, "IDLE"},
		{StatusStopped, "STOPPED"},
		{Status(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	if StatusUnknown != 0 {
		t.Errorf("StatusUnknown = %d, want 0", StatusUnknown)
	}
	if StatusRunning != 1 {
		t.Errorf("StatusRunning = %d, want 1", StatusRunning)
	}
	if StatusIdle != 2 {
		t.Errorf("StatusIdle = %d, want 2", StatusIdle)
	}
	if StatusStopped != 3 {
		t.Errorf("StatusStopped = %d, want 3", StatusStopped)
	}
}

func TestTokenSourceConstants(t *testing.T) {
	tests := []struct {
		name string
		src  TokenSource
		want string
	}{
		{"none", TokenSourceNone, ""},
		{"log", TokenSourceLog, "log"},
		{"db", TokenSourceDB, "db"},
		{"network", TokenSourceNetwork, "network"},
		{"estimated", TokenSourceEstimated, "estimated"},
		{"local_api", TokenSourceLocalAPI, "local_api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.src) != tt.want {
				t.Errorf("TokenSource %s = %q, want %q", tt.name, tt.src, tt.want)
			}
		})
	}
}

func TestAlertLevelConstants(t *testing.T) {
	if AlertInfo != "INFO" {
		t.Errorf("AlertInfo = %q, want INFO", AlertInfo)
	}
	if AlertWarning != "WARNING" {
		t.Errorf("AlertWarning = %q, want WARNING", AlertWarning)
	}
	if AlertCritical != "CRITICAL" {
		t.Errorf("AlertCritical = %q, want CRITICAL", AlertCritical)
	}
	if AlertSecurity != "SECURITY" {
		t.Errorf("AlertSecurity = %q, want SECURITY", AlertSecurity)
	}
}

func TestSecurityCategoryConstants(t *testing.T) {
	categories := []struct {
		cat  SecurityCategory
		want string
	}{
		{SecCatDangerousCommand, "dangerous_command"},
		{SecCatSensitiveFile, "sensitive_file"},
		{SecCatNetworkExfil, "network_exfil"},
		{SecCatPackageInstall, "package_install"},
		{SecCatPermEscalation, "perm_escalation"},
		{SecCatSecretsExposure, "secrets_exposure"},
		{SecCatMassDeletion, "mass_deletion"},
		{SecCatSystemModify, "system_modify"},
		{SecCatCodeInjection, "code_injection"},
		{SecCatSuspiciousNet, "suspicious_network"},
		{SecCatReverseShell, "reverse_shell"},
		{SecCatObfuscation, "obfuscation"},
		{SecCatContainerEscape, "container_escape"},
		{SecCatEnvManipulation, "env_manipulation"},
		{SecCatCredentialAccess, "credential_access"},
		{SecCatLogTampering, "log_tampering"},
		{SecCatRemoteAccess, "remote_access"},
		{SecCatShellPersistence, "shell_persistence"},
	}
	for _, tt := range categories {
		t.Run(string(tt.cat), func(t *testing.T) {
			if string(tt.cat) != tt.want {
				t.Errorf("SecurityCategory = %q, want %q", tt.cat, tt.want)
			}
		})
	}
}

func TestSecuritySeverityConstants(t *testing.T) {
	if SecSevLow != "LOW" {
		t.Errorf("SecSevLow = %q, want LOW", SecSevLow)
	}
	if SecSevMedium != "MEDIUM" {
		t.Errorf("SecSevMedium = %q, want MEDIUM", SecSevMedium)
	}
	if SecSevHigh != "HIGH" {
		t.Errorf("SecSevHigh = %q, want HIGH", SecSevHigh)
	}
	if SecSevCritical != "CRITICAL" {
		t.Errorf("SecSevCritical = %q, want CRITICAL", SecSevCritical)
	}
}

func TestLocalModelStatusConstants(t *testing.T) {
	tests := []struct {
		s    LocalModelStatus
		want string
	}{
		{LocalModelRunning, "RUNNING"},
		{LocalModelLoaded, "LOADED"},
		{LocalModelIdle, "IDLE"},
		{LocalModelStopped, "STOPPED"},
		{LocalModelUnknown, "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.s) != tt.want {
				t.Errorf("LocalModelStatus = %q, want %q", tt.s, tt.want)
			}
		})
	}
}

func TestInstanceZeroValue(t *testing.T) {
	var inst Instance
	if inst.PID != 0 {
		t.Errorf("zero Instance.PID = %d, want 0", inst.PID)
	}
	if inst.Status != StatusUnknown {
		t.Errorf("zero Instance.Status = %v, want StatusUnknown", inst.Status)
	}
	if inst.CPU != 0 {
		t.Errorf("zero Instance.CPU = %f, want 0", inst.CPU)
	}
	if inst.Memory != 0 {
		t.Errorf("zero Instance.Memory = %f, want 0", inst.Memory)
	}
	if inst.Tokens.TotalTokens != 0 {
		t.Errorf("zero Instance.Tokens.TotalTokens = %d, want 0", inst.Tokens.TotalTokens)
	}
	if len(inst.FileOps) != 0 {
		t.Errorf("zero Instance.FileOps len = %d, want 0", len(inst.FileOps))
	}
	if len(inst.NetConns) != 0 {
		t.Errorf("zero Instance.NetConns len = %d, want 0", len(inst.NetConns))
	}
	if len(inst.SecurityEvents) != 0 {
		t.Errorf("zero Instance.SecurityEvents len = %d, want 0", len(inst.SecurityEvents))
	}
}

func TestSnapshotZeroValue(t *testing.T) {
	var snap Snapshot
	if snap.Agents != nil {
		t.Errorf("zero Snapshot.Agents should be nil")
	}
	if snap.Alerts != nil {
		t.Errorf("zero Snapshot.Alerts should be nil")
	}
	if !snap.Timestamp.IsZero() {
		t.Errorf("zero Snapshot.Timestamp should be zero")
	}
}
