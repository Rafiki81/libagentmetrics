// Package config provides configuration management for agent monitoring.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Duration is a time.Duration that marshals/unmarshals as a human-readable string.
type Duration time.Duration

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch val := v.(type) {
	case string:
		dur, err := time.ParseDuration(val)
		if err != nil {
			return err
		}
		*d = Duration(dur)
	case float64:
		*d = Duration(time.Duration(int64(val)))
	default:
		return fmt.Errorf("invalid duration: %v", v)
	}
	return nil
}

// Duration returns the underlying time.Duration.
func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

// Config holds the full application configuration.
type Config struct {
	RefreshInterval Duration          `json:"refresh_interval"`
	Detection       DetectionConfig   `json:"detection"`
	Alerts          AlertConfig       `json:"alerts"`
	Security        SecurityConfig    `json:"security"`
	LocalModels     LocalModelsConfig `json:"local_models"`
	Theme           ThemeConfig       `json:"theme"`
	Export          ExportConfig      `json:"export"`
	Display         DisplayConfig     `json:"display"`
	Keybindings     KeyConfig         `json:"keybindings"`
	Monitor         MonitorConfig     `json:"monitor"`
}

// DetectionConfig controls how agents are detected.
type DetectionConfig struct {
	IgnoreProcessPatterns []string `json:"ignore_process_patterns"`
	IgnorePaths           []string `json:"ignore_paths"`
	SkipSystemProcesses   bool     `json:"skip_system_processes"`
	SkipLsofForDetection  bool     `json:"skip_lsof_for_detection"`
	OnlyExactProcessMatch bool     `json:"only_exact_process_match"`
	DisabledAgents        []string `json:"disabled_agents"`
}

// AlertConfig controls alert thresholds and behavior.
type AlertConfig struct {
	Enabled         bool    `json:"enabled"`
	CPUWarning      float64 `json:"cpu_warning"`
	CPUCritical     float64 `json:"cpu_critical"`
	MemoryWarning   float64 `json:"memory_warning_mb"`
	MemoryCritical  float64 `json:"memory_critical_mb"`
	TokenWarning    int64   `json:"token_warning"`
	TokenCritical   int64   `json:"token_critical"`
	CostWarning     float64 `json:"cost_warning_usd"`
	CostCritical    float64 `json:"cost_critical_usd"`
	IdleMinutes     int     `json:"idle_minutes"`
	CooldownMinutes int     `json:"cooldown_minutes"`
	MaxAlerts       int     `json:"max_alerts"`
}

// ThemeConfig controls UI colors (hex values).
type ThemeConfig struct {
	Primary       string `json:"primary"`
	Secondary     string `json:"secondary"`
	Success       string `json:"success"`
	Warning       string `json:"warning"`
	Danger        string `json:"danger"`
	Muted         string `json:"muted"`
	Background    string `json:"background"`
	BackgroundAlt string `json:"background_alt"`
	Foreground    string `json:"foreground"`
	Border        string `json:"border"`
}

// ExportConfig controls history export settings.
type ExportConfig struct {
	Format     string `json:"format"`
	Directory  string `json:"directory"`
	MaxHistory int    `json:"max_history"`
}

// DisplayConfig controls which sections appear in the dashboard.
type DisplayConfig struct {
	ShowTokens      bool `json:"show_tokens"`
	ShowCost        bool `json:"show_cost"`
	ShowGit         bool `json:"show_git"`
	ShowTerminal    bool `json:"show_terminal"`
	ShowNetwork     bool `json:"show_network"`
	ShowFiles       bool `json:"show_files"`
	ShowSession     bool `json:"show_session"`
	ShowAlerts      bool `json:"show_alerts"`
	ShowSecurity    bool `json:"show_security"`
	ShowLocalModels bool `json:"show_local_models"`
}

// KeyConfig controls keyboard shortcuts.
type KeyConfig struct {
	Quit    string `json:"quit"`
	Refresh string `json:"refresh"`
	Export  string `json:"export"`
	Detail  string `json:"detail"`
	Back    string `json:"back"`
	Up      string `json:"up"`
	Down    string `json:"down"`
	Toggle  string `json:"toggle"`
}

// MonitorConfig controls monitor subsystem parameters.
type MonitorConfig struct {
	MaxLogLines     int      `json:"max_log_lines"`
	MaxFileOps      int      `json:"max_file_ops"`
	MaxTermCommands int      `json:"max_terminal_commands"`
	WatchDirs       []string `json:"watch_dirs"`
}

// LocalModelsConfig controls local model server monitoring.
type LocalModelsConfig struct {
	Enabled   bool                 `json:"enabled"`
	Endpoints []LocalModelEndpoint `json:"endpoints"`
}

// LocalModelEndpoint defines a custom local model server.
type LocalModelEndpoint struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	URL  string `json:"url"`
}

// SecurityConfig controls security monitoring and alerting.
type SecurityConfig struct {
	Enabled                 bool     `json:"enabled"`
	BlockDangerousCommands  bool     `json:"block_dangerous_commands"`
	DangerousCommands       []string `json:"dangerous_commands"`
	SensitiveFiles          []string `json:"sensitive_files"`
	SuspiciousHosts         []string `json:"suspicious_hosts"`
	AllowedRegistries       []string `json:"allowed_registries"`
	EscalationCommands      []string `json:"escalation_commands"`
	CodeInjectionPatterns   []string `json:"code_injection_patterns"`
	SystemModifyCommands    []string `json:"system_modify_commands"`
	ReverseShellPatterns    []string `json:"reverse_shell_patterns"`
	ObfuscationPatterns     []string `json:"obfuscation_patterns"`
	ContainerEscapePatterns []string `json:"container_escape_patterns"`
	EnvManipulationPatterns []string `json:"env_manipulation_patterns"`
	CredentialAccessPatterns []string `json:"credential_access_patterns"`
	LogTamperingPatterns    []string `json:"log_tampering_patterns"`
	RemoteAccessPatterns    []string `json:"remote_access_patterns"`
	ShellPersistenceFiles   []string `json:"shell_persistence_files"`
	MassDeletionThreshold   int      `json:"mass_deletion_threshold"`
	MaxEvents               int      `json:"max_events"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		RefreshInterval: Duration(3 * time.Second),
		Detection: DetectionConfig{
			IgnoreProcessPatterns: []string{
				"CursorUIViewService", "com.apple.", "/System/Library/",
				"/usr/libexec/", "/usr/sbin/", "WindowServer",
				"loginwindow", "launchd", "kernel_task",
			},
			IgnorePaths:           []string{"/Library/", "/System/", "/private/", "/usr/"},
			SkipSystemProcesses:   true,
			SkipLsofForDetection:  false,
			OnlyExactProcessMatch: false,
			DisabledAgents:        []string{},
		},
		Alerts: AlertConfig{
			Enabled: true, CPUWarning: 80, CPUCritical: 95,
			MemoryWarning: 500, MemoryCritical: 1000,
			TokenWarning: 500000, TokenCritical: 2000000,
			CostWarning: 1.0, CostCritical: 5.0,
			IdleMinutes: 10, CooldownMinutes: 5, MaxAlerts: 100,
		},
		Security: SecurityConfig{
			Enabled: true, BlockDangerousCommands: false,
			DangerousCommands: []string{
				"rm -rf /", "rm -rf ~", "rm -rf *", "rm -rf .", "mkfs.", "dd if=",
				":(){:|:&};:", "> /dev/sda", "chmod -R 777", "chmod 777",
				"wget.*|.*sh", "curl.*|.*sh", "curl.*|.*bash", "wget.*|.*bash",
			},
			SensitiveFiles: []string{
				".env", ".ssh/", "id_rsa", "id_ed25519", ".aws/credentials", ".aws/config",
				".gnupg/", ".npmrc", ".pypirc", ".docker/config.json", ".kube/config",
				".git-credentials", "secrets.yml", "secrets.yaml", "credentials.json",
				"service-account.json", ".vault-token", "terraform.tfvars", ".pgpass", "shadow", "passwd",
			},
			SuspiciousHosts: []string{
				"pastebin.com", "requestbin.com", "ngrok.io", "localtunnel.me",
				"hookbin.com", "burpcollaborator.net", "interact.sh", "oast.fun",
			},
			AllowedRegistries: []string{},
			EscalationCommands: []string{
				"sudo ", "su -", "su root", "doas ", "pkexec ",
				"chown root", "chmod u+s", "chmod 4", "setuid",
			},
			CodeInjectionPatterns: []string{
				"eval(", "exec(", "os.system(", "subprocess.call(",
				"child_process.exec(", "Runtime.exec(", "shell_exec(", "system(",
				"| bash", "| sh", "| zsh", "`curl ", "`wget ", "$(curl ", "$(wget ",
			},
			SystemModifyCommands: []string{
				"crontab", "launchctl", "systemctl enable", "systemctl start",
				"chkconfig", "update-rc.d", "/etc/init.d/", "visudo",
				"usermod", "useradd", "groupadd", "iptables", "pfctl",
				"networksetup", "defaults write",
			},
			ReverseShellPatterns: []string{
				"bash -i >& /dev/tcp/", "bash -i >&/dev/tcp/", "sh -i >& /dev/tcp/",
				"/dev/tcp/", "/dev/udp/", "nc -e /bin/", "nc -e /bin/sh",
				"ncat -e ", "ncat --exec", "mkfifo /tmp/", "socat exec:", "socat tcp:",
				"python -c 'import socket", "python3 -c 'import socket",
				"ruby -rsocket", "php -r '$sock=fsockopen", "perl -e 'use Socket",
				"openssl s_client", "telnet ",
			},
			ObfuscationPatterns: []string{
				"base64 --decode", "base64 -d", "base64 -D", "| base64",
				"echo.*|.*base64", "xxd -r", "xxd -p",
			},
			ContainerEscapePatterns: []string{
				"docker run --privileged", "docker run -v /:/", "docker.sock",
				"-v /var/run/docker.sock", "--cap-add=SYS_ADMIN", "--cap-add=ALL",
				"nsenter ", "nsenter -t 1", "docker exec --privileged",
				"--pid=host", "--net=host --privileged", "runc exec", "ctr run",
			},
			EnvManipulationPatterns: []string{
				"export PATH=", "export LD_PRELOAD=", "export LD_LIBRARY_PATH=",
				"export DYLD_INSERT_LIBRARIES=", "export DYLD_LIBRARY_PATH=",
				"export PYTHONPATH=", "export NODE_PATH=",
				"unset SSL_CERT", "export NODE_TLS_REJECT_UNAUTHORIZED=0",
				"export GIT_SSL_NO_VERIFY=",
			},
			CredentialAccessPatterns: []string{
				"security find-generic-password", "security find-internet-password",
				"security dump-keychain", "security export -k",
				"secret-tool lookup", "secret-tool search", "gnome-keyring",
				"pass show", "gopass show", "Login Data", "cookies.sqlite",
				"logins.json", "chrome/Default", "firefox/Profiles",
			},
			LogTamperingPatterns: []string{
				"history -c", "history -w", "history -d",
				"> ~/.bash_history", "> ~/.zsh_history",
				"rm ~/.bash_history", "rm ~/.zsh_history",
				"unset HISTFILE", "export HISTSIZE=0", "set +o history",
				"shred ", "truncate -s 0", "cat /dev/null >",
			},
			RemoteAccessPatterns: []string{
				"ssh ", "ssh -L", "ssh -R", "ssh -D", "scp ",
				"rsync ", "sftp ", "autossh", "mosh ",
			},
			ShellPersistenceFiles: []string{
				".bashrc", ".bash_profile", ".profile", ".zshrc", ".zprofile",
				".zshenv", ".config/fish/config.fish", "Library/LaunchAgents/",
				".config/autostart/", "cron.d/", "cron.daily/",
			},
			MassDeletionThreshold: 10,
			MaxEvents:             500,
		},
		Theme: ThemeConfig{
			Primary: "#7C3AED", Secondary: "#06B6D4", Success: "#10B981",
			Warning: "#F59E0B", Danger: "#EF4444", Muted: "#6B7280",
			Background: "#1A1B26", BackgroundAlt: "#24283B",
			Foreground: "#C0CAF5", Border: "#3B4261",
		},
		Export: ExportConfig{Format: "json", Directory: "", MaxHistory: 10000},
		Display: DisplayConfig{
			ShowTokens: true, ShowCost: true, ShowGit: true, ShowTerminal: true,
			ShowNetwork: true, ShowFiles: true, ShowSession: true, ShowAlerts: true,
			ShowSecurity: true, ShowLocalModels: true,
		},
		Keybindings: KeyConfig{
			Quit: "q", Refresh: "r", Export: "e", Detail: "enter",
			Back: "esc", Up: "up", Down: "down", Toggle: "tab",
		},
		Monitor: MonitorConfig{
			MaxLogLines: 50, MaxFileOps: 200, MaxTermCommands: 50, WatchDirs: []string{},
		},
		LocalModels: LocalModelsConfig{Enabled: true, Endpoints: []LocalModelEndpoint{}},
	}
}

// ConfigPath returns the default config file path.
func ConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".agentmetrics", "config.json")
}

// Load loads config from disk, returning defaults if not found.
func Load() *Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		_ = cfg.Save()
		return cfg
	}
	_ = json.Unmarshal(data, cfg)
	return cfg
}

// Save writes config to disk.
func (c *Config) Save() error {
	path := ConfigPath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ShouldIgnoreProcess returns true if the cmdline matches an ignore pattern.
func (c *Config) ShouldIgnoreProcess(cmdline string) bool {
	if !c.Detection.SkipSystemProcesses {
		return false
	}
	for _, pattern := range c.Detection.IgnoreProcessPatterns {
		if strings.Contains(cmdline, pattern) {
			return true
		}
	}
	return false
}

// ShouldIgnorePath returns true if the path starts with an ignored prefix.
func (c *Config) ShouldIgnorePath(path string) bool {
	for _, prefix := range c.Detection.IgnorePaths {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// IsSystemProcess returns true if the command looks like a macOS system process.
func (c *Config) IsSystemProcess(cmdline string) bool {
	for _, prefix := range []string{"/System/", "/usr/libexec/", "/usr/sbin/", "/Library/Apple/"} {
		if strings.HasPrefix(cmdline, prefix) {
			return true
		}
	}
	return false
}

// IsAgentDisabled returns true if the given agent ID is in the disabled list.
func (c *Config) IsAgentDisabled(agentID string) bool {
	for _, id := range c.Detection.DisabledAgents {
		if strings.EqualFold(id, agentID) {
			return true
		}
	}
	return false
}
