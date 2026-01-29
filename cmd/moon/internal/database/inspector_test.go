package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/thalib/moon/cmd/moon/internal/registry"
)

func TestListTables(t *testing.T) {
	tests := []struct {
		name       string
		connString string
		setup      func(Driver, context.Context) error
		wantTables []string
		wantErr    bool
	}{
		{
			name:       "empty database",
			connString: "sqlite://:memory:",
			setup:      func(d Driver, ctx context.Context) error { return nil },
			wantTables: []string{},
			wantErr:    false,
		},
		{
			name:       "database with tables",
			connString: "sqlite://:memory:",
			setup: func(d Driver, ctx context.Context) error {
				_, err := d.Exec(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)")
				if err != nil {
					return fmt.Errorf("create users: %w", err)
				}
				_, err = d.Exec(ctx, "CREATE TABLE products (id INTEGER PRIMARY KEY, title TEXT)")
				if err != nil {
					return fmt.Errorf("create products: %w", err)
				}
				return nil
			},
			wantTables: []string{"products", "users"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ConnectionString: tt.connString,
				MaxOpenConns:     10,
				MaxIdleConns:     5,
				ConnMaxLifetime:  time.Minute * 5,
			}
			driver, err := NewDriver(cfg)
			if err != nil {
				t.Fatalf("failed to create driver: %v", err)
			}

			ctx := context.Background()
			if err := driver.Connect(ctx); err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer driver.Close()

			if tt.setup != nil {
				if err := tt.setup(driver, ctx); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			tables, err := driver.ListTables(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListTables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(tables) != len(tt.wantTables) {
				t.Errorf("ListTables() got %d tables, want %d", len(tables), len(tt.wantTables))
				return
			}

			for i, table := range tables {
				if table != tt.wantTables[i] {
					t.Errorf("ListTables()[%d] = %s, want %s", i, table, tt.wantTables[i])
				}
			}
		})
	}
}

func TestGetTableInfo(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		setup     func(Driver, context.Context) error
		wantCols  int
		wantErr   bool
	}{
		{
			name:      "simple table",
			tableName: "users",
			setup: func(d Driver, ctx context.Context) error {
				_, err := d.Exec(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)")
				return err
			},
			wantCols: 3,
			wantErr:  false,
		},
		{
			name:      "non-existent table",
			tableName: "nonexistent",
			setup:     func(d Driver, ctx context.Context) error { return nil },
			wantCols:  0,
			wantErr:   false, // SQLite PRAGMA doesn't error, just returns empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ConnectionString: "sqlite://:memory:",
				MaxOpenConns:     10,
				MaxIdleConns:     5,
				ConnMaxLifetime:  time.Minute * 5,
			}
			driver, err := NewDriver(cfg)
			if err != nil {
				t.Fatalf("failed to create driver: %v", err)
			}

			ctx := context.Background()
			if err := driver.Connect(ctx); err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer driver.Close()

			if tt.setup != nil {
				if err := tt.setup(driver, ctx); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			info, err := driver.GetTableInfo(ctx, tt.tableName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTableInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(info.Columns) != tt.wantCols {
					t.Errorf("GetTableInfo() got %d columns, want %d", len(info.Columns), tt.wantCols)
				}
				if info.Name != tt.tableName {
					t.Errorf("GetTableInfo() name = %s, want %s", info.Name, tt.tableName)
				}
			}
		})
	}
}

func TestTableExists(t *testing.T) {
	tests := []struct {
		name      string
		tableName string
		setup     func(Driver, context.Context) error
		want      bool
		wantErr   bool
	}{
		{
			name:      "existing table",
			tableName: "users",
			setup: func(d Driver, ctx context.Context) error {
				_, err := d.Exec(ctx, "CREATE TABLE users (id INTEGER PRIMARY KEY)")
				return err
			},
			want:    true,
			wantErr: false,
		},
		{
			name:      "non-existent table",
			tableName: "nonexistent",
			setup:     func(d Driver, ctx context.Context) error { return nil },
			want:      false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{
				ConnectionString: "sqlite://:memory:",
				MaxOpenConns:     10,
				MaxIdleConns:     5,
				ConnMaxLifetime:  time.Minute * 5,
			}
			driver, err := NewDriver(cfg)
			if err != nil {
				t.Fatalf("failed to create driver: %v", err)
			}

			ctx := context.Background()
			if err := driver.Connect(ctx); err != nil {
				t.Fatalf("failed to connect: %v", err)
			}
			defer driver.Close()

			if tt.setup != nil {
				if err := tt.setup(driver, ctx); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			}

			exists, err := driver.TableExists(ctx, tt.tableName)
			if (err != nil) != tt.wantErr {
				t.Errorf("TableExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if exists != tt.want {
				t.Errorf("TableExists() = %v, want %v", exists, tt.want)
			}
		})
	}
}

func TestInferColumnType(t *testing.T) {
	tests := []struct {
		dbType string
		want   registry.ColumnType
	}{
		{"INTEGER", registry.TypeInteger},
		{"INT", registry.TypeInteger},
		{"BIGINT", registry.TypeInteger},
		{"SERIAL", registry.TypeInteger},
		{"FLOAT", registry.TypeFloat},
		{"DOUBLE", registry.TypeFloat},
		{"REAL", registry.TypeFloat},
		{"DECIMAL", registry.TypeFloat},
		{"BOOLEAN", registry.TypeBoolean},
		{"BOOL", registry.TypeBoolean},
		{"TIMESTAMP", registry.TypeDatetime},
		{"DATE", registry.TypeDatetime},
		{"DATETIME", registry.TypeDatetime},
		{"JSON", registry.TypeJSON},
		{"JSONB", registry.TypeJSON},
		{"TEXT", registry.TypeText},
		{"CLOB", registry.TypeText},
		{"VARCHAR", registry.TypeString},
		{"CHAR", registry.TypeString},
		{"UNKNOWN_TYPE", registry.TypeString},
	}

	for _, tt := range tests {
		t.Run(tt.dbType, func(t *testing.T) {
			got := InferColumnType(tt.dbType)
			if got != tt.want {
				t.Errorf("InferColumnType(%s) = %v, want %v", tt.dbType, got, tt.want)
			}
		})
	}
}
