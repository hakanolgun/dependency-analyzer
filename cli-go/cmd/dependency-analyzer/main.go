package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/cliui"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/gomod"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/npm"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/platform"
	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/report"
)

func main() {
	var (
		projectPath  = flag.String("project", "", "project path to scan (default: first positional arg, else .)")
		openReport   = flag.Bool("open", true, "open report in browser after generation")
		jsonOut      = flag.Bool("json", false, "print JSON summary to stdout")
		ecosystem    = flag.String("ecosystem", "", "force ecosystem: npm or go (auto-detected if empty)")
		noGhost      = flag.Bool("no-ghost", false, "do not fetch package tarballs when node_modules is missing")
		noRegistry   = flag.Bool("no-registry", false, "skip npm registry metadata (downloads, maintenance, etc.)")
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

	eco := detectEcosystem(absProject, *ecosystem)

	switch eco {
	case "npm":
		runNpm(absProject, *openReport, *jsonOut, *noGhost, *noRegistry)
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

func runGo(absProject string, openReport, jsonOut bool) {
	cliui.Step("Reading go.mod and resolving module graph...")
	projectReport, err := gomod.AnalyzeProject(absProject)
	if err != nil {
		exitErr("analysis failed", err)
	}

	cliui.Step("Generating HTML report...")
	outPath, err := report.GenerateGoHTML(projectReport, absProject)
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
