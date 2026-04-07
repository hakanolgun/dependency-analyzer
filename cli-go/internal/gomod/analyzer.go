package gomod

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/cliui"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

var (
	// Native detection patterns
	cgoImportRe    = regexp.MustCompile(`import\s+"C"`)
	syscallImportRe = regexp.MustCompile(`"syscall"`)
	unsafeImportRe = regexp.MustCompile(`"unsafe"`)
	cgoExportRe    = regexp.MustCompile(`(?m)^//export\s+\w+`)

	// API surface patterns: exported funcs, types, structs, interfaces
	exportedFuncRe   = regexp.MustCompile(`(?m)^func\s+([A-Z]\w*)`)
	exportedMethodRe = regexp.MustCompile(`(?m)^func\s+\([^)]+\)\s+([A-Z]\w*)`)
	exportedStructRe = regexp.MustCompile(`(?m)^type\s+([A-Z]\w*)\s+struct\b`)
	exportedIfaceRe  = regexp.MustCompile(`(?m)^type\s+([A-Z]\w*)\s+interface\b`)
	exportedTypeRe   = regexp.MustCompile(`(?m)^type\s+([A-Z]\w*)\s+`)
	emptyIfaceRe     = regexp.MustCompile(`interface\s*\{\s*\}`)

	// Logic complexity patterns (cognitive/cyclomatic proxies)
	decisionRe     = regexp.MustCompile(`\b(if|else\s+if|for|switch|select|case|default)\b`)
	goRoutineRe    = regexp.MustCompile(`\bgo\s+func\b`)
	chanRe         = regexp.MustCompile(`\bchan\s+`)
	chanArrowRe    = regexp.MustCompile(`<-`)
	deferRe        = regexp.MustCompile(`\bdefer\b`)

	// Shell leak / entanglement
	osExecImportRe = regexp.MustCompile(`"os/exec"`)
	execCommandRe  = regexp.MustCompile(`exec\.Command`)
	reflectImportRe = regexp.MustCompile(`"reflect"`)

	// Import extraction (non-std)
	importBlockRe   = regexp.MustCompile(`(?s)import\s*\((.*?)\)`)
	singleImportRe  = regexp.MustCompile(`(?m)^import\s+"([^"]+)"`)

	// Standard library prefixes heuristic — Go stdlib packages don't have dots in first segment
	// External: anything with a dot (e.g. github.com/...)
)

// AnalyzeProject reads go.mod from projectPath and analyzes all direct dependencies.
func AnalyzeProject(projectPath string) (*engine.GoProjectReport, error) {
	return AnalyzeProjectWithClient(projectPath, NewClient())
}

// AnalyzeProjectWithClient allows injecting a custom proxy client (for tests).
func AnalyzeProjectWithClient(projectPath string, client *Client) (*engine.GoProjectReport, error) {
	goModPath := filepath.Join(projectPath, "go.mod")
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, fmt.Errorf("go.mod not found in target directory: %w", err)
	}

	parsed, err := ParseGoMod(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	if len(parsed.Dependencies) == 0 {
		return nil, fmt.Errorf("no direct dependencies found in go.mod")
	}

	cliui.Step("Fetching module sources from the Go proxy and analyzing replaceability...")

	ctx := context.Background()
	results := make([]engine.GoModuleResult, 0, len(parsed.Dependencies))
	failures := 0

	for _, dep := range parsed.Dependencies {
		result := analyzeDependency(ctx, client, dep.Path, dep.Version)
		if result.Error != "" {
			failures++
		}
		results = append(results, result)
	}

	report := &engine.GoProjectReport{
		ProjectPath:  projectPath,
		ModuleName:   parsed.ModuleName,
		GoVersion:    parsed.GoVersion,
		Dependencies: results,
		ScannedCount: len(results) - failures,
		FailedCount:  failures,
	}
	return report, nil
}

func analyzeDependency(ctx context.Context, client *Client, modulePath, version string) engine.GoModuleResult {
	res := engine.GoModuleResult{
		Name:           modulePath,
		CurrentVersion: version,
	}

	// Fetch metadata (latest version, maintained status, repo URL)
	meta, metaErr := client.FetchModuleMeta(ctx, modulePath)
	if metaErr == nil {
		res.LatestVersion = meta.LatestVersion
		res.LastUpdateDate = meta.LastUpdateDate
		res.TimeSinceLastUpdate = meta.TimeSinceLastUpdate
		res.IsMaintained = meta.IsMaintained
		res.RepoURL = meta.RepoURL
	}

	// Determine which version to download for source analysis
	// Use the version pinned in go.mod
	zipVersion := version

	// Download module source zip for replaceability analysis
	zipReader, zipErr := client.FetchModuleZip(ctx, modulePath, zipVersion)
	if zipErr != nil {
		// Fallback: still report metadata but mark scoring as failed
		res.Error = fmt.Sprintf("source download failed: %v", zipErr)
		return res
	}

	// Analyze source for replaceability
	metrics := analyzeSourceZip(zipReader)

	// Fetch the dependency's own go.mod for entanglement enrichment
	depModContent, depModErr := client.FetchModuleMod(ctx, modulePath, zipVersion)
	if depModErr == nil {
		enrichEntanglement(&metrics, depModContent)
	}

	norm := engine.ComputeNormalized(metrics)
	res.Normalized = norm
	res.Score = engine.ToPercentageScore(norm)
	res.Label = engine.ScoreLabel(norm)
	res.Metrics = metrics

	return res
}

// sourceStats accumulates data across all .go files in a module zip.
type sourceStats struct {
	sloc             int
	exportedFuncs    int
	exportedMethods  int
	exportedStructs  int
	exportedIfaces   int
	exportedTypes    int
	emptyIfaceCount  int
	maxBraceDepth    int
	hasCgo           bool
	hasSyscall       bool
	hasUnsafe        bool
	hasCgoExport     bool
	hasOsExec        bool
	hasExecCommand   bool
	hasReflect       bool
	decisionCount    int
	goRoutineCount   int
	chanCount        int
	chanArrowCount   int
	deferCount       int
	nonStdImports    int
	externalModules  int
	hasCFiles        bool
	totalFiles       int
	goFileCount      int
	testFileCount    int
}

func analyzeSourceZip(zr *zip.Reader) engine.Metrics {
	var stats sourceStats

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}

		name := filepath.Base(f.Name)
		ext := strings.ToLower(filepath.Ext(name))
		lowerName := strings.ToLower(name)
		stats.totalFiles++

		// Detect C files (cgo native signals)
		if ext == ".c" || ext == ".h" || ext == ".cc" || ext == ".cpp" || ext == ".s" {
			stats.hasCFiles = true
		}

		// Only analyze .go files
		if ext != ".go" {
			continue
		}

		// Skip test files for main analysis (but count them)
		if strings.HasSuffix(lowerName, "_test.go") {
			stats.testFileCount++
			continue
		}

		stats.goFileCount++

		rc, err := f.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		src := string(content)

		analyzeGoSource(&stats, src)
	}

	return computeGoMetrics(&stats)
}

func analyzeGoSource(stats *sourceStats, src string) {
	// Strip comments for SLOC
	cleaned := stripGoComments(src)
	lines := strings.Split(cleaned, "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			stats.sloc++
		}
	}

	// Native presence
	if cgoImportRe.MatchString(src) {
		stats.hasCgo = true
	}
	if syscallImportRe.MatchString(src) {
		stats.hasSyscall = true
	}
	if unsafeImportRe.MatchString(src) {
		stats.hasUnsafe = true
	}
	if cgoExportRe.MatchString(src) {
		stats.hasCgoExport = true
	}

	// API surface
	stats.exportedFuncs += len(exportedFuncRe.FindAllString(src, -1))
	stats.exportedMethods += len(exportedMethodRe.FindAllString(src, -1))
	stats.exportedStructs += len(exportedStructRe.FindAllString(src, -1))
	stats.exportedIfaces += len(exportedIfaceRe.FindAllString(src, -1))
	stats.exportedTypes += len(exportedTypeRe.FindAllString(src, -1))
	stats.emptyIfaceCount += len(emptyIfaceRe.FindAllString(src, -1))

	// Brace depth (structural complexity)
	depth := computeMaxBraceDepth(src)
	if depth > stats.maxBraceDepth {
		stats.maxBraceDepth = depth
	}

	// Logic complexity
	stats.decisionCount += len(decisionRe.FindAllString(src, -1))
	stats.goRoutineCount += len(goRoutineRe.FindAllString(src, -1))
	stats.chanCount += len(chanRe.FindAllString(src, -1))
	stats.chanArrowCount += len(chanArrowRe.FindAllString(src, -1))
	stats.deferCount += len(deferRe.FindAllString(src, -1))

	// Shell leak / entanglement signals
	if osExecImportRe.MatchString(src) {
		stats.hasOsExec = true
	}
	if execCommandRe.MatchString(src) {
		stats.hasExecCommand = true
	}
	if reflectImportRe.MatchString(src) {
		stats.hasReflect = true
	}

	// Count imports (non-std)
	extractImports(stats, src)
}

func extractImports(stats *sourceStats, src string) {
	// Multi-import blocks
	blocks := importBlockRe.FindAllStringSubmatch(src, -1)
	for _, block := range blocks {
		if len(block) < 2 {
			continue
		}
		lines := strings.Split(block[1], "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			trimmed = strings.Trim(trimmed, "\"")
			// Remove alias if present
			if idx := strings.LastIndex(trimmed, " "); idx > 0 {
				trimmed = strings.TrimSpace(trimmed[idx:])
				trimmed = strings.Trim(trimmed, "\"")
			}
			if trimmed == "" || trimmed == "C" {
				continue
			}
			if isExternalImport(trimmed) {
				stats.externalModules++
			} else if !isStdLibImport(trimmed) {
				stats.nonStdImports++
			}
		}
	}

	// Single-line imports
	singles := singleImportRe.FindAllStringSubmatch(src, -1)
	for _, m := range singles {
		if len(m) < 2 {
			continue
		}
		path := m[1]
		if path == "C" {
			continue
		}
		if isExternalImport(path) {
			stats.externalModules++
		} else if !isStdLibImport(path) {
			stats.nonStdImports++
		}
	}
}

// isExternalImport checks if an import path refers to an external module (contains a dot in first segment).
func isExternalImport(importPath string) bool {
	parts := strings.SplitN(importPath, "/", 2)
	return strings.Contains(parts[0], ".")
}

// isStdLibImport is a heuristic: standard library packages don't have dots in them.
func isStdLibImport(importPath string) bool {
	return !strings.Contains(importPath, ".")
}

// enrichEntanglement parses the dependency's own go.mod and counts its require lines.
func enrichEntanglement(m *engine.Metrics, goModContent string) {
	parsed, err := ParseGoMod(goModContent)
	if err != nil {
		return
	}
	// "require" entries in the dep's go.mod = transitive dependency count
	directDeps := len(parsed.Dependencies)
	// Add to the entanglement calculation
	// Score = (non_std_imports × 1.5) + (external_modules × 2) — already in base
	// Here we add the dependency's own deps as additional entanglement signal
	additionalEntanglement := clamp(float64(directDeps) * 1.5 / 40.0)
	m.Entanglement = clamp(m.Entanglement + additionalEntanglement)
}

func computeGoMetrics(stats *sourceStats) engine.Metrics {
	var m engine.Metrics

	// ── Native Presence (0.40 weight) ──
	// CGO, syscall, unsafe, or C files → 1.0
	if stats.hasCgo || stats.hasCFiles {
		m.Native = 1.0
	} else if stats.hasSyscall || stats.hasUnsafe {
		m.Native = 1.0
	}
	// Foreign contract boost: //export locks to 1.0
	if stats.hasCgoExport {
		m.Native = 1.0
	}

	// ── Code Volume (0.10 weight) ──
	m.Volume = engine.VolumeScore(stats.sloc)

	// ── API Surface (0.10 weight) ──
	// Score = (exported_functions × 1) + (exported_structs × avg_methods × 0.5) + (exported_interfaces × 0.5)
	avgMethods := 0.0
	if stats.exportedStructs > 0 {
		avgMethods = float64(stats.exportedMethods) / float64(stats.exportedStructs)
	}
	apiRaw := float64(stats.exportedFuncs) +
		float64(stats.exportedStructs)*avgMethods*0.5 +
		float64(stats.exportedIfaces)*0.5
	api := clamp(apiRaw / 50.0)
	// Structural penalty: +10% per nesting level of anonymous structs / interface{} usage
	depthPenalty := float64(stats.emptyIfaceCount) * 0.1
	api = clamp(api * (1.0 + float64(stats.maxBraceDepth)*0.1 + depthPenalty))
	m.APISurface = api

	// ── Entanglement (0.15 weight) ──
	// Score = (non_std_imports × 1.5) + (external_modules × 2)
	entanglementRaw := float64(stats.nonStdImports)*1.5 + float64(stats.externalModules)*2.0
	m.Entanglement = clamp(entanglementRaw / 40.0)
	// Shell leak penalty
	if stats.hasOsExec {
		m.Entanglement = clamp(m.Entanglement + 0.1)
	}

	// ── Logic Complexity (0.25 weight) ──
	// Base: cognitive complexity proxy = decision points + concurrency features
	cognitiveRaw := stats.decisionCount +
		stats.goRoutineCount*3 + // goroutines are significantly harder to replace
		stats.chanCount*2 +
		stats.chanArrowCount +
		stats.deferCount
	logic := clamp(float64(cognitiveRaw) / 50.0)
	// Shell leak penalty
	if stats.hasExecCommand {
		logic = clamp(logic + 0.1)
	}
	// reflect/unsafe modifier
	if stats.hasReflect {
		logic = clamp(logic + 0.1)
	}
	if stats.hasUnsafe {
		logic = clamp(logic + 0.1)
	}
	// Test coverage discount
	if stats.testFileCount > 5 {
		logic = clamp(logic - 0.1)
	}
	m.LogicComplexity = logic

	return m
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

func stripGoComments(src string) string {
	multiLine := regexp.MustCompile(`(?s)/\*.*?\*/`)
	noMulti := multiLine.ReplaceAllString(src, "")
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
