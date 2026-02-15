package monitor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
)

func TestNewFileWatcher(t *testing.T) {
	fw := NewFileWatcher(50)
	if fw == nil {
		t.Fatal("NewFileWatcher returned nil")
	}
	if fw.maxOps != 50 {
		t.Errorf("maxOps = %d, want 50", fw.maxOps)
	}
}

func TestNewFileWatcher_DefaultMaxOps(t *testing.T) {
	fw := NewFileWatcher(0)
	if fw.maxOps != 100 {
		t.Errorf("maxOps = %d, want 100 (default)", fw.maxOps)
	}

	fw2 := NewFileWatcher(-10)
	if fw2.maxOps != 100 {
		t.Errorf("maxOps = %d, want 100 (default for negative)", fw2.maxOps)
	}
}

func TestFileWatcher_AddRemoveDir(t *testing.T) {
	fw := NewFileWatcher(50)

	fw.AddDir("/tmp/test1")
	fw.AddDir("/tmp/test2")

	fw.mu.Lock()
	if len(fw.dirs) != 2 {
		t.Errorf("dirs count = %d, want 2", len(fw.dirs))
	}
	fw.mu.Unlock()

	fw.RemoveDir("/tmp/test1")

	fw.mu.Lock()
	if len(fw.dirs) != 1 {
		t.Errorf("dirs count = %d, want 1", len(fw.dirs))
	}
	if !fw.dirs["/tmp/test2"] {
		t.Error("expected /tmp/test2 to still be present")
	}
	fw.mu.Unlock()
}

func TestFileWatcher_GetOperations_Empty(t *testing.T) {
	fw := NewFileWatcher(50)
	ops := fw.GetOperations()
	if len(ops) != 0 {
		t.Errorf("got %d operations on empty watcher, want 0", len(ops))
	}
}

func TestFileWatcher_GetOperationsForDir_Empty(t *testing.T) {
	fw := NewFileWatcher(50)
	ops := fw.GetOperationsForDir("/nonexistent")
	if len(ops) != 0 {
		t.Errorf("got %d operations, want 0", len(ops))
	}
}

func TestFileWatcher_DetectChanges(t *testing.T) {
	tmpDir := t.TempDir()
	fw := NewFileWatcher(100)
	fw.AddDir(tmpDir)

	// Take initial snapshot
	fw.takeSnapshots()

	// Create a file
	testFile := filepath.Join(tmpDir, "newfile.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Detect changes
	fw.detectChanges()

	ops := fw.GetOperations()
	if len(ops) == 0 {
		t.Fatal("expected at least 1 operation for created file")
	}

	found := false
	for _, op := range ops {
		if op.Path == testFile && op.Op == "CREATE" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected CREATE operation for %s, got: %+v", testFile, ops)
	}
}

func TestFileWatcher_DetectModify(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fw := NewFileWatcher(100)
	fw.AddDir(tmpDir)
	fw.takeSnapshots()

	// Wait a moment and modify
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fw.detectChanges()

	ops := fw.GetOperations()
	found := false
	for _, op := range ops {
		if op.Path == testFile && op.Op == "MODIFY" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected MODIFY operation for %s", testFile)
	}
}

func TestFileWatcher_DetectDelete(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "todelete.txt")
	if err := os.WriteFile(testFile, []byte("delete me"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	fw := NewFileWatcher(100)
	fw.AddDir(tmpDir)
	fw.takeSnapshots()

	os.Remove(testFile)

	fw.detectChanges()

	ops := fw.GetOperations()
	found := false
	for _, op := range ops {
		if op.Path == testFile && op.Op == "DELETE" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected DELETE operation for %s", testFile)
	}
}

func TestFileWatcher_MaxOps(t *testing.T) {
	fw := NewFileWatcher(5)
	now := time.Now()

	fw.mu.Lock()
	for i := 0; i < 10; i++ {
		fw.addOp(agent.FileOperation{
			Timestamp: now,
			Path:      "/test/file",
			Op:        "CREATE",
		})
	}
	fw.mu.Unlock()

	ops := fw.GetOperations()
	if len(ops) > 5 {
		t.Errorf("got %d operations, want <= 5 (maxOps)", len(ops))
	}
}

func TestFileWatcher_SkipsGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	os.MkdirAll(gitDir, 0755)
	os.WriteFile(filepath.Join(gitDir, "config"), []byte("test"), 0644)

	fw := NewFileWatcher(100)
	fw.AddDir(tmpDir)
	fw.takeSnapshots()

	fw.mu.Lock()
	snapshot := fw.snapshots[tmpDir]
	fw.mu.Unlock()

	for path := range snapshot {
		if filepath.Base(filepath.Dir(path)) == ".git" {
			t.Errorf("snapshot should skip .git directory, found: %s", path)
		}
	}
}

func TestIsUnder(t *testing.T) {
	tests := []struct {
		path, dir string
		want      bool
	}{
		{"/home/user/project/file.go", "/home/user/project", true},
		{"/home/user/project/sub/file.go", "/home/user/project", true},
		{"/home/other/file.go", "/home/user/project", false},
		{"/home/user/project", "/home/user/project", false}, // same dir
	}

	for _, tt := range tests {
		got := isUnder(tt.path, tt.dir)
		if got != tt.want {
			t.Errorf("isUnder(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
		}
	}
}

func TestFileWatcher_StartStop(t *testing.T) {
	fw := NewFileWatcher(50)
	tmpDir := t.TempDir()
	fw.AddDir(tmpDir)

	fw.Start(100 * time.Millisecond)
	time.Sleep(50 * time.Millisecond)
	fw.Stop()
	// No panic = success
}
