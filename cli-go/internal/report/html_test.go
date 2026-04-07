package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hakanolgun/dependency-analyzer/cli-go/internal/engine"
)

func TestGenerateHTML(t *testing.T) {
	tmp := t.TempDir()
	r := &engine.ProjectReport{
		ProjectPath:    tmp,
		ScannedCount:   1,
		HasReactNative: true,
		Dependencies: []engine.DependencyResult{
			{
				Name:                "pkg-a",
				Version:             "^1.0.0",
				Score:               42,
				Label:               "MEDIUM",
				Metrics:             engine.Metrics{},
				LatestVersion:       "1.1.0",
				RepoURL:             "https://www.npmjs.com/package/pkg-a",
				WeeklyDownloads:     ptrInt(1200),
				LastUpdateDate:      "2025-06-01T12:00:00Z",
				TimeSinceLastUpdate: "3 months ago",
				IsMaintained:        "yes",
			},
		},
	}

	out, err := GenerateJsHTML(r, tmp)
	if err != nil {
		t.Fatalf("GenerateHTML err: %v", err)
	}
	if filepath.Base(out) != "dependency-report.html" {
		t.Fatalf("unexpected report file: %s", out)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("report read: %v", err)
	}
	s := string(b)
	for _, needle := range []string{
		`class="glass-panel"`,
		`class="table-container"`,
		"Package Name",
		"Your Version",
		"Latest Version",
		"Weekly Downloads",
		"Last Update",
		"Maintained",
		"Replaceability",
		"New Arch Support",
		"Maintenance Status Key",
		"Replaceability Score",
	} {
		if !strings.Contains(s, needle) {
			t.Fatalf("report missing %q", needle)
		}
	}
}

func ptrInt(n int) *int {
	return &n
}
