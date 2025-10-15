package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/config"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <config-file>\n", os.Args[0])
		os.Exit(1)
	}

	configPath := os.Args[1]

	debug := len(os.Args) > 2 && os.Args[2] == "-debug"

	// Create a new runtime config with the specified config path
	runtime := config.New().WithConfigPath(configPath)

	if debug {
		fmt.Fprintf(os.Stderr, "ConfigPath: %s\n", runtime.ConfigPath)
		fmt.Fprintf(os.Stderr, "ConfigDir: %s\n", runtime.ConfigDir)
	}

	// Load the configuration (this will process includes)
	err := runtime.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Marshal the final configuration to YAML
	output, err := yaml.Marshal(runtime.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config to YAML: %v\n", err)
		os.Exit(1)
	}

	// Print to stdout
	fmt.Print(string(output))
}
