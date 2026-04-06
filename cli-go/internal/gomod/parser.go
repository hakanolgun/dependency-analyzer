package gomod

import (
	"errors"
	"strings"
)

// GoModDependency represents a single dependency entry from go.mod.
type GoModDependency struct {
	Path    string
	Version string
}

// GoModParseResult holds the parsed content of a go.mod file.
type GoModParseResult struct {
	ModuleName   string
	GoVersion    string
	Dependencies []GoModDependency
}

// ParseGoMod parses go.mod content and extracts direct dependencies.
// Indirect dependencies (lines with "// indirect") are skipped.
func ParseGoMod(content string) (*GoModParseResult, error) {
	lines := strings.Split(content, "\n")

	var moduleName, goVersion string
	var deps []GoModDependency

	// Extract module name
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "module ") {
			moduleName = strings.TrimSpace(strings.TrimPrefix(trimmed, "module "))
			break
		}
	}
	if moduleName == "" {
		return nil, errors.New("no module declaration found in go.mod")
	}

	// Extract go version
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "go ") && !strings.HasPrefix(trimmed, "golang") {
			goVersion = strings.TrimSpace(strings.TrimPrefix(trimmed, "go "))
			break
		}
	}

	inRequireBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip indirect dependencies
		if strings.Contains(trimmed, "// indirect") {
			continue
		}

		// Single-line require: require github.com/pkg/errors v0.9.1
		if strings.HasPrefix(trimmed, "require ") && !strings.Contains(trimmed, "(") {
			parts := strings.Fields(strings.TrimPrefix(trimmed, "require "))
			if len(parts) >= 2 {
				deps = append(deps, GoModDependency{Path: parts[0], Version: parts[1]})
			}
			continue
		}

		// Block require start
		if trimmed == "require (" || strings.HasPrefix(trimmed, "require (") {
			inRequireBlock = true
			continue
		}

		// Block require end
		if inRequireBlock && trimmed == ")" {
			inRequireBlock = false
			continue
		}

		// Inside require block
		if inRequireBlock && len(trimmed) > 0 && !strings.HasPrefix(trimmed, "//") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				deps = append(deps, GoModDependency{Path: parts[0], Version: parts[1]})
			}
		}
	}

	return &GoModParseResult{
		ModuleName:   moduleName,
		GoVersion:    goVersion,
		Dependencies: deps,
	}, nil
}
