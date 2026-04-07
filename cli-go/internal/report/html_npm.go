package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

func GenerateJsHTML(report *engine.ProjectReport, outDir string) (string, error) {
	report.GeneratedAt = time.Now().Format(time.RFC3339)
	outPath := filepath.Join(outDir, "dependency-report.html")

	file, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := buildHTMLData(report)
	if err != nil {
		return "", err
	}

	if err := reportTemplates.ExecuteTemplate(file, "npm", data); err != nil {
		return "", err
	}
	return outPath, nil
}

type htmlData struct {
	PageTitle      string
	ReportCSS      template.CSS
	ReportClientJS template.JS
	ProjectPath    string
	GeneratedAt    string
	HasReactNative bool
	PackageCount   int
	ScannedCount   int
	FailedCount    int
	Rows           []htmlRow
	ReportJSON     template.JS
}

type htmlRow struct {
	Name        string
	RepoURL     string
	PackageErr  string
	CurrVersion string

	LatestVersion string
	HasLatest     bool

	WeeklyDownloads string

	LastPrimary   string
	LastSecondary string
	ShowLast      bool

	MaintainedClass string
	MaintainedText  string
	ShowMaintained  bool

	ReplaceShow   bool
	RepLabel      string
	RepLabelClass string
	Score         int
	ConfClass     string
	ConfText      string
	ConfShow      bool

	NewArchCell string // "ok", "no", "dash"
}

func buildHTMLData(r *engine.ProjectReport) (htmlData, error) {
	d := htmlData{
		PageTitle:      fmt.Sprintf("Dependency Analysis – %s", r.ProjectPath),
		ReportCSS:      template.CSS(reportCSS),
		ReportClientJS: template.JS(reportNpmJS),
		ProjectPath:    r.ProjectPath,
		GeneratedAt:    r.GeneratedAt,
		HasReactNative: r.HasReactNative,
		PackageCount:   len(r.Dependencies),
		ScannedCount:   r.ScannedCount,
		FailedCount:    r.FailedCount,
		Rows:           make([]htmlRow, 0, len(r.Dependencies)),
	}
	for _, dep := range r.Dependencies {
		d.Rows = append(d.Rows, htmlRowFrom(dep, r.HasReactNative))
	}

	b, err := json.Marshal(r)
	if err != nil {
		return htmlData{}, fmt.Errorf("marshal report JSON: %w", err)
	}
	d.ReportJSON = template.JS(b)

	return d, nil
}

func htmlRowFrom(dep engine.DependencyResult, hasRNRoot bool) htmlRow {
	row := htmlRow{
		Name:        dep.Name,
		RepoURL:     dep.RepoURL,
		PackageErr:  dep.Error,
		CurrVersion: stripRange(dep.Version),
	}

	if dep.LatestVersion != "" {
		row.LatestVersion = dep.LatestVersion
		row.HasLatest = true
	}

	row.WeeklyDownloads = formatDownloadsK(dep.WeeklyDownloads)

	if dep.TimeSinceLastUpdate != "" {
		row.LastPrimary = dep.TimeSinceLastUpdate
		if dep.LastUpdateDate != "" {
			if t, err := time.Parse(time.RFC3339, dep.LastUpdateDate); err == nil {
				row.LastSecondary = t.Format("1/2/2006")
			} else {
				row.LastSecondary = dep.LastUpdateDate
			}
		}
		row.ShowLast = true
	}

	if dep.IsMaintained != "" {
		row.ShowMaintained = true
		row.MaintainedClass = maintainedBadgeClass(dep.IsMaintained)
		row.MaintainedText = maintainedDisplay(dep.IsMaintained)
	}

	if dep.Error == "" {
		row.ReplaceShow = true
		lbl, cls := replaceabilityTier(dep.Score)
		row.RepLabel = lbl
		row.RepLabelClass = cls
		row.Score = dep.Score
		switch strings.ToLower(dep.Confidence) {
		case "high":
			row.ConfShow = true
			row.ConfClass = "success"
			row.ConfText = "High"
		case "medium":
			row.ConfShow = true
			row.ConfClass = "warning"
			row.ConfText = "Medium"
		case "low":
			row.ConfShow = true
			row.ConfClass = "danger"
			row.ConfText = "Low"
		}
	}

	if hasRNRoot {
		row.NewArchCell = newArchCell(dep)
	} else {
		row.NewArchCell = ""
	}

	return row
}

func newArchCell(dep engine.DependencyResult) string {
	if dep.NewArchitecture == nil {
		return "dash"
	}
	if *dep.NewArchitecture {
		return "ok"
	}
	return "no"
}

func stripRange(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "^")
	v = strings.TrimPrefix(v, "~")
	return v
}

func formatDownloadsK(p *int) string {
	if p == nil {
		return "-"
	}
	return fmt.Sprintf("%d K", *p/1000)
}

func maintainedBadgeClass(m string) string {
	switch m {
	case "yes":
		return "success"
	case "unlikely":
		return "warning"
	case "no":
		return "danger"
	default:
		return "info"
	}
}

func maintainedDisplay(m string) string {
	switch m {
	case "yes":
		return "Yes"
	case "unlikely":
		return "Unlikely"
	case "no":
		return "No"
	default:
		return m
	}
}

func replaceabilityTier(score int) (label string, badgeClass string) {
	if score >= 71 {
		return "Hard", "danger"
	}
	if score >= 31 {
		return "Medium", "warning"
	}
	return "Easy", "success"
}
