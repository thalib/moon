// Package consistency provides database consistency checking and repair logic.
// It ensures that the in-memory schema registry remains synchronized with
// physical database tables across restarts and failures.
package consistency

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/config"
	"github.com/thalib/moon/cmd/moon/internal/constants"
	"github.com/thalib/moon/cmd/moon/internal/database"
	"github.com/thalib/moon/cmd/moon/internal/logging"
	"github.com/thalib/moon/cmd/moon/internal/registry"
)

// IssueType represents the type of consistency issue
type IssueType string

const (
	// IssueOrphanedTable indicates a table exists in the database but not in the registry
	IssueOrphanedTable IssueType = "orphaned_table"

	// IssueOrphanedRegistry indicates a collection is in the registry but the table doesn't exist
	IssueOrphanedRegistry IssueType = "orphaned_registry"
)

// Issue represents a detected consistency issue
type Issue struct {
	Type        IssueType `json:"type"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Repaired    bool      `json:"repaired"`
}

// CheckResult contains the results of a consistency check
type CheckResult struct {
	Consistent bool          `json:"consistent"`
	Issues     []Issue       `json:"issues"`
	Duration   time.Duration `json:"duration"`
	TimedOut   bool          `json:"timed_out"`
}

// Checker performs consistency checks between registry and database
type Checker struct {
	db       database.Driver
	registry *registry.SchemaRegistry
	config   *config.RecoveryConfig
}

// NewChecker creates a new consistency checker
func NewChecker(db database.Driver, reg *registry.SchemaRegistry, cfg *config.RecoveryConfig) *Checker {
	return &Checker{
		db:       db,
		registry: reg,
		config:   cfg,
	}
}

// Check performs a consistency check and optionally repairs issues
func (c *Checker) Check(ctx context.Context) (*CheckResult, error) {
	start := time.Now()

	// Create a timeout context based on configuration
	timeout := time.Duration(c.config.CheckTimeout) * time.Second
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	result := &CheckResult{
		Consistent: true,
		Issues:     []Issue{},
	}

	// Get all physical tables
	allTables, err := c.db.ListTables(checkCtx)
	if err != nil {
		// Check if we timed out
		if checkCtx.Err() == context.DeadlineExceeded {
			result.TimedOut = true
			result.Duration = time.Since(start)
			return result, fmt.Errorf("%s", constants.ConsistencyErrorMessages.CheckTimeout)
		}
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	// Filter out system tables - they should not be managed by consistency checker
	tables := make([]string, 0, len(allTables))
	for _, table := range allTables {
		if !constants.IsSystemTable(table) {
			tables = append(tables, table)
		}
	}

	// Get all registered collections
	collections := c.registry.List()

	// Build maps for quick lookup
	tableMap := make(map[string]bool)
	for _, table := range tables {
		tableMap[table] = true
	}

	collectionMap := make(map[string]bool)
	for _, col := range collections {
		collectionMap[col] = true
	}

	// Check for orphaned registry entries (in registry but table doesn't exist)
	for _, col := range collections {
		// Skip system tables in registry check
		if constants.IsSystemTable(col) {
			continue
		}

		if !tableMap[col] {
			issue := Issue{
				Type:        IssueOrphanedRegistry,
				Name:        col,
				Description: constants.ConsistencyErrorMessages.OrphanedRegistry,
			}

			if c.config.AutoRepair {
				// Remove from registry
				if err := c.registry.Delete(col); err != nil {
					logging.Warnf("Failed to remove orphaned registry entry '%s': %v", col, err)
				} else {
					issue.Repaired = true
					logging.Infof("Removed orphaned registry entry: %s", col)
				}
			}

			result.Issues = append(result.Issues, issue)
			result.Consistent = false
		}
	}

	// Check for orphaned tables (in database but not in registry)
	for _, table := range tables {
		if !collectionMap[table] {
			issue := Issue{
				Type:        IssueOrphanedTable,
				Name:        table,
				Description: constants.ConsistencyErrorMessages.OrphanedTable,
			}

			if c.config.AutoRepair {
				if c.config.DropOrphans {
					// Validate table name to prevent SQL injection
					if !isValidTableName(table) {
						logging.Warnf("Skipping drop of table '%s': invalid table name", table)
						continue
					}

					// Drop the orphaned table
					dropSQL := fmt.Sprintf("DROP TABLE %s", table)
					if _, err := c.db.Exec(checkCtx, dropSQL); err != nil {
						logging.Warnf("Failed to drop orphaned table '%s': %v", table, err)
					} else {
						issue.Repaired = true
						logging.Infof("Dropped orphaned table: %s", table)
					}
				} else {
					// Try to register the orphaned table
					if err := c.registerOrphanedTable(checkCtx, table); err != nil {
						logging.Warnf("Failed to register orphaned table '%s': %v", table, err)
					} else {
						issue.Repaired = true
						logging.Infof("Registered orphaned table: %s", table)
					}
				}
			}

			result.Issues = append(result.Issues, issue)
			result.Consistent = false
		}
	}

	result.Duration = time.Since(start)

	// Log summary
	if result.Consistent {
		logging.Info("Consistency check passed: registry and database are synchronized")
	} else {
		logging.Warnf("Consistency check found %d issue(s)", len(result.Issues))
		for _, issue := range result.Issues {
			status := "not repaired"
			if issue.Repaired {
				status = "repaired"
			}
			logging.Warnf("  - %s: %s (%s)", issue.Type, issue.Name, status)
		}
	}

	return result, nil
}

// registerOrphanedTable attempts to infer schema and register an orphaned table
func (c *Checker) registerOrphanedTable(ctx context.Context, tableName string) error {
	// Get table info from database
	tableInfo, err := c.db.GetTableInfo(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}

	if len(tableInfo.Columns) == 0 {
		return fmt.Errorf("table has no columns")
	}

	// Convert database columns to registry columns
	var columns []registry.Column
	for _, col := range tableInfo.Columns {
		// Skip primary key column (ulid) as it's automatically added
		if col.IsPrimaryKey && strings.ToLower(col.Name) == "ulid" {
			continue
		}

		regCol := registry.Column{
			Name:         col.Name,
			Type:         database.InferColumnType(col.Type),
			Nullable:     col.Nullable,
			Unique:       col.IsUnique,
			DefaultValue: col.DefaultValue,
		}
		columns = append(columns, regCol)
	}

	// Only register if we have columns besides the primary key
	if len(columns) == 0 {
		return fmt.Errorf("table only has primary key column")
	}

	// Register in the registry
	collection := &registry.Collection{
		Name:    tableName,
		Columns: columns,
	}

	if err := c.registry.Set(collection); err != nil {
		return fmt.Errorf("failed to set registry: %w", err)
	}

	return nil
}

// GetStatus returns a simple status string for health checks
func (c *Checker) GetStatus(ctx context.Context) string {
	result, err := c.Check(ctx)
	if err != nil {
		return "error"
	}

	if result.TimedOut {
		return "timeout"
	}

	if result.Consistent {
		return "ok"
	}

	// Check if all issues are repaired
	allRepaired := true
	for _, issue := range result.Issues {
		if !issue.Repaired {
			allRepaired = false
			break
		}
	}

	if allRepaired {
		return "ok"
	}

	return "inconsistent"
}

// isValidTableName validates that a table name contains only safe characters
// to prevent SQL injection in DROP TABLE and other operations
func isValidTableName(name string) bool {
	if name == "" {
		return false
	}

	// Table names should only contain alphanumeric characters and underscores
	// and must start with a letter or underscore
	if len(name) > 64 {
		return false // Reasonable limit for table names
	}

	for i, ch := range name {
		if i == 0 {
			// First character must be letter or underscore
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_') {
				return false
			}
		} else {
			// Subsequent characters can be alphanumeric or underscore
			if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
				return false
			}
		}
	}

	return true
}
