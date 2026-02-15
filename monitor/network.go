package monitor

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// NetworkMonitor tracks network connections for agent processes.
type NetworkMonitor struct{}

// NewNetworkMonitor creates a new network monitor.
func NewNetworkMonitor() *NetworkMonitor {
	return &NetworkMonitor{}
}

// GetConnections returns active network connections for a PID.
func (nm *NetworkMonitor) GetConnections(pid int) []agent.NetConnection {
	cmd := exec.Command("lsof", "-i", "-n", "-P", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var conns []agent.NetConnection
	lines := strings.Split(string(out), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		conn := parseLsofNetLine(line)
		if conn != nil {
			conns = append(conns, *conn)
		}
	}

	return conns
}

// GetAllAgentConnections returns connections for all given PIDs.
func (nm *NetworkMonitor) GetAllAgentConnections(pids []int) map[int][]agent.NetConnection {
	result := make(map[int][]agent.NetConnection)
	for _, pid := range pids {
		conns := nm.GetConnections(pid)
		if len(conns) > 0 {
			result[pid] = conns
		}
	}
	return result
}

func parseLsofNetLine(line string) *agent.NetConnection {
	fields := strings.Fields(line)
	if len(fields) < 9 {
		return nil
	}

	node := strings.ToUpper(fields[7])
	if node != "TCP" && node != "UDP" {
		return nil
	}

	protocol := strings.ToLower(node)
	name := fields[8]

	if !strings.Contains(name, ":") {
		return nil
	}

	state := ""
	if len(fields) > 9 {
		state = strings.Trim(fields[9], "()")
	}

	parts := strings.Split(name, "->")
	localAddr := parts[0]
	remoteAddr := ""
	if len(parts) > 1 {
		remoteAddr = parts[1]
	}

	return &agent.NetConnection{
		LocalAddr:  localAddr,
		RemoteAddr: remoteAddr,
		State:      state,
		Protocol:   protocol,
	}
}

// GetListeningPorts returns a map of TCP port → PID for all processes
// currently in LISTEN state. Uses lsof on macOS.
func (nm *NetworkMonitor) GetListeningPorts() map[int]int {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	result := make(map[int]int)
	lines := strings.Split(string(out), "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		name := fields[8]
		colonIdx := strings.LastIndex(name, ":")
		if colonIdx >= 0 {
			portStr := name[colonIdx+1:]
			port, err := strconv.Atoi(portStr)
			if err == nil {
				result[port] = pid
			}
		}
	}

	return result
}

// DescribeConnection returns a human-readable one-line summary of a connection,
// e.g. "tcp 127.0.0.1:8080 → 10.0.0.1:443 [ESTABLISHED]".
func DescribeConnection(conn agent.NetConnection) string {
	if conn.RemoteAddr == "" {
		return fmt.Sprintf("%s %s (LISTEN)", conn.Protocol, conn.LocalAddr)
	}
	return fmt.Sprintf("%s %s → %s [%s]", conn.Protocol, conn.LocalAddr, conn.RemoteAddr, conn.State)
}
