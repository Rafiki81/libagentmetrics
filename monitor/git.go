package monitor

import (
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// GitMonitor tracks git activity in agent working directories.
type GitMonitor struct {
	lastCommitHash map[string]string
}

// NewGitMonitor creates a new git monitor.
func NewGitMonitor() *GitMonitor {
	return &GitMonitor{
		lastCommitHash: make(map[string]string),
	}
}

// Collect gathers git metrics for an agent's working directory.
func (gm *GitMonitor) Collect(a *agent.Instance) {
	if a.WorkDir == "" {
		return
	}

	if !isGitRepo(a.WorkDir) {
		return
	}

	a.Git.Branch = gitCurrentBranch(a.WorkDir)
	a.Git.RecentCommits = gitRecentCommits(a.WorkDir, 5)
	a.Git.Uncommitted = gitUncommittedCount(a.WorkDir)

	added, removed, files := gitDiffStats(a.WorkDir)
	a.Git.LinesAdded = added
	a.Git.LinesRemoved = removed
	a.Git.FilesChanged = files

	a.LOC.Added = added
	a.LOC.Removed = removed
	a.LOC.Net = added - removed
	a.LOC.Files = files
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func gitCurrentBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitRecentCommits(dir string, count int) []agent.GitCommit {
	format := "%h|%s|%ct|%an"
	cmd := exec.Command("git", "-C", dir, "log",
		"--oneline",
		"--format="+format,
		"-n", strconv.Itoa(count),
		"--no-merges",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil
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

	return commits
}

func gitUncommittedCount(dir string) int {
	cmd := exec.Command("git", "-C", dir, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

func gitDiffStats(dir string) (added, removed, files int) {
	a1, r1, f1 := parseDiffStat(dir, "diff", "--stat")
	a2, r2, f2 := parseDiffStat(dir, "diff", "--cached", "--stat")
	return a1 + a2, r1 + r2, f1 + f2
}

func parseDiffStat(dir string, args ...string) (added, removed, files int) {
	fullArgs := append([]string{"-C", dir}, args...)
	cmd := exec.Command("git", fullArgs...)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0
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
		return 0, 0, len(lines) - 1
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

	return added, removed, files
}
