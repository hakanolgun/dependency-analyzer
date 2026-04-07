package npm

import (
	"os"
	"path/filepath"
)

// InstallLayout describes how JavaScript dependencies are materialized on disk.
type InstallLayout int

const (
	// LayoutNodeModules indicates a non-empty node_modules tree (npm, pnpm, yarn classic, etc.).
	LayoutNodeModules InstallLayout = iota
	// LayoutYarnPnP means Yarn Plug'n'Play (.pnp.cjs) without a usable node_modules install.
	LayoutYarnPnP
	// LayoutMissingNodeModules means no node_modules (or empty); analysis may use ghost fetch.
	LayoutMissingNodeModules
)

// DetectInstallLayout classifies the project for UX messaging and default strategies.
// Per-dependency resolution still tries local node_modules first when the directory exists.
func DetectInstallLayout(projectPath string) InstallLayout {
	hasPnP := fileExists(filepath.Join(projectPath, ".pnp.cjs")) ||
		fileExists(filepath.Join(projectPath, ".pnp.js"))

	nm := filepath.Join(projectPath, "node_modules")
	entries, err := os.ReadDir(nm)
	hasNM := err == nil && len(entries) > 0

	if hasPnP && !hasNM {
		return LayoutYarnPnP
	}
	if !hasNM {
		return LayoutMissingNodeModules
	}
	return LayoutNodeModules
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
