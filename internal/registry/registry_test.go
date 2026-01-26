package registry

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewSchemaRegistry(t *testing.T) {
	registry := NewSchemaRegistry()
	if registry == nil {
		t.Fatal("NewSchemaRegistry() returned nil")
	}

	if registry.Count() != 0 {
		t.Errorf("Expected empty registry, got count %d", registry.Count())
	}
}

func TestSchemaRegistry_Set_Get(t *testing.T) {
	registry := NewSchemaRegistry()

	collection := &Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger, Nullable: false, Unique: true},
			{Name: "name", Type: TypeString, Nullable: false},
			{Name: "email", Type: TypeString, Nullable: false, Unique: true},
		},
	}

	// Test Set
	if err := registry.Set(collection); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Test Get
	retrieved, ok := registry.Get("users")
	if !ok {
		t.Fatal("Get() returned false for existing collection")
	}

	if retrieved.Name != collection.Name {
		t.Errorf("Expected name %s, got %s", collection.Name, retrieved.Name)
	}

	if len(retrieved.Columns) != len(collection.Columns) {
		t.Errorf("Expected %d columns, got %d", len(collection.Columns), len(retrieved.Columns))
	}

	// Verify columns
	for i, col := range retrieved.Columns {
		if col.Name != collection.Columns[i].Name {
			t.Errorf("Column %d: expected name %s, got %s", i, collection.Columns[i].Name, col.Name)
		}
		if col.Type != collection.Columns[i].Type {
			t.Errorf("Column %d: expected type %s, got %s", i, collection.Columns[i].Type, col.Type)
		}
	}
}

func TestSchemaRegistry_Set_NilCollection(t *testing.T) {
	registry := NewSchemaRegistry()

	err := registry.Set(nil)
	if err == nil {
		t.Error("Expected error for nil collection, got nil")
	}
}

func TestSchemaRegistry_Set_EmptyName(t *testing.T) {
	registry := NewSchemaRegistry()

	collection := &Collection{
		Name:    "",
		Columns: []Column{},
	}

	err := registry.Set(collection)
	if err == nil {
		t.Error("Expected error for empty collection name, got nil")
	}
}

func TestSchemaRegistry_Get_NonExistent(t *testing.T) {
	registry := NewSchemaRegistry()

	_, ok := registry.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for non-existent collection")
	}
}

func TestSchemaRegistry_Delete(t *testing.T) {
	registry := NewSchemaRegistry()

	collection := &Collection{
		Name: "products",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
			{Name: "name", Type: TypeString},
		},
	}

	registry.Set(collection)

	// Verify it exists
	if !registry.Exists("products") {
		t.Fatal("Collection should exist before delete")
	}

	// Delete
	if err := registry.Delete("products"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	if registry.Exists("products") {
		t.Error("Collection should not exist after delete")
	}
}

func TestSchemaRegistry_Delete_EmptyName(t *testing.T) {
	registry := NewSchemaRegistry()

	err := registry.Delete("")
	if err == nil {
		t.Error("Expected error for empty collection name, got nil")
	}
}

func TestSchemaRegistry_Exists(t *testing.T) {
	registry := NewSchemaRegistry()

	if registry.Exists("test") {
		t.Error("Exists() returned true for non-existent collection")
	}

	collection := &Collection{
		Name:    "test",
		Columns: []Column{},
	}

	registry.Set(collection)

	if !registry.Exists("test") {
		t.Error("Exists() returned false for existing collection")
	}
}

func TestSchemaRegistry_List(t *testing.T) {
	registry := NewSchemaRegistry()

	// Empty registry
	if len(registry.List()) != 0 {
		t.Error("List() should return empty slice for empty registry")
	}

	// Add collections
	collections := []string{"users", "products", "orders"}
	for _, name := range collections {
		registry.Set(&Collection{
			Name:    name,
			Columns: []Column{},
		})
	}

	list := registry.List()
	if len(list) != len(collections) {
		t.Errorf("Expected %d collections, got %d", len(collections), len(list))
	}

	// Verify all collections are in the list
	listMap := make(map[string]bool)
	for _, name := range list {
		listMap[name] = true
	}

	for _, name := range collections {
		if !listMap[name] {
			t.Errorf("Collection %s not found in list", name)
		}
	}
}

func TestSchemaRegistry_GetAll(t *testing.T) {
	registry := NewSchemaRegistry()

	// Add collections
	registry.Set(&Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
		},
	})

	registry.Set(&Collection{
		Name: "products",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
			{Name: "name", Type: TypeString},
		},
	})

	all := registry.GetAll()
	if len(all) != 2 {
		t.Errorf("Expected 2 collections, got %d", len(all))
	}
}

func TestSchemaRegistry_Clear(t *testing.T) {
	registry := NewSchemaRegistry()

	// Add collections
	registry.Set(&Collection{Name: "users", Columns: []Column{}})
	registry.Set(&Collection{Name: "products", Columns: []Column{}})

	if registry.Count() != 2 {
		t.Errorf("Expected count 2 before clear, got %d", registry.Count())
	}

	registry.Clear()

	if registry.Count() != 0 {
		t.Errorf("Expected count 0 after clear, got %d", registry.Count())
	}
}

func TestSchemaRegistry_Count(t *testing.T) {
	registry := NewSchemaRegistry()

	if registry.Count() != 0 {
		t.Errorf("Expected initial count 0, got %d", registry.Count())
	}

	registry.Set(&Collection{Name: "users", Columns: []Column{}})
	if registry.Count() != 1 {
		t.Errorf("Expected count 1, got %d", registry.Count())
	}

	registry.Set(&Collection{Name: "products", Columns: []Column{}})
	if registry.Count() != 2 {
		t.Errorf("Expected count 2, got %d", registry.Count())
	}

	registry.Delete("users")
	if registry.Count() != 1 {
		t.Errorf("Expected count 1 after delete, got %d", registry.Count())
	}
}

func TestSchemaRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewSchemaRegistry()
	var wg sync.WaitGroup

	// Concurrent writes
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("collection_%d", index)
			collection := &Collection{
				Name: name,
				Columns: []Column{
					{Name: "id", Type: TypeInteger},
				},
			}
			registry.Set(collection)
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := fmt.Sprintf("collection_%d", index)
			registry.Get(name)
		}(i)
	}

	wg.Wait()

	// Verify count
	if registry.Count() != 100 {
		t.Errorf("Expected count 100 after concurrent operations, got %d", registry.Count())
	}
}

func TestValidateColumnType(t *testing.T) {
	validTypes := []ColumnType{
		TypeString, TypeInteger, TypeFloat, TypeBoolean, TypeDatetime, TypeText, TypeJSON,
	}

	for _, colType := range validTypes {
		if !ValidateColumnType(colType) {
			t.Errorf("ValidateColumnType(%s) returned false", colType)
		}
	}

	if ValidateColumnType("invalid") {
		t.Error("ValidateColumnType() should return false for invalid type")
	}
}

func TestMapGoTypeToColumnType(t *testing.T) {
	tests := []struct {
		goType       string
		expectedType ColumnType
		expectError  bool
	}{
		{"string", TypeString, false},
		{"int", TypeInteger, false},
		{"int64", TypeInteger, false},
		{"float64", TypeFloat, false},
		{"bool", TypeBoolean, false},
		{"time.Time", TypeDatetime, false},
		{"invalid", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.goType, func(t *testing.T) {
			colType, err := MapGoTypeToColumnType(tt.goType)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if colType != tt.expectedType {
					t.Errorf("Expected type %s, got %s", tt.expectedType, colType)
				}
			}
		})
	}
}

func TestSchemaRegistry_ImmutableCopy(t *testing.T) {
	registry := NewSchemaRegistry()

	original := &Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
		},
	}

	registry.Set(original)

	// Modify original after Set
	original.Name = "modified"
	original.Columns[0].Name = "modified_id"

	// Retrieved should not be affected
	retrieved, _ := registry.Get("users")
	if retrieved.Name != "users" {
		t.Errorf("Expected name 'users', got '%s'", retrieved.Name)
	}
	if retrieved.Columns[0].Name != "id" {
		t.Errorf("Expected column name 'id', got '%s'", retrieved.Columns[0].Name)
	}

	// Modify retrieved should not affect registry
	retrieved.Name = "modified_again"
	retrieved.Columns[0].Name = "modified_id_again"

	retrieved2, _ := registry.Get("users")
	if retrieved2.Name != "users" {
		t.Errorf("Expected name 'users', got '%s'", retrieved2.Name)
	}
	if retrieved2.Columns[0].Name != "id" {
		t.Errorf("Expected column name 'id', got '%s'", retrieved2.Columns[0].Name)
	}
}

// Benchmark tests
func BenchmarkSchemaRegistry_Set(b *testing.B) {
	registry := NewSchemaRegistry()
	collection := &Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
			{Name: "name", Type: TypeString},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Set(collection)
	}
}

func BenchmarkSchemaRegistry_Get(b *testing.B) {
	registry := NewSchemaRegistry()
	collection := &Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
			{Name: "name", Type: TypeString},
		},
	}
	registry.Set(collection)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Get("users")
	}
}

func BenchmarkSchemaRegistry_Exists(b *testing.B) {
	registry := NewSchemaRegistry()
	collection := &Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
		},
	}
	registry.Set(collection)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Exists("users")
	}
}

func BenchmarkSchemaRegistry_ConcurrentReads(b *testing.B) {
	registry := NewSchemaRegistry()
	collection := &Collection{
		Name: "users",
		Columns: []Column{
			{Name: "id", Type: TypeInteger},
		},
	}
	registry.Set(collection)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			registry.Get("users")
		}
	})
}
