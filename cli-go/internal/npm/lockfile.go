package npm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ResolveLockedVersion returns the exact version recorded in a lockfile for a direct dependency.
// Tries package-lock.json, pnpm-lock.yaml, then yarn.lock. ok is false if not found.
func ResolveLockedVersion(projectPath, packageName string) (version string, ok bool) {
	if v, ok := npmLockResolvedVersion(projectPath, packageName); ok {
		return v, true
	}
	if v, ok := pnpmLockResolvedVersion(projectPath, packageName); ok {
		return v, true
	}
	if v, ok := yarnLockResolvedVersion(projectPath, packageName); ok {
		return v, true
	}
	return "", false
}

// PackageJSONVersionUsableForGhost returns true if the specifier is likely an exact npm version
// (no range operators), so the registry can fetch it without a lockfile.
func PackageJSONVersionUsableForGhost(spec string) bool {
	s := strings.TrimSpace(spec)
	if s == "" {
		return false
	}
	if strings.ContainsAny(s, "^~*><=|") {
		return false
	}
	// workspace:, file:, link:, npm:, etc.
	if strings.Contains(s, ":") {
		return false
	}
	return true
}

// --- npm package-lock.json ---

type npmLockFile struct {
	LockfileVersion int `json:"lockfileVersion"`
	Packages        map[string]struct {
		Version string `json:"version"`
	} `json:"packages"`
	Dependencies map[string]npmLockV1Node `json:"dependencies"`
}

type npmLockV1Node struct {
	Version      string                    `json:"version"`
	Dependencies map[string]npmLockV1Node `json:"dependencies"`
}

func npmLockResolvedVersion(projectPath, packageName string) (string, bool) {
	p := filepath.Join(projectPath, "package-lock.json")
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	var lock npmLockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return "", false
	}
	key := "node_modules/" + packageName
	if lock.LockfileVersion >= 2 && lock.Packages != nil {
		if ent, ok := lock.Packages[key]; ok && ent.Version != "" {
			return ent.Version, true
		}
		return "", false
	}
	if lock.Dependencies != nil {
		if ent, ok := walkNpmV1(lock.Dependencies, packageName); ok {
			return ent.Version, true
		}
	}
	return "", false
}

func walkNpmV1(nodes map[string]npmLockV1Node, name string) (npmLockV1Node, bool) {
	if nodes == nil {
		return npmLockV1Node{}, false
	}
	n, ok := nodes[name]
	if !ok {
		return npmLockV1Node{}, false
	}
	if n.Version != "" {
		return n, true
	}
	return npmLockV1Node{}, false
}

// --- pnpm pnpm-lock.yaml ---

type pnpmLockRoot struct {
	Importers map[string]pnpmImporter `yaml:"importers"`
}

type pnpmImporter struct {
	Dependencies    map[string]pnpmDepRef `yaml:"dependencies"`
	DevDependencies map[string]pnpmDepRef `yaml:"devDependencies"`
}

type pnpmDepRef struct {
	Version   string `yaml:"version"`
	Specifier string `yaml:"specifier"`
}

func pnpmLockResolvedVersion(projectPath, packageName string) (string, bool) {
	p := filepath.Join(projectPath, "pnpm-lock.yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	var root pnpmLockRoot
	if err := yaml.Unmarshal(data, &root); err != nil {
		return "", false
	}
	if root.Importers == nil {
		return "", false
	}
	// Prefer root importer "."
	imp, ok := root.Importers["."]
	if !ok {
		for _, i := range root.Importers {
			if v, ok := pnpmImporterLookup(i, packageName); ok {
				return v, true
			}
		}
		return "", false
	}
	return pnpmImporterLookup(imp, packageName)
}

func pnpmImporterLookup(imp pnpmImporter, packageName string) (string, bool) {
	if ref, ok := imp.Dependencies[packageName]; ok {
		if v := normalizePNPMVersion(ref.Version); v != "" {
			return v, true
		}
	}
	if ref, ok := imp.DevDependencies[packageName]; ok {
		if v := normalizePNPMVersion(ref.Version); v != "" {
			return v, true
		}
	}
	return "", false
}

func normalizePNPMVersion(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '('); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// --- yarn classic yarn.lock ---

func yarnLockResolvedVersion(projectPath, packageName string) (string, bool) {
	p := filepath.Join(projectPath, "yarn.lock")
	data, err := os.ReadFile(p)
	if err != nil {
		return "", false
	}
	return parseYarnLockForPackage(data, packageName)
}

func parseYarnLockForPackage(data []byte, packageName string) (string, bool) {
	content := string(data)
	// Yarn Berry may start with __metadata — not supported here; no classic blocks.
	if strings.HasPrefix(strings.TrimSpace(content), "__metadata") {
		return "", false
	}
	blocks := strings.Split(content, "\n\n")
	needle := packageName + "@"
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" || strings.HasPrefix(block, "#") {
			continue
		}
		lines := strings.Split(block, "\n")
		if len(lines) < 2 {
			continue
		}
		head := strings.TrimSpace(lines[0])
		if !strings.Contains(head, needle) {
			continue
		}
		for _, line := range lines[1:] {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "version ") {
				rest := strings.TrimSpace(strings.TrimPrefix(line, "version"))
				rest = strings.Trim(rest, `"'`)
				if rest != "" {
					return rest, true
				}
			}
		}
	}
	return "", false
}

var errNoLockedVersion = errors.New("no locked version")

func resolveVersionForGhost(projectPath, packageName, packageJSONSpec string) (string, error) {
	if v, ok := ResolveLockedVersion(projectPath, packageName); ok {
		return v, nil
	}
	if PackageJSONVersionUsableForGhost(packageJSONSpec) {
		return strings.TrimSpace(packageJSONSpec), nil
	}
	return "", errNoLockedVersion
}
