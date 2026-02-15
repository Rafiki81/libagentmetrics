// Package monitor provides subsystems for tracking AI coding agent activity.
//
// It includes monitors for:
//   - Process CPU/memory metrics ([ProcessMonitor])
//   - Token usage and cost estimation ([TokenMonitor], [EstimateCost])
//   - Git activity in working directories ([GitMonitor])
//   - File system changes ([FileWatcher])
//   - Terminal commands ([TerminalMonitor])
//   - Session timing and idle detection ([SessionMonitor])
//   - Network connections ([NetworkMonitor])
//   - Security analysis of agent behavior ([SecurityMonitor])
//   - Alert generation against configurable thresholds ([AlertMonitor])
//   - Historical metric recording and export ([HistoryStore])
//   - Local model server discovery ([LocalModelMonitor])
//
// All monitors are safe for concurrent use. Most monitors follow the pattern
// of creating an instance with NewXxx, then calling Collect to gather metrics
// into an [agent.Instance].
package monitor
