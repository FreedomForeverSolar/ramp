package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"ramp/cmd"
	"github.com/spf13/cobra/doc"
)

func main() {
	// Get the docs directory path
	docsDir := filepath.Join("docs", "commands")

	// Create the commands directory if it doesn't exist
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		log.Fatalf("Failed to create docs directory: %v", err)
	}

	// Get the root command
	rootCmd := cmd.GetRootCmd()

	// Configure identity for link handling
	identity := func(s string) string { return s }
	emptyStr := func(s string) string { return "" }

	// Generate markdown documentation
	if err := doc.GenMarkdownTreeCustom(rootCmd, docsDir, emptyStr, identity); err != nil {
		log.Fatalf("Failed to generate documentation: %v", err)
	}

	fmt.Printf("✅ Command documentation generated in %s/\n", docsDir)

	// List generated files
	files, err := os.ReadDir(docsDir)
	if err != nil {
		log.Fatalf("Failed to read docs directory: %v", err)
	}

	fmt.Println("\nGenerated files:")
	for _, file := range files {
		if !file.IsDir() {
			fmt.Printf("  - %s\n", file.Name())
		}
	}
}
