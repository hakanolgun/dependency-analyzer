package npm

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/cliui"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

var (
	nativeDirSignals  = []string{"ios", "android", "cpp"}
	nativeFileSignals = []string{
		".node", ".cpp", ".cc", ".c", ".h", ".hpp", ".m", ".mm", ".swift", ".kt", ".java",
	}
	jsExts = map[string]struct{}{
		".js": {}, ".mjs": {}, ".cjs": {}, ".ts": {}, ".tsx": {},
	}
	exportRe          = regexp.MustCompile(`\bexport\b`)
	classExportRe     = regexp.MustCompile(`\bexport\s+class\s+`)
	publicMethodRe    = regexp.MustCompile(`\b(public\s+)?[A-Za-z_][A-Za-z0-9_]*\s*\([^)]*\)\s*\{`)
	interfaceExportRe = regexp.MustCompile(`\bexport\s+interface\s+`)
	childProcessRe    = regexp.MustCompile(`\b(child_process|execa)\b`)
	shellExecRe       = regexp.MustCompile(`\.(spawn|exec)\s*\(`)
	decisionRe        = regexp.MustCompile(`\b(if|else\s+if|for|while|switch|case|catch|\?\s*[^:]+\s*:)\b`)
	blackMagicRe      = regexp.MustCompile(`(eval\s*\(|new\s+Function\s*\(|WebAssembly|regexp|RegExp)`)
)

type rootPackageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

type depPackageJSON struct {
	Dependencies     map[string]string `json:"dependencies"`
	PeerDependencies map[string]string `json:"peerDependencies"`
}

// AnalyzeProject runs a local DRSE scan and enriches rows with npm registry metadata (requires network).
func AnalyzeProject(projectPath string) (*engine.ProjectReport, error) {
	return AnalyzeProjectWithRegistry(projectPath, NewClient())
}

// AnalyzeProjectWithRegistry allows tests to pass nil registry to skip network calls.
func AnalyzeProjectWithRegistry(projectPath string, reg *Client) (*engine.ProjectReport, error) {
	pkgPath := filepath.Join(projectPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil, errors.New("package.json not found in target directory")
	}

	var root rootPackageJSON
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, errors.New("failed to parse root package.json")
	}

	merged := mergeDeps(&root)
	if len(merged) == 0 {
		return nil, errors.New("no dependencies or devDependencies in package.json")
	}

	names := sortedKeys(merged)
	_, hasReactNative := merged["react-native"]

	cliui.Step("Scanning node_modules and analyzing dependency code (volume, API surface, complexity)...")

	ctx := context.Background()
	results := make([]engine.DependencyResult, len(names))
	failures := 0

	for i, name := range names {
		version := merged[name]
		result := analyzeDependency(projectPath, name, version)
		if result.Error == "" {
			result.Confidence = "high"
		} else {
			failures++
			result.Confidence = "low"
		}
		results[i] = result
	}

	if reg != nil {
		cliui.Step("Fetching package metadata from the npm registry...")
		for i, name := range names {
			result := &results[i]
			meta, err := reg.FetchRegistryMeta(ctx, name, hasReactNative)
			if err == nil {
				applyRegistryMeta(result, meta, hasReactNative)
				if result.Error != "" {
					result.Confidence = "medium"
				}
			} else if result.Error != "" {
				result.Confidence = "low"
			}
		}
	}

	scanned := len(results) - failures

	report := &engine.ProjectReport{
		ProjectPath:    projectPath,
		Dependencies:   results,
		ScannedCount:   scanned,
		FailedCount:    failures,
		HasReactNative: hasReactNative,
	}
	return report, nil
}

func mergeDeps(root *rootPackageJSON) map[string]string {
	merged := make(map[string]string)
	if root.Dependencies != nil {
		for k, v := range root.Dependencies {
			merged[k] = v
		}
	}
	if root.DevDependencies != nil {
		for k, v := range root.DevDependencies {
			merged[k] = v
		}
	}
	return merged
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func applyRegistryMeta(res *engine.DependencyResult, meta RegistryMeta, hasRNRoot bool) {
	if meta.LatestVersion != "" {
		res.LatestVersion = meta.LatestVersion
	}
	if meta.RepoURL != "" {
		res.RepoURL = meta.RepoURL
	}
	if meta.WeeklyDownloads != nil {
		res.WeeklyDownloads = meta.WeeklyDownloads
	}
	if meta.LastUpdateDate != "" {
		res.LastUpdateDate = meta.LastUpdateDate
	}
	if meta.TimeSinceLastUpdate != "" {
		res.TimeSinceLastUpdate = meta.TimeSinceLastUpdate
	}
	if meta.IsMaintained != "" {
		res.IsMaintained = meta.IsMaintained
	}
	if hasRNRoot {
		res.IsReactNativeLib = meta.IsReactNativeLib
		res.NewArchitecture = meta.NewArchitecture
	}
}

func analyzeDependency(projectPath, depName, reqVersion string) engine.DependencyResult {
	res := engine.DependencyResult{Name: depName, Version: reqVersion}
	depPath := filepath.Join(projectPath, "node_modules", depName)

	if _, err := os.Stat(depPath); err != nil {
		res.Error = "dependency directory missing in node_modules"
		return res
	}

	metrics, err := collectMetrics(depPath)
	if err != nil {
		res.Error = err.Error()
		return res
	}

	norm := engine.ComputeNormalized(metrics)
	res.Normalized = norm
	res.Score = engine.ToPercentageScore(norm)
	res.Label = engine.ScoreLabel(norm)
	res.Metrics = metrics

	return res
}

func collectMetrics(depPath string) (engine.Metrics, error) {
	var m engine.Metrics
	pkgJSONPath := filepath.Join(depPath, "package.json")
	pkgJSONBytes, err := os.ReadFile(pkgJSONPath)
	if err != nil {
		return m, errors.New("package.json missing in dependency")
	}

	var depPkg depPackageJSON
	if err := json.Unmarshal(pkgJSONBytes, &depPkg); err != nil {
		return m, errors.New("invalid dependency package.json")
	}

	var (
		sloc             int
		exportCount      int
		classCount       int
		methodCount      int
		interfaceCount   int
		maxDepth         int
		cyclomaticCount  int
		hasShellImport   bool
		hasShellExec     bool
		hasBlackMagic    bool
		hasNativeSignals bool
		hasBindingGyp    bool
		testFileCount    int
	)

	err = filepath.WalkDir(depPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		name := d.Name()
		lowerName := strings.ToLower(name)
		if d.IsDir() {
			if lowerName == "node_modules" {
				return filepath.SkipDir
			}
			for _, signal := range nativeDirSignals {
				if lowerName == signal {
					hasNativeSignals = true
				}
			}
			if lowerName == "__tests__" || lowerName == "tests" || lowerName == "test" {
				testFileCount++
			}
			return nil
		}

		if lowerName == "binding.gyp" {
			hasBindingGyp = true
		}

		ext := strings.ToLower(filepath.Ext(name))
		for _, signal := range nativeFileSignals {
			if ext == signal {
				hasNativeSignals = true
				break
			}
		}

		if _, ok := jsExts[ext]; !ok {
			return nil
		}

		bytes, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		src := stripComments(string(bytes))
		lines := strings.Split(src, "\n")
		for _, l := range lines {
			if strings.TrimSpace(l) != "" {
				sloc++
			}
		}

		exportCount += len(exportRe.FindAllStringIndex(src, -1))
		classCount += len(classExportRe.FindAllStringIndex(src, -1))
		methodCount += len(publicMethodRe.FindAllStringIndex(src, -1))
		interfaceCount += len(interfaceExportRe.FindAllStringIndex(src, -1))

		depth := computeMaxBraceDepth(src)
		if depth > maxDepth {
			maxDepth = depth
		}

		cyclomaticCount += len(decisionRe.FindAllStringIndex(src, -1))
		hasShellImport = hasShellImport || childProcessRe.MatchString(src)
		hasShellExec = hasShellExec || shellExecRe.MatchString(src)
		hasBlackMagic = hasBlackMagic || blackMagicRe.MatchString(src)
		return nil
	})
	if err != nil {
		return m, err
	}

	directDeps := len(depPkg.Dependencies)
	peerDeps := len(depPkg.PeerDependencies)
	transitiveDepth := estimateTransitiveDepth(depPath)

	native := 0.0
	if hasNativeSignals || hasBindingGyp {
		native = 1.0
	}

	apiRaw := float64(exportCount) + float64(classCount)*avgPublicMethods(classCount, methodCount)*0.5 + float64(interfaceCount)*0.5
	api := clamp(apiRaw / 50.0)
	api = clamp(api * (1.0 + float64(maxDepth)*0.1))

	entanglementRaw := float64(directDeps) + float64(transitiveDepth)*2.0 + float64(peerDeps)*2.0
	entanglement := clamp(entanglementRaw / 40.0)
	if hasShellImport {
		entanglement = clamp(entanglement + 0.1)
	}

	logic := clamp(float64(cyclomaticCount) / 50.0)
	if hasShellExec {
		logic = clamp(logic + 0.1)
	}
	if hasBlackMagic {
		logic = clamp(logic + 0.1)
	}
	if testFileCount > 5 {
		logic = clamp(logic - 0.1)
	}

	m = engine.Metrics{
		Native:          native,
		Volume:          engine.VolumeScore(sloc),
		APISurface:      api,
		Entanglement:    entanglement,
		LogicComplexity: logic,
	}
	return m, nil
}

func avgPublicMethods(classCount, methodCount int) float64 {
	if classCount == 0 {
		return 0
	}
	return float64(methodCount) / float64(classCount)
}

func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func stripComments(input string) string {
	multiLine := regexp.MustCompile(`(?s)/\*.*?\*/`)
	noMulti := multiLine.ReplaceAllString(input, "")
	singleLine := regexp.MustCompile(`(?m)//.*$`)
	return singleLine.ReplaceAllString(noMulti, "")
}

func computeMaxBraceDepth(src string) int {
	depth := 0
	maxDepth := 0
	for _, ch := range src {
		switch ch {
		case '{':
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
		case '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return maxDepth
}

func estimateTransitiveDepth(depPath string) int {
	nm := filepath.Join(depPath, "node_modules")
	info, err := os.Stat(nm)
	if err != nil || !info.IsDir() {
		return 0
	}
	maxDepth := 0
	_ = filepath.WalkDir(nm, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}
		rel, relErr := filepath.Rel(nm, path)
		if relErr != nil || rel == "." {
			return nil
		}
		segments := strings.Split(rel, string(filepath.Separator))
		depth := len(segments)
		if depth > maxDepth {
			maxDepth = depth
		}
		return nil
	})
	if maxDepth == 0 {
		return 0
	}
	return maxDepth
}
