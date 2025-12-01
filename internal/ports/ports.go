package ports

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	DefaultBasePort = 3000
	DefaultMaxPorts = 100
	PortAllocationsFile = "port_allocations.json"
)

type PortAllocations struct {
	allocations map[string][]int
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
		allocations: make(map[string][]int),
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

	// Try to unmarshal into new format first (map[string][]int)
	var newFormat map[string][]int
	if err := json.Unmarshal(data, &newFormat); err == nil {
		// Successfully parsed as new format
		pa.allocations = newFormat
		return nil
	}

	// Fall back to old format (map[string]int) and migrate
	var oldFormat map[string]int
	if err := json.Unmarshal(data, &oldFormat); err != nil {
		return fmt.Errorf("failed to parse port allocations file: %w", err)
	}

	// Convert old format to new format
	pa.allocations = make(map[string][]int)
	for feature, port := range oldFormat {
		pa.allocations[feature] = []int{port}
	}

	// Save in new format for future loads
	return pa.save()
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

func (pa *PortAllocations) AllocatePort(featureName string, count int) ([]int, error) {
	// Check if feature already has ports
	if ports, exists := pa.allocations[featureName]; exists {
		return ports, nil
	}

	// Find N consecutive available ports
	ports := pa.findNextAvailablePorts(count)
	if len(ports) < count {
		return nil, fmt.Errorf("insufficient available ports (need %d, found %d) in range %d-%d",
			count, len(ports), pa.basePort, pa.basePort+pa.maxPorts-1)
	}

	pa.allocations[featureName] = ports
	if err := pa.save(); err != nil {
		return nil, fmt.Errorf("failed to save port allocation: %w", err)
	}

	return ports, nil
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

func (pa *PortAllocations) GetPorts(featureName string) ([]int, bool) {
	ports, exists := pa.allocations[featureName]
	return ports, exists
}

func (pa *PortAllocations) GetPort(featureName string) (int, bool) {
	ports, exists := pa.allocations[featureName]
	if !exists || len(ports) == 0 {
		return 0, false
	}
	return ports[0], true
}

func (pa *PortAllocations) findNextAvailablePorts(count int) []int {
	// Flatten all allocated ports into a map for O(1) lookup
	allocatedPorts := make(map[int]bool)
	for _, ports := range pa.allocations {
		for _, port := range ports {
			allocatedPorts[port] = true
		}
	}

	// Find N consecutive available ports
	result := make([]int, 0, count)
	for port := pa.basePort; port < pa.basePort+pa.maxPorts && len(result) < count; port++ {
		if !allocatedPorts[port] {
			result = append(result, port)
		}
	}

	return result
}

func (pa *PortAllocations) ListAllocations() map[string][]int {
	result := make(map[string][]int)
	for feature, ports := range pa.allocations {
		// Copy slice to avoid external modifications
		result[feature] = append([]int{}, ports...)
	}
	return result
}