// Package agent provides types and detection for AI coding agents.
package agent

import "time"

// Status represents the current state of an agent.
type Status int

const (
	StatusUnknown Status = iota
	StatusRunning
	StatusIdle
	StatusStopped
)

func (s Status) String() string {
	switch s {
	case StatusRunning:
		return "RUNNING"
	case StatusIdle:
		return "IDLE"
	case StatusStopped:
		return "STOPPED"
	default:
		return "UNKNOWN"
	}
}

// Info holds metadata about a known agent type.
type Info struct {
	Name           string
	ID             string
	ProcessNames   []string
	LogPaths       []string
	Ports          []int
	Description    string
	DetectPatterns []string
}

// TokenSource indicates how token data was obtained.
type TokenSource string

const (
	TokenSourceNone      TokenSource = ""
	TokenSourceLog       TokenSource = "log"
	TokenSourceDB        TokenSource = "db"
	TokenSourceNetwork   TokenSource = "network"
	TokenSourceEstimated TokenSource = "estimated"
	TokenSourceLocalAPI  TokenSource = "local_api"
)

// TokenMetrics holds token usage data for an agent.
type TokenMetrics struct {
	InputTokens   int64       `json:"input_tokens"`
	OutputTokens  int64       `json:"output_tokens"`
	TotalTokens   int64       `json:"total_tokens"`
	TokensPerSec  float64     `json:"tokens_per_sec"`
	RequestCount  int         `json:"request_count"`
	LastModel     string      `json:"last_model"`
	Source        TokenSource `json:"source"`
	Confidence    float64     `json:"confidence"`
	LastRequestAt time.Time   `json:"last_request_at"`
	EstCost       float64     `json:"est_cost"`
	AvgLatencyMs  int64       `json:"avg_latency_ms"`
}

// GitActivity holds git-related metrics for an agent's working directory.
type GitActivity struct {
	Branch        string      `json:"branch"`
	RecentCommits []GitCommit `json:"recent_commits"`
	Uncommitted   int         `json:"uncommitted"`
	LinesAdded    int         `json:"lines_added"`
	LinesRemoved  int         `json:"lines_removed"`
	FilesChanged  int         `json:"files_changed"`
}

// GitCommit represents a single git commit.
type GitCommit struct {
	Hash    string    `json:"hash"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Author  string    `json:"author"`
}

// TerminalActivity holds terminal command tracking for an agent.
type TerminalActivity struct {
	RecentCommands []TerminalCommand `json:"recent_commands"`
	TotalCommands  int               `json:"total_commands"`
}

// TerminalCommand represents a detected terminal command.
type TerminalCommand struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
	Category  string    `json:"category"`
}

// SessionMetrics holds session timing data.
type SessionMetrics struct {
	StartedAt    time.Time     `json:"started_at"`
	Uptime       time.Duration `json:"uptime"`
	ActiveTime   time.Duration `json:"active_time"`
	IdleTime     time.Duration `json:"idle_time"`
	LastActiveAt time.Time     `json:"last_active_at"`
}

// LOCMetrics holds lines-of-code metrics.
type LOCMetrics struct {
	Added   int `json:"added"`
	Removed int `json:"removed"`
	Net     int `json:"net"`
	Files   int `json:"files_modified"`
}

// FileOperation represents a file change detected.
type FileOperation struct {
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	Op        string    `json:"op"`
}

// NetConnection represents a network connection.
type NetConnection struct {
	LocalAddr  string `json:"local_addr"`
	RemoteAddr string `json:"remote_addr"`
	State      string `json:"state"`
	Protocol   string `json:"protocol"`
}

// AlertLevel represents severity of an alert.
type AlertLevel string

const (
	AlertInfo     AlertLevel = "INFO"
	AlertWarning  AlertLevel = "WARNING"
	AlertCritical AlertLevel = "CRITICAL"
	AlertSecurity AlertLevel = "SECURITY"
)

// Alert represents a triggered alert.
type Alert struct {
	Timestamp time.Time  `json:"timestamp"`
	Level     AlertLevel `json:"level"`
	AgentID   string     `json:"agent_id"`
	AgentName string     `json:"agent_name"`
	Message   string     `json:"message"`
}

// SecurityCategory categorizes the type of security event.
type SecurityCategory string

const (
	SecCatDangerousCommand SecurityCategory = "dangerous_command"
	SecCatSensitiveFile    SecurityCategory = "sensitive_file"
	SecCatNetworkExfil     SecurityCategory = "network_exfil"
	SecCatPackageInstall   SecurityCategory = "package_install"
	SecCatPermEscalation   SecurityCategory = "perm_escalation"
	SecCatSecretsExposure  SecurityCategory = "secrets_exposure"
	SecCatMassDeletion     SecurityCategory = "mass_deletion"
	SecCatSystemModify     SecurityCategory = "system_modify"
	SecCatCodeInjection    SecurityCategory = "code_injection"
	SecCatSuspiciousNet    SecurityCategory = "suspicious_network"
	SecCatReverseShell     SecurityCategory = "reverse_shell"
	SecCatObfuscation      SecurityCategory = "obfuscation"
	SecCatContainerEscape  SecurityCategory = "container_escape"
	SecCatEnvManipulation  SecurityCategory = "env_manipulation"
	SecCatCredentialAccess SecurityCategory = "credential_access"
	SecCatLogTampering     SecurityCategory = "log_tampering"
	SecCatRemoteAccess     SecurityCategory = "remote_access"
	SecCatShellPersistence SecurityCategory = "shell_persistence"
)

// SecuritySeverity indicates how dangerous the event is.
type SecuritySeverity string

const (
	SecSevLow      SecuritySeverity = "LOW"
	SecSevMedium   SecuritySeverity = "MEDIUM"
	SecSevHigh     SecuritySeverity = "HIGH"
	SecSevCritical SecuritySeverity = "CRITICAL"
)

// SecurityEvent represents a detected security-relevant action by an agent.
type SecurityEvent struct {
	Timestamp   time.Time        `json:"timestamp"`
	AgentID     string           `json:"agent_id"`
	AgentName   string           `json:"agent_name"`
	Category    SecurityCategory `json:"category"`
	Severity    SecuritySeverity `json:"severity"`
	Description string           `json:"description"`
	Detail      string           `json:"detail"`
	Blocked     bool             `json:"blocked"`
	Rule        string           `json:"rule"`
}

// LocalModelStatus represents the status of a locally running model.
type LocalModelStatus string

const (
	LocalModelRunning LocalModelStatus = "RUNNING"
	LocalModelLoaded  LocalModelStatus = "LOADED"
	LocalModelIdle    LocalModelStatus = "IDLE"
	LocalModelStopped LocalModelStatus = "STOPPED"
	LocalModelUnknown LocalModelStatus = "UNKNOWN"
)

// LocalModelInfo holds metadata about a local model server.
type LocalModelInfo struct {
	ServerName  string           `json:"server_name"`
	ServerID    string           `json:"server_id"`
	Endpoint    string           `json:"endpoint"`
	PID         int              `json:"pid"`
	Status      LocalModelStatus `json:"status"`
	Models      []LocalModel     `json:"models"`
	ActiveModel string           `json:"active_model"`
	CPU         float64          `json:"cpu"`
	MemoryMB    float64          `json:"memory_mb"`
	VRAM_MB     float64          `json:"vram_mb"`
	UptimeStr   string           `json:"uptime"`
	LastSeen    time.Time        `json:"last_seen"`

	TotalRequests     int64   `json:"total_requests"`
	TokensGenerated   int64   `json:"tokens_generated"`
	TokensPerSec      float64 `json:"tokens_per_sec"`
	AvgLatencyMs      int64   `json:"avg_latency_ms"`
	ActiveConnections int     `json:"active_connections"`
}

// LocalModel represents a single model available on a local server.
type LocalModel struct {
	Name       string  `json:"name"`
	Size       string  `json:"size"`
	SizeBytes  int64   `json:"size_bytes"`
	QuantLevel string  `json:"quant_level"`
	Family     string  `json:"family"`
	Parameters string  `json:"parameters"`
	Running    bool    `json:"running"`
	VRAM_MB    float64 `json:"vram_mb"`
}

// Instance represents a running or detected agent instance.
type Instance struct {
	Info           Info
	PID            int
	Status         Status
	StartTime      time.Time
	LastSeen       time.Time
	CPU            float64
	Memory         float64
	CmdLine        string
	WorkDir        string
	LogLines       []string
	FileOps        []FileOperation
	NetConns       []NetConnection
	Tokens         TokenMetrics
	Git            GitActivity
	Terminal       TerminalActivity
	Session        SessionMetrics
	LOC            LOCMetrics
	SecurityEvents []SecurityEvent
}

// Snapshot is a point-in-time capture of all agent activity.
type Snapshot struct {
	Timestamp time.Time  `json:"timestamp"`
	Agents    []Instance `json:"agents"`
	Alerts    []Alert    `json:"alerts,omitempty"`
}
