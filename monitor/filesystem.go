package monitor

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

// FileWatcher monitors file system changes in directories where agents are working.
type FileWatcher struct {
	mu         sync.Mutex
	dirs       map[string]bool
	operations []agent.FileOperation
	maxOps     int
	stopCh     chan struct{}
	snapshots  map[string]map[string]time.Time
}

// NewFileWatcher creates a new file system watcher.
func NewFileWatcher(maxOps int) *FileWatcher {
	if maxOps <= 0 {
		maxOps = 100
	}
	return &FileWatcher{
		dirs:      make(map[string]bool),
		maxOps:    maxOps,
		stopCh:    make(chan struct{}),
		snapshots: make(map[string]map[string]time.Time),
	}
}

// AddDir adds a directory to watch.
func (fw *FileWatcher) AddDir(dir string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.dirs[dir] = true
}

// RemoveDir removes a directory from watch.
func (fw *FileWatcher) RemoveDir(dir string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	delete(fw.dirs, dir)
}

// Start begins polling for file changes at the given interval.
// It takes an initial snapshot and then checks for CREATE, MODIFY, and DELETE
// operations in a background goroutine. Call [FileWatcher.Stop] to terminate.
func (fw *FileWatcher) Start(interval time.Duration) {
	fw.takeSnapshots()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fw.detectChanges()
			case <-fw.stopCh:
				return
			}
		}
	}()
}

// Stop stops the file watcher.
func (fw *FileWatcher) Stop() {
	close(fw.stopCh)
}

// GetOperations returns recent file operations.
func (fw *FileWatcher) GetOperations() []agent.FileOperation {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	ops := make([]agent.FileOperation, len(fw.operations))
	copy(ops, fw.operations)
	return ops
}

// GetOperationsForDir returns operations filtered by directory.
func (fw *FileWatcher) GetOperationsForDir(dir string) []agent.FileOperation {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	var result []agent.FileOperation
	for _, op := range fw.operations {
		if isUnder(op.Path, dir) {
			result = append(result, op)
		}
	}
	return result
}

func isUnder(path, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return len(rel) > 0 && rel[0] != '.'
}

func (fw *FileWatcher) takeSnapshots() {
	fw.mu.Lock()
	dirs := make([]string, 0, len(fw.dirs))
	for d := range fw.dirs {
		dirs = append(dirs, d)
	}
	fw.mu.Unlock()

	for _, dir := range dirs {
		snapshot := make(map[string]time.Time)
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if base == ".git" || base == "node_modules" || base == ".next" || base == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			snapshot[path] = info.ModTime()
			return nil
		})

		fw.mu.Lock()
		fw.snapshots[dir] = snapshot
		fw.mu.Unlock()
	}
}

func (fw *FileWatcher) detectChanges() {
	fw.mu.Lock()
	dirs := make([]string, 0, len(fw.dirs))
	for d := range fw.dirs {
		dirs = append(dirs, d)
	}
	fw.mu.Unlock()

	for _, dir := range dirs {
		current := make(map[string]time.Time)
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				base := filepath.Base(path)
				if base == ".git" || base == "node_modules" || base == ".next" || base == "__pycache__" {
					return filepath.SkipDir
				}
				return nil
			}
			current[path] = info.ModTime()
			return nil
		})

		fw.mu.Lock()
		prevSnapshot := fw.snapshots[dir]
		if prevSnapshot == nil {
			prevSnapshot = make(map[string]time.Time)
		}

		now := time.Now()

		for path, modTime := range current {
			prevMod, existed := prevSnapshot[path]
			if !existed {
				fw.addOp(agent.FileOperation{Timestamp: now, Path: path, Op: "CREATE"})
			} else if modTime.After(prevMod) {
				fw.addOp(agent.FileOperation{Timestamp: now, Path: path, Op: "MODIFY"})
			}
		}

		for path := range prevSnapshot {
			if _, exists := current[path]; !exists {
				fw.addOp(agent.FileOperation{Timestamp: now, Path: path, Op: "DELETE"})
			}
		}

		fw.snapshots[dir] = current
		fw.mu.Unlock()
	}
}

func (fw *FileWatcher) addOp(op agent.FileOperation) {
	fw.operations = append(fw.operations, op)
	if len(fw.operations) > fw.maxOps {
		fw.operations = fw.operations[len(fw.operations)-fw.maxOps:]
	}
}
