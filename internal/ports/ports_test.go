package ports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPortAllocations(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ramp-ports-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .ramp subdirectory
	rampDir := filepath.Join(tempDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("Failed to create .ramp dir: %v", err)
	}

	// Test basic allocation
	pa, err := NewPortAllocations(tempDir, 3000, 10)
	if err != nil {
		t.Fatalf("Failed to create PortAllocations: %v", err)
	}

	// Test first allocation
	port1, err := pa.AllocatePort("feature_a")
	if err != nil {
		t.Fatalf("Failed to allocate port for feature_a: %v", err)
	}
	if port1 != 3000 {
		t.Errorf("Expected port 3000, got %d", port1)
	}

	// Test second allocation
	port2, err := pa.AllocatePort("bug_fix_here")
	if err != nil {
		t.Fatalf("Failed to allocate port for bug_fix_here: %v", err)
	}
	if port2 != 3001 {
		t.Errorf("Expected port 3001, got %d", port2)
	}

	// Test third allocation
	port3, err := pa.AllocatePort("another")
	if err != nil {
		t.Fatalf("Failed to allocate port for another: %v", err)
	}
	if port3 != 3002 {
		t.Errorf("Expected port 3002, got %d", port3)
	}

	// Test re-allocating same feature returns same port
	port1Again, err := pa.AllocatePort("feature_a")
	if err != nil {
		t.Fatalf("Failed to re-allocate port for feature_a: %v", err)
	}
	if port1Again != port1 {
		t.Errorf("Expected same port %d, got %d", port1, port1Again)
	}

	// Test releasing port
	err = pa.ReleasePort("bug_fix_here")
	if err != nil {
		t.Fatalf("Failed to release port for bug_fix_here: %v", err)
	}

	// Test that released port is reused
	port4, err := pa.AllocatePort("new_feature")
	if err != nil {
		t.Fatalf("Failed to allocate port for new_feature: %v", err)
	}
	if port4 != 3001 { // Should reuse the released port
		t.Errorf("Expected reused port 3001, got %d", port4)
	}

	// Test getting port that exists
	port, exists := pa.GetPort("feature_a")
	if !exists {
		t.Error("Expected feature_a port to exist")
	}
	if port != 3000 {
		t.Errorf("Expected port 3000, got %d", port)
	}

	// Test getting port that doesn't exist
	_, exists = pa.GetPort("nonexistent")
	if exists {
		t.Error("Expected nonexistent feature port to not exist")
	}
}

func TestPortAllocationsPersistence(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ramp-ports-persistence-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .ramp subdirectory
	rampDir := filepath.Join(tempDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("Failed to create .ramp dir: %v", err)
	}

	// Create first instance and allocate ports
	pa1, err := NewPortAllocations(tempDir, 3000, 10)
	if err != nil {
		t.Fatalf("Failed to create PortAllocations: %v", err)
	}

	port1, _ := pa1.AllocatePort("feature_a")
	port2, _ := pa1.AllocatePort("feature_b")

	// Create second instance (should load from file)
	pa2, err := NewPortAllocations(tempDir, 3000, 10)
	if err != nil {
		t.Fatalf("Failed to create second PortAllocations: %v", err)
	}

	// Check that ports are loaded correctly
	loadedPort1, exists1 := pa2.GetPort("feature_a")
	if !exists1 || loadedPort1 != port1 {
		t.Errorf("Expected feature_a port %d to be loaded, got %d (exists: %v)", port1, loadedPort1, exists1)
	}

	loadedPort2, exists2 := pa2.GetPort("feature_b")
	if !exists2 || loadedPort2 != port2 {
		t.Errorf("Expected feature_b port %d to be loaded, got %d (exists: %v)", port2, loadedPort2, exists2)
	}

	// Verify the file format
	filePath := filepath.Join(rampDir, PortAllocationsFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read port allocations file: %v", err)
	}

	var allocations map[string]int
	if err := json.Unmarshal(data, &allocations); err != nil {
		t.Fatalf("Failed to parse port allocations file: %v", err)
	}

	if len(allocations) != 2 {
		t.Errorf("Expected 2 allocations in file, got %d", len(allocations))
	}
}

func TestPortAllocationsEdgeCases(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ramp-ports-edge-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with small max ports
	pa, err := NewPortAllocations(tempDir, 3000, 2)
	if err != nil {
		t.Fatalf("Failed to create PortAllocations: %v", err)
	}

	// Allocate all available ports
	port1, _ := pa.AllocatePort("feature_1")
	port2, _ := pa.AllocatePort("feature_2")

	if port1 != 3000 || port2 != 3001 {
		t.Errorf("Expected ports 3000, 3001, got %d, %d", port1, port2)
	}

	// Try to allocate one more (should fail)
	_, err = pa.AllocatePort("feature_3")
	if err == nil {
		t.Error("Expected error when no ports available")
	}

	// Release one port and try again
	err = pa.ReleasePort("feature_1")
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	port3, err := pa.AllocatePort("feature_3")
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}
	if port3 != 3000 {
		t.Errorf("Expected reused port 3000, got %d", port3)
	}
}