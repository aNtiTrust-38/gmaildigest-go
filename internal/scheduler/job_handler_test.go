package scheduler

import (
	"context"
	"testing"
)

func TestJobHandlerRegistry_Basic(t *testing.T) {
	registry := NewJobHandlerRegistry()

	// Test registering and getting a handler
	testHandler := func(ctx context.Context, job *Job) error {
		return nil
	}

	registry.RegisterHandler("test_type", testHandler)
	handler := registry.GetHandler("test_type")
	if handler == nil {
		t.Error("Expected to get registered handler, got nil")
	}

	// Test getting non-existent handler
	handler = registry.GetHandler("non_existent")
	if handler != nil {
		t.Error("Expected nil for non-existent handler")
	}

	// Test unregistering handler
	registry.UnregisterHandler("test_type")
	handler = registry.GetHandler("test_type")
	if handler != nil {
		t.Error("Handler should be nil after unregistering")
	}
}

func TestJobHandlerRegistry_ListHandlerTypes(t *testing.T) {
	registry := NewJobHandlerRegistry()

	// Register multiple handlers
	testHandler := func(ctx context.Context, job *Job) error {
		return nil
	}

	registry.RegisterHandler("type1", testHandler)
	registry.RegisterHandler("type2", testHandler)
	registry.RegisterHandler("type3", testHandler)

	// Test listing handler types
	types := registry.ListHandlerTypes()
	if len(types) != 3 {
		t.Errorf("Expected 3 handler types, got %d", len(types))
	}

	// Verify all types are present
	typeMap := make(map[string]bool)
	for _, t := range types {
		typeMap[t] = true
	}

	expectedTypes := []string{"type1", "type2", "type3"}
	for _, expected := range expectedTypes {
		if !typeMap[expected] {
			t.Errorf("Expected type %s not found in handler types", expected)
		}
	}
}

func TestJobHandlerRegistry_InvalidInputs(t *testing.T) {
	registry := NewJobHandlerRegistry()

	// Test registering with empty type
	registry.RegisterHandler("", func(ctx context.Context, job *Job) error {
		return nil
	})
	if handler := registry.GetHandler(""); handler != nil {
		t.Error("Should not register handler with empty type")
	}

	// Test registering with nil handler
	registry.RegisterHandler("test_type", nil)
	if handler := registry.GetHandler("test_type"); handler != nil {
		t.Error("Should not register nil handler")
	}

	// Test unregistering non-existent type
	registry.UnregisterHandler("non_existent")
	// Should not panic
}

func TestJobHandlerRegistry_Concurrency(t *testing.T) {
	registry := NewJobHandlerRegistry()
	done := make(chan bool)

	// Test concurrent registration and lookup
	go func() {
		for i := 0; i < 100; i++ {
			registry.RegisterHandler("test_type", func(ctx context.Context, job *Job) error {
				return nil
			})
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.GetHandler("test_type")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			registry.UnregisterHandler("test_type")
		}
		done <- true
	}()

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}
	// Should not deadlock or panic
} 