package cmd

import (
	"testing"
)

// TestVersionCommand tests the version command
func TestVersionCommand(t *testing.T) {
	// The version command just prints the version, we can't easily test stdout
	// but we can verify the version variable is set
	if version == "" {
		t.Error("version should not be empty")
	}
}

// TestVersionValue tests the version value
func TestVersionValue(t *testing.T) {
	// Version should default to "dev"
	if version != "dev" {
		t.Logf("version = %q (expected 'dev' but may be set via build flag)", version)
	}
}
