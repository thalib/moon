package constants

// Default file and directory paths used by the application.
const (
	// DefaultPIDFile is the default location for the daemon PID file.
	// Used in: daemon/daemon.go
	// Purpose: Stores the process ID when running in daemon mode
	DefaultPIDFile = "/var/run/moon.pid"

	// DefaultWorkingDirectory is the default working directory for the daemon.
	// Used in: daemon/daemon.go
	// Purpose: Standard root directory for daemon processes
	DefaultWorkingDirectory = "/"
)
