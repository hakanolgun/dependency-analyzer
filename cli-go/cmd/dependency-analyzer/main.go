package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/cliui"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/npm"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/platform"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/report"
)

func main() {
	var (
		projectPath = flag.String("project", "", "project path to scan (default: first positional arg, else .)")
		openReport  = flag.Bool("open", true, "open report in browser after generation")
		jsonOut     = flag.Bool("json", false, "print JSON summary to stdout")
		ecosystem   = flag.String("ecosystem", "", "must be npm or empty (Go module analysis was removed)")
		noGhost     = flag.Bool("no-ghost", false, "do not fetch package tarballs when node_modules is missing")
		noRegistry  = flag.Bool("no-registry", false, "skip npm registry metadata (downloads, maintenance, etc.)")
	)
	flag.Parse()

	proj := *projectPath
	if proj == "" {
		if flag.NArg() > 0 {
			proj = flag.Arg(0)
		} else {
			proj = "."
		}
	}

	absProject, err := filepath.Abs(proj)
	if err != nil {
		exitErr("failed to resolve project path", err)
	}

	if err := validateEcosystem(*ecosystem); err != nil {
		fmt.Fprintf(os.Stderr, "dependency-analyzer: %v\n", err)
		os.Exit(1)
	}

	if !fileExists(filepath.Join(absProject, "package.json")) {
		fmt.Fprintf(os.Stderr, "dependency-analyzer: no package.json in %s\n", absProject)
		fmt.Fprintf(os.Stderr, "  this tool analyzes npm/JavaScript projects only\n")
		os.Exit(1)
	}

	runNpm(absProject, *openReport, *jsonOut, *noGhost, *noRegistry)
}

func validateEcosystem(override string) error {
	switch override {
	case "":
		return nil
	case "npm":
		return nil
	case "go":
		return fmt.Errorf("Go module analysis is no longer supported (npm/JavaScript only)")
	default:
		return fmt.Errorf("unknown ecosystem %q (use npm or leave empty)", override)
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func runNpm(absProject string, openReport, jsonOut, noGhost, noRegistry bool) {
	cliui.Step("Reading package.json and resolving dependencies...")
	opts := npm.DefaultAnalyzeOptions()
	if noGhost {
		opts.AllowGhost = false
	}
	var reg *npm.Client
	if !noRegistry {
		reg = npm.NewClient()
	}
	projectReport, err := npm.AnalyzeProjectWithRegistryOpts(absProject, reg, opts)
	if err != nil {
		exitErr("analysis failed", err)
	}

	cliui.Step("Generating HTML report...")
	outPath, err := report.GenerateJsHTML(projectReport, absProject)
	if err != nil {
		exitErr("report generation failed", err)
	}
	projectReport.ReportOutPath = outPath

	if jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(projectReport)
	}

	if openReport {
		_ = platform.OpenBrowser(outPath)
	}
}

func exitErr(message string, err error) {
	fmt.Fprintf(os.Stderr, "dependency-analyzer: %s: %v\n", message, err)
	os.Exit(1)
}
