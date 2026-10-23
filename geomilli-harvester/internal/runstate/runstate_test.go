package runstate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/travayo/auto-complete/internal/runstate"
)

func TestFileLock(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "lock_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	lockPath := filepath.Join(tmpDir, "pipeline.lock")

	// 1. Acquire lock
	lock1, err := runstate.Acquire(lockPath)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// 2. Second acquire should fail because lock is already held
	_, err2 := runstate.Acquire(lockPath)
	if err2 == nil {
		t.Fatalf("expected error acquiring already-held lock, got nil")
	}

	// 3. Release lock
	if err := lock1.Release(); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// 4. Now should be able to acquire again
	lock3, err := runstate.Acquire(lockPath)
	if err != nil {
		t.Fatalf("failed to re-acquire lock after release: %v", err)
	}
	_ = lock3.Release()
}

func TestManifestAndSHA256(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manifest_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a dummy file
	testFile := filepath.Join(tmpDir, "data.ndjson")
	content := []byte(`{"id":"node:1","name":"Atomium"}` + "\n")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	sha, err := runstate.FileSHA256(testFile)
	if err != nil {
		t.Fatalf("FileSHA256 failed: %v", err)
	}
	if len(sha) != 64 {
		t.Errorf("expected 64-char sha256 hex string, got %s", sha)
	}

	// Write manifest
	manifestPath := filepath.Join(tmpDir, "manifest.json")
	m := runstate.Manifest{
		Job:          "test-job",
		InputPath:    testFile,
		InputSHA256:  sha,
		OutputPath:   testFile,
		OutputSHA256: sha,
		Rows:         1,
	}
	if err := runstate.WriteManifest(manifestPath, m); err != nil {
		t.Fatalf("WriteManifest failed: %v", err)
	}

	// Verify manifest was written
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("manifest file does not exist: %v", err)
	}
}
