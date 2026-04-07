package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

func GenerateGoHTML(report *engine.GoProjectReport, outDir string) (string, error) {
	report.GeneratedAt = time.Now().Format(time.RFC3339)
	outPath := filepath.Join(outDir, "dependency-report.html")

	file, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := buildGoHTMLData(report)
	if err != nil {
		return "", err
	}

	if err := reportTemplates.ExecuteTemplate(file, "go", data); err != nil {
		return "", err
	}
	return outPath, nil
}

type goHTMLData struct {
	PageTitle      string
	ReportCSS      template.CSS
	ReportClientJS template.JS
	ProjectPath    string
	ModuleName     string
	GoVersion      string
	GeneratedAt    string
	ModuleCount    int
	ScannedCount   int
	FailedCount    int
	Rows           []goHTMLRow
	ReportJSON     template.JS
}

type goHTMLRow struct {
	Name        string
	RepoURL     string
	ModuleErr   string
	CurrVersion string

	LatestVersion string
	HasLatest     bool

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
}

func buildGoHTMLData(r *engine.GoProjectReport) (goHTMLData, error) {
	d := goHTMLData{
		PageTitle:      fmt.Sprintf("Go Module Analysis – %s", r.ProjectPath),
		ReportCSS:      template.CSS(reportCSS),
		ReportClientJS: template.JS(reportGoJS),
		ProjectPath:    r.ProjectPath,
		ModuleName:     r.ModuleName,
		GoVersion:      r.GoVersion,
		GeneratedAt:    r.GeneratedAt,
		ModuleCount:    len(r.Dependencies),
		ScannedCount:   r.ScannedCount,
		FailedCount:    r.FailedCount,
		Rows:           make([]goHTMLRow, 0, len(r.Dependencies)),
	}
	for _, dep := range r.Dependencies {
		d.Rows = append(d.Rows, goHTMLRowFrom(dep))
	}

	b, err := json.Marshal(r)
	if err != nil {
		return goHTMLData{}, fmt.Errorf("marshal go report JSON: %w", err)
	}
	d.ReportJSON = template.JS(b)

	return d, nil
}

func goHTMLRowFrom(dep engine.GoModuleResult) goHTMLRow {
	row := goHTMLRow{
		Name:        dep.Name,
		RepoURL:     dep.RepoURL,
		ModuleErr:   dep.Error,
		CurrVersion: dep.CurrentVersion,
	}

	if dep.LatestVersion != "" {
		row.LatestVersion = dep.LatestVersion
		row.HasLatest = true
	}

	if dep.TimeSinceLastUpdate != "" {
		row.LastPrimary = dep.TimeSinceLastUpdate
		if dep.LastUpdateDate != "" {
			if t, err := time.Parse(time.RFC3339Nano, dep.LastUpdateDate); err == nil {
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
	}

	return row
}
