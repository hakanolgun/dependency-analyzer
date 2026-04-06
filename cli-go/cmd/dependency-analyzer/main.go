package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/gomod"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/npm"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/platform"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/report"
)

func main() {
	var (
		projectPath = flag.String("project", ".", "project path to scan")
		openReport  = flag.Bool("open", true, "open report in browser after generation")
		jsonOut     = flag.Bool("json", false, "print JSON summary to stdout")
		ecosystem   = flag.String("ecosystem", "", "force ecosystem: npm or go (auto-detected if empty)")
	)
	flag.Parse()

	absProject, err := filepath.Abs(*projectPath)
	if err != nil {
		exitErr("failed to resolve project path", err)
	}

	eco := detectEcosystem(absProject, *ecosystem)

	switch eco {
	case "npm":
		runNpm(absProject, *openReport, *jsonOut)
	case "go":
		runGo(absProject, *openReport, *jsonOut)
	default:
		fmt.Fprintf(os.Stderr, "dependency-analyzer: could not detect ecosystem in %s (no package.json or go.mod found)\n", absProject)
		fmt.Fprintf(os.Stderr, "  hint: use --ecosystem npm or --ecosystem go to override\n")
		os.Exit(1)
	}
}

func detectEcosystem(projectPath, override string) string {
	if override != "" {
		switch override {
		case "npm", "go":
			return override
		default:
			fmt.Fprintf(os.Stderr, "dependency-analyzer: unknown ecosystem %q (use npm or go)\n", override)
			os.Exit(1)
		}
	}

	hasGoMod := fileExists(filepath.Join(projectPath, "go.mod"))
	hasPkgJSON := fileExists(filepath.Join(projectPath, "package.json"))

	switch {
	case hasGoMod && hasPkgJSON:
		fmt.Fprintf(os.Stderr, "dependency-analyzer: found both go.mod and package.json, defaulting to npm\n")
		fmt.Fprintf(os.Stderr, "  hint: use --ecosystem go to analyze Go modules instead\n")
		return "npm"
	case hasGoMod:
		return "go"
	case hasPkgJSON:
		return "npm"
	default:
		return ""
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func runNpm(absProject string, openReport, jsonOut bool) {
	projectReport, err := npm.AnalyzeProject(absProject)
	if err != nil {
		exitErr("analysis failed", err)
	}

	outPath, err := report.GenerateHTML(projectReport, absProject)
	if err != nil {
		exitErr("report generation failed", err)
	}
	projectReport.ReportOutPath = outPath

	printNpmSummary(projectReport)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(projectReport)
	}

	if openReport {
		_ = platform.OpenBrowser(outPath)
	}
}

func runGo(absProject string, openReport, jsonOut bool) {
	projectReport, err := gomod.AnalyzeProject(absProject)
	if err != nil {
		exitErr("analysis failed", err)
	}

	outPath, err := report.GenerateGoHTML(projectReport, absProject)
	if err != nil {
		exitErr("report generation failed", err)
	}
	projectReport.ReportOutPath = outPath

	printGoSummary(projectReport)

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(projectReport)
	}

	if openReport {
		_ = platform.OpenBrowser(outPath)
	}
}

func printNpmSummary(projectReport *engine.ProjectReport) {
	fmt.Printf("\nDep-Scan NPM scan completed\n")
	fmt.Printf("Project: %s\n", projectReport.ProjectPath)
	fmt.Printf("Scanned: %d, Failed: %d\n", projectReport.ScannedCount, projectReport.FailedCount)
	fmt.Printf("Report:  %s\n\n", projectReport.ReportOutPath)

	for _, dep := range projectReport.Dependencies {
		if dep.Error != "" {
			fmt.Printf("- %s (%s): ERROR - %s\n", dep.Name, dep.Version, dep.Error)
			continue
		}
		fmt.Printf(
			"- %s (%s): %d/100 %s | native=%.2f volume=%.2f api=%.2f ent=%.2f logic=%.2f\n",
			dep.Name,
			dep.Version,
			dep.Score,
			dep.Label,
			dep.Metrics.Native,
			dep.Metrics.Volume,
			dep.Metrics.APISurface,
			dep.Metrics.Entanglement,
			dep.Metrics.LogicComplexity,
		)
	}
}

func printGoSummary(projectReport *engine.GoProjectReport) {
	fmt.Printf("\nDep-Scan Go module scan completed\n")
	fmt.Printf("Module:  %s (go %s)\n", projectReport.ModuleName, projectReport.GoVersion)
	fmt.Printf("Project: %s\n", projectReport.ProjectPath)
	fmt.Printf("Scanned: %d, Failed: %d\n", projectReport.ScannedCount, projectReport.FailedCount)
	fmt.Printf("Report:  %s\n\n", projectReport.ReportOutPath)

	for _, dep := range projectReport.Dependencies {
		if dep.Error != "" {
			fmt.Printf("- %s (%s): ERROR - %s\n", dep.Name, dep.CurrentVersion, dep.Error)
			continue
		}
		fmt.Printf(
			"- %s (%s): %d/100 %s | native=%.2f volume=%.2f api=%.2f ent=%.2f logic=%.2f\n",
			dep.Name,
			dep.CurrentVersion,
			dep.Score,
			dep.Label,
			dep.Metrics.Native,
			dep.Metrics.Volume,
			dep.Metrics.APISurface,
			dep.Metrics.Entanglement,
			dep.Metrics.LogicComplexity,
		)
	}
}

func exitErr(message string, err error) {
	fmt.Fprintf(os.Stderr, "dependency-analyzer: %s: %v\n", message, err)
	os.Exit(1)
}
