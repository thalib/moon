package constants

import "os"

// File and directory permission constants used for creating files and directories.
// These constants ensure consistent permission settings across the application.
const (
	// DirPermissions is the default permission mode for creating directories.
	// Value: 0755 (rwxr-xr-x) - Owner: read/write/execute, Group/Others: read/execute
	// Used in: daemon/daemon.go, logging/logger.go, preflight/preflight.go
	// Purpose: Standard Unix directory permissions for application directories
	DirPermissions os.FileMode = 0755

	// FilePermissions is the default permission mode for creating regular files.
	// Value: 0644 (rw-r--r--) - Owner: read/write, Group/Others: read-only
	// Used in: daemon/daemon.go, logging/logger.go, preflight/preflight.go
	// Purpose: Standard Unix file permissions for application files
	FilePermissions os.FileMode = 0644
)
