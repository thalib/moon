package preflight

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAndCreate_ExistingDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "existing")
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	checks := []FileCheck{
		{Path: existingDir, IsDir: true, Required: true, FailFatal: true},
	}

	results, err := ValidateAndCreate(checks)
	if err != nil {
		t.Errorf("ValidateAndCreate() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Exists {
		t.Error("Expected existing directory to be marked as exists")
	}

	if results[0].Created {
		t.Error("Expected existing directory to not be marked as created")
	}

	if results[0].Error != nil {
		t.Errorf("Expected no error, got: %v", results[0].Error)
	}
}

func TestValidateAndCreate_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	newDir := filepath.Join(tmpDir, "new", "nested", "dir")

	checks := []FileCheck{
		{Path: newDir, IsDir: true, Required: true, FailFatal: true},
	}

	results, err := ValidateAndCreate(checks)
	if err != nil {
		t.Errorf("ValidateAndCreate() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Exists {
		t.Error("Expected new directory to not be marked as exists before creation")
	}

	if !results[0].Created {
		t.Error("Expected new directory to be marked as created")
	}

	if results[0].Error != nil {
		t.Errorf("Expected no error, got: %v", results[0].Error)
	}

	// Verify directory was actually created
	info, err := os.Stat(newDir)
	if err != nil {
		t.Errorf("Directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Error("Created path is not a directory")
	}
}

func TestValidateAndCreate_CreateFile(t *testing.T) {
	tmpDir := t.TempDir()
	newFile := filepath.Join(tmpDir, "new", "nested", "file.txt")

	checks := []FileCheck{
		{Path: newFile, IsDir: false, Required: true, FailFatal: true},
	}

	results, err := ValidateAndCreate(checks)
	if err != nil {
		t.Errorf("ValidateAndCreate() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Created {
		t.Error("Expected new file to be marked as created")
	}

	if results[0].Error != nil {
		t.Errorf("Expected no error, got: %v", results[0].Error)
	}

	// Verify file was actually created
	info, err := os.Stat(newFile)
	if err != nil {
		t.Errorf("File was not created: %v", err)
	}
	if info.IsDir() {
		t.Error("Created path is a directory, expected file")
	}
}

func TestValidateAndCreate_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checks := []FileCheck{
		{Path: existingFile, IsDir: false, Required: true, FailFatal: true},
	}

	results, err := ValidateAndCreate(checks)
	if err != nil {
		t.Errorf("ValidateAndCreate() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if !results[0].Exists {
		t.Error("Expected existing file to be marked as exists")
	}

	if results[0].Created {
		t.Error("Expected existing file to not be marked as created")
	}
}

func TestValidateAndCreate_WrongType_FileIsDir(t *testing.T) {
	tmpDir := t.TempDir()
	existingDir := filepath.Join(tmpDir, "dir")
	if err := os.Mkdir(existingDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	checks := []FileCheck{
		{Path: existingDir, IsDir: false, Required: true, FailFatal: true},
	}

	results, err := ValidateAndCreate(checks)
	if err == nil {
		t.Error("Expected error when path is directory but file was expected")
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("Expected error in result when path type is wrong")
	}
}

func TestValidateAndCreate_WrongType_DirIsFile(t *testing.T) {
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "file.txt")
	if err := os.WriteFile(existingFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checks := []FileCheck{
		{Path: existingFile, IsDir: true, Required: true, FailFatal: true},
	}

	results, err := ValidateAndCreate(checks)
	if err == nil {
		t.Error("Expected error when path is file but directory was expected")
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	if results[0].Error == nil {
		t.Error("Expected error in result when path type is wrong")
	}
}

func TestValidateAndCreate_MultipleChecks(t *testing.T) {
	tmpDir := t.TempDir()

	checks := []FileCheck{
		{Path: filepath.Join(tmpDir, "dir1"), IsDir: true, Required: true, FailFatal: false},
		{Path: filepath.Join(tmpDir, "dir2/nested"), IsDir: true, Required: true, FailFatal: false},
		{Path: filepath.Join(tmpDir, "file1.txt"), IsDir: false, Required: true, FailFatal: false},
		{Path: filepath.Join(tmpDir, "subdir/file2.txt"), IsDir: false, Required: true, FailFatal: false},
	}

	results, err := ValidateAndCreate(checks)
	if err != nil {
		t.Errorf("ValidateAndCreate() failed: %v", err)
	}

	if len(results) != 4 {
		t.Fatalf("Expected 4 results, got %d", len(results))
	}

	// All should be created successfully
	for i, result := range results {
		if !result.Created {
			t.Errorf("Check %d: expected to be created", i)
		}
		if result.Error != nil {
			t.Errorf("Check %d: unexpected error: %v", i, result.Error)
		}
	}
}

func TestValidateAndCreate_NonFatalError(t *testing.T) {
	// Try to create in a non-existent root path (permission denied scenario)
	checks := []FileCheck{
		{Path: "/nonexistent/root/path/dir", IsDir: true, Required: true, FailFatal: false},
	}

	results, err := ValidateAndCreate(checks)
	// Should not return error because FailFatal is false
	if err != nil {
		t.Errorf("Expected no error for non-fatal check, got: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	// Should have an error in the result
	if results[0].Error == nil {
		t.Error("Expected error in result for failed creation")
	}
}

func TestCreateOrTruncateFile_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "new", "test.log")

	err := CreateOrTruncateFile(filePath)
	if err != nil {
		t.Errorf("CreateOrTruncateFile() failed: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(filePath)
	if err != nil {
		t.Errorf("File was not created: %v", err)
	}

	if info.Size() != 0 {
		t.Errorf("Expected file size 0, got %d", info.Size())
	}
}

func TestCreateOrTruncateFile_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "existing.log")

	// Create file with content
	content := []byte("some existing content that should be removed")
	if err := os.WriteFile(filePath, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Verify file has content
	info, _ := os.Stat(filePath)
	if info.Size() == 0 {
		t.Fatal("Test file should have content before truncation")
	}

	// Truncate the file
	err := CreateOrTruncateFile(filePath)
	if err != nil {
		t.Errorf("CreateOrTruncateFile() failed: %v", err)
	}

	// Verify file was truncated
	info, err = os.Stat(filePath)
	if err != nil {
		t.Errorf("File does not exist after truncation: %v", err)
	}

	if info.Size() != 0 {
		t.Errorf("Expected file size 0 after truncation, got %d", info.Size())
	}
}

func TestCreateOrTruncateFile_CreateParentDirs(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nested", "deep", "dirs", "file.log")

	err := CreateOrTruncateFile(filePath)
	if err != nil {
		t.Errorf("CreateOrTruncateFile() failed: %v", err)
	}

	// Verify file was created
	info, err := os.Stat(filePath)
	if err != nil {
		t.Errorf("File was not created: %v", err)
	}

	if info.Size() != 0 {
		t.Errorf("Expected file size 0, got %d", info.Size())
	}

	// Verify parent directories were created
	parentDir := filepath.Dir(filePath)
	if _, err := os.Stat(parentDir); err != nil {
		t.Errorf("Parent directory was not created: %v", err)
	}
}
