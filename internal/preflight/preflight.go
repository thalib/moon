package preflight

import (
	"fmt"
	"os"
	"path/filepath"
)

// FileCheck represents a required file or directory
type FileCheck struct {
	Path      string
	IsDir     bool
	Required  bool
	FailFatal bool // If true, failure to create causes program exit
}

// CheckResult represents the result of a preflight check
type CheckResult struct {
	Path    string
	Exists  bool
	Created bool
	Error   error
}

// ValidateAndCreate checks if required files and directories exist
// and creates them if they don't. Returns results for all checks.
func ValidateAndCreate(checks []FileCheck) ([]CheckResult, error) {
	var results []CheckResult
	var fatalErrors []error

	for _, check := range checks {
		result := CheckResult{
			Path:   check.Path,
			Exists: false,
		}

		// Check if path exists
		info, err := os.Stat(check.Path)
		if err == nil {
			// Path exists
			result.Exists = true

			// Verify it's the correct type
			if check.IsDir && !info.IsDir() {
				result.Error = fmt.Errorf("path exists but is not a directory: %s", check.Path)
				if check.FailFatal {
					fatalErrors = append(fatalErrors, result.Error)
				}
			} else if !check.IsDir && info.IsDir() {
				result.Error = fmt.Errorf("path exists but is a directory: %s", check.Path)
				if check.FailFatal {
					fatalErrors = append(fatalErrors, result.Error)
				}
			}
		} else if os.IsNotExist(err) {
			// Path doesn't exist, try to create it
			if check.IsDir {
				// Create directory
				if err := os.MkdirAll(check.Path, 0755); err != nil {
					result.Error = fmt.Errorf("failed to create directory %s: %w", check.Path, err)
					if check.FailFatal {
						fatalErrors = append(fatalErrors, result.Error)
					}
				} else {
					result.Created = true
				}
			} else {
				// For files, create parent directory and touch the file
				dir := filepath.Dir(check.Path)
				if err := os.MkdirAll(dir, 0755); err != nil {
					result.Error = fmt.Errorf("failed to create parent directory for %s: %w", check.Path, err)
					if check.FailFatal {
						fatalErrors = append(fatalErrors, result.Error)
					}
				} else {
					// Create empty file (touch) - O_EXCL ensures we don't overwrite
					f, err := os.OpenFile(check.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
					if err != nil {
						result.Error = fmt.Errorf("failed to create file %s: %w", check.Path, err)
						if check.FailFatal {
							fatalErrors = append(fatalErrors, result.Error)
						}
					} else {
						defer f.Close()
						result.Created = true
					}
				}
			}
		} else {
			// Some other error occurred
			result.Error = fmt.Errorf("failed to check path %s: %w", check.Path, err)
			if check.FailFatal {
				fatalErrors = append(fatalErrors, result.Error)
			}
		}

		results = append(results, result)
	}

	// If any fatal errors occurred, return the first one
	if len(fatalErrors) > 0 {
		return results, fatalErrors[0]
	}

	return results, nil
}

// CreateOrTruncateFile creates a new file or truncates an existing file to zero length.
// If the file doesn't exist, it creates it. If it exists, it truncates it.
// This ensures the file is empty and ready for new content.
func CreateOrTruncateFile(path string) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
	}

	// Truncate the file (create if doesn't exist)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create or truncate file %s: %w", path, err)
	}
	defer f.Close()

	return nil
}
