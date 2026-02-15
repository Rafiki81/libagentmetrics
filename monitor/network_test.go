package monitor

import (
	"testing"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestParseLsofNetLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		wantNil bool
		want    *agent.NetConnection
	}{
		{
			name: "TCP established",
			line: "node    12345 user   20u  IPv4 0x1234  0t0  TCP 127.0.0.1:8080->142.250.80.46:443 (ESTABLISHED)",
			want: &agent.NetConnection{
				LocalAddr:  "127.0.0.1:8080",
				RemoteAddr: "142.250.80.46:443",
				State:      "ESTABLISHED",
				Protocol:   "tcp",
			},
		},
		{
			name: "TCP listen",
			line: "node    12345 user   20u  IPv4 0x1234  0t0  TCP *:3000 (LISTEN)",
			want: &agent.NetConnection{
				LocalAddr:  "*:3000",
				RemoteAddr: "",
				State:      "LISTEN",
				Protocol:   "tcp",
			},
		},
		{
			name: "UDP",
			line: "node    12345 user   20u  IPv4 0x1234  0t0  UDP 127.0.0.1:5353->224.0.0.251:5353",
			want: &agent.NetConnection{
				LocalAddr:  "127.0.0.1:5353",
				RemoteAddr: "224.0.0.251:5353",
				State:      "",
				Protocol:   "udp",
			},
		},
		{
			name:    "too few fields",
			line:    "node 12345 user 20u IPv4",
			wantNil: true,
		},
		{
			name:    "not TCP/UDP",
			line:    "node    12345 user   20u  IPv4 0x1234  0t0  PIPE something",
			wantNil: true,
		},
		{
			name:    "no colon in name",
			line:    "node    12345 user   20u  IPv4 0x1234  0t0  TCP noport (LISTEN)",
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseLsofNetLine(tt.line)
			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}
			if result == nil {
				t.Fatal("expected non-nil result")
			}
			if result.LocalAddr != tt.want.LocalAddr {
				t.Errorf("LocalAddr = %q, want %q", result.LocalAddr, tt.want.LocalAddr)
			}
			if result.RemoteAddr != tt.want.RemoteAddr {
				t.Errorf("RemoteAddr = %q, want %q", result.RemoteAddr, tt.want.RemoteAddr)
			}
			if result.State != tt.want.State {
				t.Errorf("State = %q, want %q", result.State, tt.want.State)
			}
			if result.Protocol != tt.want.Protocol {
				t.Errorf("Protocol = %q, want %q", result.Protocol, tt.want.Protocol)
			}
		})
	}
}

func TestDescribeConnection(t *testing.T) {
	tests := []struct {
		name string
		conn agent.NetConnection
		want string
	}{
		{
			name: "listen",
			conn: agent.NetConnection{Protocol: "tcp", LocalAddr: "*:3000"},
			want: "tcp *:3000 (LISTEN)",
		},
		{
			name: "established",
			conn: agent.NetConnection{
				Protocol: "tcp", LocalAddr: "127.0.0.1:8080",
				RemoteAddr: "142.250.80.46:443", State: "ESTABLISHED",
			},
			want: "tcp 127.0.0.1:8080 → 142.250.80.46:443 [ESTABLISHED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DescribeConnection(tt.conn)
			if got != tt.want {
				t.Errorf("DescribeConnection = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewNetworkMonitor(t *testing.T) {
	nm := NewNetworkMonitor()
	if nm == nil {
		t.Fatal("NewNetworkMonitor returned nil")
	}
}

func TestIsUnusualPort(t *testing.T) {
	tests := []struct {
		addr string
		want bool
	}{
		{"192.168.1.1:443", false},
		{"192.168.1.1:80", false},
		{"192.168.1.1:8080", false},
		{"192.168.1.1:22", false},
		{"192.168.1.1:3000", false},
		{"192.168.1.1:5432", false},
		{"192.168.1.1:9999", true},
		{"192.168.1.1:4444", true},
		{"192.168.1.1:31337", true},
		{"noport", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := isUnusualPort(tt.addr)
			if got != tt.want {
				t.Errorf("isUnusualPort(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}

func TestGetAllAgentConnections_EmptyPIDs(t *testing.T) {
	nm := NewNetworkMonitor()
	result := nm.GetAllAgentConnections(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 results for nil PIDs, got %d", len(result))
	}

	result2 := nm.GetAllAgentConnections([]int{})
	if len(result2) != 0 {
		t.Errorf("expected 0 results for empty PIDs, got %d", len(result2))
	}
}
