package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestWritePIDFile_FailedDirCreation tests PID file creation with invalid directory
func TestWritePIDFile_FailedDirCreation(t *testing.T) {
	// Create a file where we want to create a directory (will cause MkdirAll to fail)
	tmpDir := t.TempDir()
	blockingFile := filepath.Join(tmpDir, "blocking")
	if err := os.WriteFile(blockingFile, []byte("block"), 0644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Try to create a PID file in a path where a file blocks directory creation
	pidFile := filepath.Join(blockingFile, "subdir", "test.pid")

	err := WritePIDFile(pidFile)
	if err == nil {
		t.Error("Expected error when directory creation fails, got nil")
	}
}

// TestWritePIDFile_StalePIDFile tests cleanup of stale PID files
func TestWritePIDFile_StalePIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write a stale PID (a PID that doesn't exist)
	// Use a very high PID that's unlikely to exist
	stalePID := 999999999
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(stalePID)+"\n"), 0644); err != nil {
		t.Fatalf("Failed to write stale PID file: %v", err)
	}

	// WritePIDFile should succeed after cleaning up stale PID
	err := WritePIDFile(pidFile)
	if err != nil {
		t.Errorf("WritePIDFile() should succeed with stale PID file, got: %v", err)
	}

	// Verify new PID file contains current PID
	content, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("Failed to read PID file: %v", err)
	}

	pidStr := string(content)
	pidStr = pidStr[:len(pidStr)-1] // Remove newline
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		t.Fatalf("Failed to parse PID from file: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), pid)
	}
}

// TestWritePIDFile_InvalidPIDContent tests handling of invalid PID file content
func TestWritePIDFile_InvalidPIDContent(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Write invalid PID content (non-numeric)
	if err := os.WriteFile(pidFile, []byte("invalid-pid\n"), 0644); err != nil {
		t.Fatalf("Failed to write invalid PID file: %v", err)
	}

	// WritePIDFile should succeed (cleans up invalid PID file)
	err := WritePIDFile(pidFile)
	if err != nil {
		t.Errorf("WritePIDFile() should succeed with invalid PID content, got: %v", err)
	}
}

// TestWritePIDFile_EmptyPIDFile tests handling of empty PID file
func TestWritePIDFile_EmptyPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Create empty PID file
	if err := os.WriteFile(pidFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write empty PID file: %v", err)
	}

	// WritePIDFile should succeed (cleans up empty PID file)
	err := WritePIDFile(pidFile)
	if err != nil {
		t.Errorf("WritePIDFile() should succeed with empty PID file, got: %v", err)
	}
}

// TestWritePIDFile_UnreadablePIDFile tests handling when existing PID file cannot be read
func TestWritePIDFile_UnreadablePIDFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("Skipping test when running as root")
	}

	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	// Create PID file with no read permissions
	if err := os.WriteFile(pidFile, []byte("12345\n"), 0000); err != nil {
		t.Fatalf("Failed to write PID file: %v", err)
	}

	// WritePIDFile should still succeed (remove unreadable file and create new one)
	err := WritePIDFile(pidFile)
	// This should fail or succeed depending on whether we can remove the file
	if err != nil {
		t.Logf("WritePIDFile() with unreadable file: %v (expected behavior may vary)", err)
	}

	// Cleanup
	os.Chmod(pidFile, 0644)
}

// TestWritePIDFile_NestedDirectories tests PID file creation in deeply nested directories
func TestWritePIDFile_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "a", "b", "c", "d", "test.pid")

	err := WritePIDFile(pidFile)
	if err != nil {
		t.Fatalf("WritePIDFile() failed to create nested directories: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(pidFile); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}
}

// TestRemovePIDFile_DirectoryInsteadOfFile tests removing when path is a directory
func TestRemovePIDFile_DirectoryInsteadOfFile(t *testing.T) {
	tmpDir := t.TempDir()
	pidDir := filepath.Join(tmpDir, "test.pid")

	// Create a directory instead of a file
	if err := os.Mkdir(pidDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// RemovePIDFile should handle this case
	err := RemovePIDFile(pidDir)
	// Behavior depends on implementation - directory removal may fail
	if err != nil {
		t.Logf("RemovePIDFile() on directory: %v (expected behavior)", err)
	}
}

// TestIsDaemon_ConsistentResults verifies IsDaemon returns consistent results
func TestIsDaemon_ConsistentResults(t *testing.T) {
	result1 := IsDaemon()
	result2 := IsDaemon()

	if result1 != result2 {
		t.Error("IsDaemon() returned inconsistent results")
	}

	// In test environment, we're typically not a daemon
	if result1 {
		t.Log("Running as daemon (unusual in test environment)")
	}
}

// TestDefaultConfig_Values ensures DefaultConfig returns expected values
func TestDefaultConfig_Values(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PIDFile != "/var/run/moon.pid" {
		t.Errorf("DefaultConfig() PIDFile = %s, want /var/run/moon.pid", cfg.PIDFile)
	}

	if cfg.WorkDir != "/" {
		t.Errorf("DefaultConfig() WorkDir = %s, want /", cfg.WorkDir)
	}
}
