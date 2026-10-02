package main

import (
	"fmt"
	"os"

	"bandit/pkg/cli"
	"bandit/pkg/config"
)

func main() {
	cfg := config.LoadConfig()
	if err := cli.RunCLI(cfg); err != nil {
		fmt.Printf("Fatal error starting Bandit CLI: %v\n", err)
		os.Exit(1)
	}
}
