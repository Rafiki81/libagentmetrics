package monitor

import (
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

const (
	gitErrRepo   = "repo"
	gitErrBranch = "branch"
	gitErrLog    = "log"
	gitErrStatus = "status"
	gitErrDiff   = "diff"
)

// GitMonitor tracks git activity in agent working directories.
type GitMonitor struct {
	lastCommitHash map[string]string
	mu             sync.Mutex
	errorStats     map[string]MonitorErrorStats
}

func (gm *GitMonitor) ensureInit() {
	if gm.lastCommitHash == nil {
		gm.lastCommitHash = make(map[string]string)
	}
	if gm.errorStats == nil {
		gm.errorStats = make(map[string]MonitorErrorStats)
	}
}

// NewGitMonitor creates a new git monitor.
func NewGitMonitor() *GitMonitor {
	return &GitMonitor{
		lastCommitHash: make(map[string]string),
		errorStats:     make(map[string]MonitorErrorStats),
	}
}

// GetErrorStats returns a snapshot of operational errors per source.
func (gm *GitMonitor) GetErrorStats() map[string]MonitorErrorStats {
	gm.mu.Lock()
	defer gm.mu.Unlock()
	gm.ensureInit()

	stats := make(map[string]MonitorErrorStats, len(gm.errorStats))
	for k, v := range gm.errorStats {
		stats[k] = v
	}
	return stats
}

func (gm *GitMonitor) recordError(source string, err error) {
	if err == nil {
		return
	}
	gm.ensureInit()

	stat := gm.errorStats[source]
	stat.Count++
	stat.LastError = err.Error()
	stat.LastAt = time.Now()
	gm.errorStats[source] = stat
}

// Collect gathers git metrics for an agent's working directory.
func (gm *GitMonitor) Collect(a *agent.Instance) {
	gm.mu.Lock()
	gm.ensureInit()
	gm.mu.Unlock()

	if a.WorkDir == "" {
		return
	}

	isRepo, err := gm.isGitRepo(a.WorkDir)
	if err != nil {
		gm.mu.Lock()
		gm.recordError(gitErrRepo, err)
		gm.mu.Unlock()
		return
	}
	if !isRepo {
		return
	}

	branch, err := gm.gitCurrentBranch(a.WorkDir)
	if err != nil {
		gm.mu.Lock()
		gm.recordError(gitErrBranch, err)
		gm.mu.Unlock()
	}
	a.Git.Branch = branch

	commits, err := gm.gitRecentCommits(a.WorkDir, 5)
	if err != nil {
		gm.mu.Lock()
		gm.recordError(gitErrLog, err)
		gm.mu.Unlock()
	}
	a.Git.RecentCommits = commits

	uncommitted, err := gm.gitUncommittedCount(a.WorkDir)
	if err != nil {
		gm.mu.Lock()
		gm.recordError(gitErrStatus, err)
		gm.mu.Unlock()
	}
	a.Git.Uncommitted = uncommitted

	added, removed, files, err := gm.gitDiffStats(a.WorkDir)
	if err != nil {
		gm.mu.Lock()
		gm.recordError(gitErrDiff, err)
		gm.mu.Unlock()
	}
	a.Git.LinesAdded = added
	a.Git.LinesRemoved = removed
	a.Git.FilesChanged = files

	a.LOC.Added = added
	a.LOC.Removed = removed
	a.LOC.Net = added - removed
	a.LOC.Files = files
}

func (gm *GitMonitor) isGitRepo(dir string) (bool, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func (gm *GitMonitor) gitCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (gm *GitMonitor) gitRecentCommits(dir string, count int) ([]agent.GitCommit, error) {
	format := "%h|%s|%ct|%an"
	cmd := exec.Command("git", "-C", dir, "log",
		"--oneline",
		"--format="+format,
		"-n", strconv.Itoa(count),
		"--no-merges",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []agent.GitCommit
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		ts, _ := strconv.ParseInt(parts[2], 10, 64)
		commits = append(commits, agent.GitCommit{
			Hash:    parts[0],
			Message: parts[1],
			Time:    time.Unix(ts, 0),
			Author:  parts[3],
		})
	}

	return commits, nil
}

func (gm *GitMonitor) gitUncommittedCount(dir string) (int, error) {
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0, nil
	}
	return len(lines), nil
}

func (gm *GitMonitor) gitDiffStats(dir string) (added, removed, files int, err error) {
	a1, r1, f1, err1 := gm.parseDiffStat(dir, "diff", "--stat")
	a2, r2, f2, err2 := gm.parseDiffStat(dir, "diff", "--cached", "--stat")
	if err1 != nil && err2 != nil {
		return 0, 0, 0, err1
	}
	if err1 != nil {
		err = err1
	}
	if err2 != nil {
		err = err2
	}
	return a1 + a2, r1 + r2, f1 + f2, err
}

func (gm *GitMonitor) parseDiffStat(dir string, args ...string) (added, removed, files int, err error) {
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0, err
	}

	numArgs := make([]string, 0, len(args)+2)
	numArgs = append(numArgs, "-C", dir)
	for _, a := range args {
		if a != "--stat" {
			numArgs = append(numArgs, a)
		}
	}
	numArgs = append(numArgs, "--numstat")

	cmd2 := exec.Command("git", numArgs...)
	out2, err := cmd2.Output()
	if err != nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		return 0, 0, len(lines) - 1, err
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out2)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		a, _ := strconv.Atoi(parts[0])
		r, _ := strconv.Atoi(parts[1])
		added += a
		removed += r
		files++
	}

	return added, removed, files, nil
}
