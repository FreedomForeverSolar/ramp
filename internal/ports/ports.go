package ports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const (
	DefaultBasePort = 3000
	DefaultMaxPorts = 100
	PortAllocationsFile = "port_allocations.json"
)

type PortAllocations struct {
	allocations map[string]int
	filePath    string
	basePort    int
	maxPorts    int
}

func NewPortAllocations(projectDir string, basePort, maxPorts int) (*PortAllocations, error) {
	if basePort <= 0 {
		basePort = DefaultBasePort
	}
	if maxPorts <= 0 {
		maxPorts = DefaultMaxPorts
	}

	filePath := filepath.Join(projectDir, ".ramp", PortAllocationsFile)
	
	pa := &PortAllocations{
		allocations: make(map[string]int),
		filePath:    filePath,
		basePort:    basePort,
		maxPorts:    maxPorts,
	}

	if err := pa.load(); err != nil {
		return nil, fmt.Errorf("failed to load port allocations: %w", err)
	}

	return pa, nil
}

func (pa *PortAllocations) load() error {
	if _, err := os.Stat(pa.filePath); os.IsNotExist(err) {
		// File doesn't exist yet, that's fine
		return nil
	}

	data, err := os.ReadFile(pa.filePath)
	if err != nil {
		return fmt.Errorf("failed to read port allocations file: %w", err)
	}

	if len(data) == 0 {
		// Empty file, that's fine
		return nil
	}

	if err := json.Unmarshal(data, &pa.allocations); err != nil {
		return fmt.Errorf("failed to parse port allocations file: %w", err)
	}

	return nil
}

func (pa *PortAllocations) save() error {
	// Ensure .ramp directory exists
	if err := os.MkdirAll(filepath.Dir(pa.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create .ramp directory: %w", err)
	}

	data, err := json.MarshalIndent(pa.allocations, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal port allocations: %w", err)
	}

	if err := os.WriteFile(pa.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write port allocations file: %w", err)
	}

	return nil
}

func (pa *PortAllocations) AllocatePort(featureName string) (int, error) {
	// Check if feature already has a port
	if port, exists := pa.allocations[featureName]; exists {
		return port, nil
	}

	// Find the next available port
	port := pa.findNextAvailablePort()
	if port == -1 {
		return 0, fmt.Errorf("no available ports in range %d-%d", pa.basePort, pa.basePort+pa.maxPorts-1)
	}

	pa.allocations[featureName] = port
	if err := pa.save(); err != nil {
		return 0, fmt.Errorf("failed to save port allocation: %w", err)
	}

	return port, nil
}

func (pa *PortAllocations) ReleasePort(featureName string) error {
	if _, exists := pa.allocations[featureName]; !exists {
		// Already released or never allocated
		return nil
	}

	delete(pa.allocations, featureName)
	if err := pa.save(); err != nil {
		return fmt.Errorf("failed to save port allocation after release: %w", err)
	}

	return nil
}

func (pa *PortAllocations) GetPort(featureName string) (int, bool) {
	port, exists := pa.allocations[featureName]
	return port, exists
}

func (pa *PortAllocations) findNextAvailablePort() int {
	// Get all allocated ports and sort them
	allocatedPorts := make([]int, 0, len(pa.allocations))
	for _, port := range pa.allocations {
		allocatedPorts = append(allocatedPorts, port)
	}
	sort.Ints(allocatedPorts)

	// Find the first gap or the next port after all allocated ones
	for port := pa.basePort; port < pa.basePort+pa.maxPorts; port++ {
		if !pa.isPortAllocated(port, allocatedPorts) {
			return port
		}
	}

	return -1 // No available ports
}

func (pa *PortAllocations) isPortAllocated(port int, sortedPorts []int) bool {
	for _, allocatedPort := range sortedPorts {
		if allocatedPort == port {
			return true
		}
		if allocatedPort > port {
			// Since the list is sorted, we can stop here
			break
		}
	}
	return false
}

func (pa *PortAllocations) ListAllocations() map[string]int {
	result := make(map[string]int)
	for feature, port := range pa.allocations {
		result[feature] = port
	}
	return result
}