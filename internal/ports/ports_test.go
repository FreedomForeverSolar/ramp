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

	// Test first allocation (single port)
	ports1, err := pa.AllocatePort("feature_a", 1)
	if err != nil {
		t.Fatalf("Failed to allocate port for feature_a: %v", err)
	}
	if len(ports1) != 1 || ports1[0] != 3000 {
		t.Errorf("Expected [3000], got %v", ports1)
	}

	// Test second allocation (single port)
	ports2, err := pa.AllocatePort("bug_fix_here", 1)
	if err != nil {
		t.Fatalf("Failed to allocate port for bug_fix_here: %v", err)
	}
	if len(ports2) != 1 || ports2[0] != 3001 {
		t.Errorf("Expected [3001], got %v", ports2)
	}

	// Test third allocation (single port)
	ports3, err := pa.AllocatePort("another", 1)
	if err != nil {
		t.Fatalf("Failed to allocate port for another: %v", err)
	}
	if len(ports3) != 1 || ports3[0] != 3002 {
		t.Errorf("Expected [3002], got %v", ports3)
	}

	// Test re-allocating same feature returns same ports
	ports1Again, err := pa.AllocatePort("feature_a", 1)
	if err != nil {
		t.Fatalf("Failed to re-allocate port for feature_a: %v", err)
	}
	if len(ports1Again) != 1 || ports1Again[0] != ports1[0] {
		t.Errorf("Expected same ports %v, got %v", ports1, ports1Again)
	}

	// Test releasing port
	err = pa.ReleasePort("bug_fix_here")
	if err != nil {
		t.Fatalf("Failed to release port for bug_fix_here: %v", err)
	}

	// Test that released port is reused
	ports4, err := pa.AllocatePort("new_feature", 1)
	if err != nil {
		t.Fatalf("Failed to allocate port for new_feature: %v", err)
	}
	if len(ports4) != 1 || ports4[0] != 3001 { // Should reuse the released port
		t.Errorf("Expected [3001], got %v", ports4)
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

	ports1, _ := pa1.AllocatePort("feature_a", 1)
	ports2, _ := pa1.AllocatePort("feature_b", 1)

	// Create second instance (should load from file)
	pa2, err := NewPortAllocations(tempDir, 3000, 10)
	if err != nil {
		t.Fatalf("Failed to create second PortAllocations: %v", err)
	}

	// Check that ports are loaded correctly
	loadedPort1, exists1 := pa2.GetPort("feature_a")
	if !exists1 || loadedPort1 != ports1[0] {
		t.Errorf("Expected feature_a port %d to be loaded, got %d (exists: %v)", ports1[0], loadedPort1, exists1)
	}

	loadedPort2, exists2 := pa2.GetPort("feature_b")
	if !exists2 || loadedPort2 != ports2[0] {
		t.Errorf("Expected feature_b port %d to be loaded, got %d (exists: %v)", ports2[0], loadedPort2, exists2)
	}

	// Verify the file format
	filePath := filepath.Join(rampDir, PortAllocationsFile)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read port allocations file: %v", err)
	}

	var allocations map[string][]int
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
	ports1, _ := pa.AllocatePort("feature_1", 1)
	ports2, _ := pa.AllocatePort("feature_2", 1)

	if len(ports1) != 1 || ports1[0] != 3000 || len(ports2) != 1 || ports2[0] != 3001 {
		t.Errorf("Expected [3000], [3001], got %v, %v", ports1, ports2)
	}

	// Try to allocate one more (should fail)
	_, err = pa.AllocatePort("feature_3", 1)
	if err == nil {
		t.Error("Expected error when no ports available")
	}

	// Release one port and try again
	err = pa.ReleasePort("feature_1")
	if err != nil {
		t.Fatalf("Failed to release port: %v", err)
	}

	ports3, err := pa.AllocatePort("feature_3", 1)
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}
	if len(ports3) != 1 || ports3[0] != 3000 {
		t.Errorf("Expected [3000], got %v", ports3)
	}
}

func TestMultiPortAllocation(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ramp-ports-multi-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	pa, err := NewPortAllocations(tempDir, 3000, 10)
	if err != nil {
		t.Fatalf("Failed to create PortAllocations: %v", err)
	}

	// Allocate 3 ports for feature-a
	ports, err := pa.AllocatePort("feature-a", 3)
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	if len(ports) != 3 {
		t.Errorf("Expected 3 ports, got %d", len(ports))
	}

	// Check ports are consecutive
	if ports[0] != 3000 || ports[1] != 3001 || ports[2] != 3002 {
		t.Errorf("Expected consecutive ports [3000, 3001, 3002], got %v", ports)
	}

	// Allocate single port for another feature
	ports2, err := pa.AllocatePort("feature-b", 1)
	if err != nil {
		t.Fatalf("Failed to allocate port: %v", err)
	}

	if len(ports2) != 1 || ports2[0] != 3003 {
		t.Errorf("Expected [3003], got %v", ports2)
	}

	// Allocate 2 more ports
	ports3, err := pa.AllocatePort("feature-c", 2)
	if err != nil {
		t.Fatalf("Failed to allocate ports: %v", err)
	}

	if len(ports3) != 2 || ports3[0] != 3004 || ports3[1] != 3005 {
		t.Errorf("Expected [3004, 3005], got %v", ports3)
	}

	// Test GetPorts method
	retrievedPorts, exists := pa.GetPorts("feature-a")
	if !exists {
		t.Error("Expected feature-a to exist")
	}
	if len(retrievedPorts) != 3 || retrievedPorts[0] != 3000 {
		t.Errorf("Expected [3000, 3001, 3002], got %v", retrievedPorts)
	}

	// Test GetPort method returns first port for backward compatibility
	firstPort, exists := pa.GetPort("feature-a")
	if !exists || firstPort != 3000 {
		t.Errorf("Expected GetPort to return 3000, got %d", firstPort)
	}
}

func TestPortAllocationsMigration(t *testing.T) {
	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "ramp-ports-migration-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .ramp subdirectory
	rampDir := filepath.Join(tempDir, ".ramp")
	if err := os.MkdirAll(rampDir, 0755); err != nil {
		t.Fatalf("Failed to create .ramp dir: %v", err)
	}

	// Write old format file
	oldFormat := map[string]int{
		"feature-a": 3000,
		"feature-b": 3001,
		"feature-c": 3005,
	}
	oldData, _ := json.MarshalIndent(oldFormat, "", "  ")
	filePath := filepath.Join(rampDir, PortAllocationsFile)
	if err := os.WriteFile(filePath, oldData, 0644); err != nil {
		t.Fatalf("Failed to write old format file: %v", err)
	}

	// Load with new code (should migrate automatically)
	pa, err := NewPortAllocations(tempDir, 3000, 100)
	if err != nil {
		t.Fatalf("Failed to create PortAllocations: %v", err)
	}

	// Verify old data was migrated
	port, exists := pa.GetPort("feature-a")
	if !exists || port != 3000 {
		t.Errorf("Expected feature-a port 3000, got %d (exists: %v)", port, exists)
	}

	ports, exists := pa.GetPorts("feature-b")
	if !exists || len(ports) != 1 || ports[0] != 3001 {
		t.Errorf("Expected feature-b ports [3001], got %v", ports)
	}

	// Verify file was saved in new format
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read migrated file: %v", err)
	}

	var newFormat map[string][]int
	if err := json.Unmarshal(data, &newFormat); err != nil {
		t.Fatalf("Failed to parse as new format: %v", err)
	}

	if len(newFormat["feature-a"]) != 1 || newFormat["feature-a"][0] != 3000 {
		t.Errorf("Expected feature-a to be [3000] in new format, got %v", newFormat["feature-a"])
	}
}