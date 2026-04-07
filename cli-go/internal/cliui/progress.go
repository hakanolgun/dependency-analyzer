package cliui

import (
	"fmt"
	"os"
)

// Step prints a human-readable progress line to stderr so stdout can be used for --json.
func Step(msg string) {
	fmt.Fprintf(os.Stderr, "dependency-analyzer: %s\n", msg)
}
