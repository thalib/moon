package daemon

import (
"fmt"
"os"
"path/filepath"
"strconv"
"strings"
"syscall"
)

// Config holds daemon configuration
type Config struct {
// PIDFile is the path to the PID file
PIDFile string

// WorkDir is the working directory for the daemon
WorkDir string
}

// DefaultConfig returns the default daemon configuration
func DefaultConfig() Config {
return Config{
PIDFile: "/var/run/moon.pid",
WorkDir: "/",
}
}

// Daemonize detaches the process from the terminal and runs in background
// This implements Unix daemon best practices with double fork
func Daemonize(config Config) error {
// Check if already running as daemon
if os.Getppid() == 1 {
// Already running as daemon (parent process is init)
return nil
}

// First fork - create child process
_, err := syscall.ForkExec(
os.Args[0],
os.Args,
&syscall.ProcAttr{
Dir:   config.WorkDir,
Env:   os.Environ(),
Files: []uintptr{0, 1, 2}, // stdin, stdout, stderr
Sys: &syscall.SysProcAttr{
Setsid: true, // Create new session
},
},
)

if err != nil {
return fmt.Errorf("failed to fork process: %w", err)
}

// Parent process exits, child continues
// ForkExec always returns pid > 0 in parent, so this always exits
os.Exit(0)
return nil // Unreachable but required for type-checker
}

// WritePIDFile writes the current process ID to the PID file
func WritePIDFile(pidFile string) error {
// Ensure directory exists
dir := filepath.Dir(pidFile)
if err := os.MkdirAll(dir, 0755); err != nil {
return fmt.Errorf("failed to create PID directory: %w", err)
}

// Check if PID file already exists
if _, err := os.Stat(pidFile); err == nil {
// PID file exists, check if process is still running
content, readErr := os.ReadFile(pidFile)
if readErr == nil {
pidStr := strings.TrimSpace(string(content))
if pid, parseErr := strconv.Atoi(pidStr); parseErr == nil {
if process, findErr := os.FindProcess(pid); findErr == nil {
// Try to send signal 0 to check if process exists
if err := process.Signal(syscall.Signal(0)); err == nil {
return fmt.Errorf("daemon already running with PID %d", pid)
}
}
}
}
// Remove stale PID file (ignore errors as it's cleanup)
_ = os.Remove(pidFile)
}

// Write current PID to file
pid := os.Getpid()
content := []byte(fmt.Sprintf("%d\n", pid))

if err := os.WriteFile(pidFile, content, 0644); err != nil {
return fmt.Errorf("failed to write PID file: %w", err)
}

return nil
}

// RemovePIDFile removes the PID file
func RemovePIDFile(pidFile string) error {
if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
return fmt.Errorf("failed to remove PID file: %w", err)
}
return nil
}

// IsDaemon checks if the current process is running as a daemon
func IsDaemon() bool {
return os.Getppid() == 1
}
