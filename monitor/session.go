package monitor

import (
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// SessionMonitor tracks session timing for agents.
type SessionMonitor struct {
	mu       sync.Mutex
	sessions map[string]*sessionState // agentID -> state
}

type sessionState struct {
	startedAt    time.Time
	lastActiveAt time.Time
	activeTime   time.Duration
	idleTime     time.Duration
	lastCPU      float64
	lastCheck    time.Time
}

const cpuActiveThreshold = 0.5 // CPU% above which agent is considered "active"

// NewSessionMonitor creates a new session monitor.
func NewSessionMonitor() *SessionMonitor {
	return &SessionMonitor{
		sessions: make(map[string]*sessionState),
	}
}

// Collect updates session metrics for an agent.
func (sm *SessionMonitor) Collect(a *agent.Instance) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	id := a.Info.ID
	now := time.Now()

	s, exists := sm.sessions[id]
	if !exists {
		s = &sessionState{
			startedAt:    now,
			lastActiveAt: now,
			lastCheck:    now,
		}
		sm.sessions[id] = s
	}

	// Calculate time delta since last check
	delta := now.Sub(s.lastCheck)
	if delta > 30*time.Second {
		delta = 2 * time.Second // Cap at refresh interval if gap is too large
	}

	// Update active/idle time based on CPU usage
	if a.CPU > cpuActiveThreshold {
		s.activeTime += delta
		s.lastActiveAt = now
	} else {
		s.idleTime += delta
	}
	s.lastCPU = a.CPU
	s.lastCheck = now

	// Populate agent's session data
	a.Session.StartedAt = s.startedAt
	a.Session.Uptime = now.Sub(s.startedAt)
	a.Session.ActiveTime = s.activeTime
	a.Session.IdleTime = s.idleTime
	a.Session.LastActiveAt = s.lastActiveAt
}

// Reset clears session data for an agent that has stopped.
func (sm *SessionMonitor) Reset(agentID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, agentID)
}

// FormatDuration formats a duration for human-readable display.
// Returns "—" for zero/negative, "Xs" for seconds-only,
// "Xm Xs" for minutes, or "Xh Xm" for hours.
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return "—"
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmtTime(hours, minutes, seconds, true)
	}
	if minutes > 0 {
		return fmtTime(0, minutes, seconds, false)
	}
	return fmtSec(seconds)
}

func fmtTime(h, m, s int, showH bool) string {
	if showH {
		return intToStr(h) + "h " + intToStr(m) + "m"
	}
	return intToStr(m) + "m " + intToStr(s) + "s"
}

func fmtSec(s int) string {
	return intToStr(s) + "s"
}

func intToStr(n int) string {
	if n < 10 {
		return string(rune('0' + n))
	}
	return string(rune('0'+n/10)) + string(rune('0'+n%10))
}
