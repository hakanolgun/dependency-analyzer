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

// GenerateHTML writes dep-report.html using the same layout, styling and
// interactivity as the web NPM results view.
func GenerateHTML(report *engine.ProjectReport, outDir string) (string, error) {
	report.GeneratedAt = time.Now().Format(time.RFC3339)
	outPath := filepath.Join(outDir, "dep-report.html")

	file, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	data, err := buildHTMLData(report)
	if err != nil {
		return "", err
	}

	tpl := template.Must(template.New("report").Parse(reportTemplate))

	if err := tpl.Execute(file, data); err != nil {
		return "", err
	}
	return outPath, nil
}

type htmlData struct {
	ProjectPath    string
	GeneratedAt    string
	HasReactNative bool
	PackageCount   int
	ScannedCount   int
	FailedCount    int
	Rows           []htmlRow
	// JSON blob embedded in <script> for client-side sorting & download.
	ReportJSON template.JS
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

	// Embed the full report as JSON for the client-side JS.
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

func titleCaseLabel(s string) string {
	switch strings.ToUpper(s) {
	case "EASY":
		return "Easy"
	case "MEDIUM":
		return "Medium"
	case "HARD":
		return "Hard"
	default:
		return s
	}
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

const reportTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Dependency Analysis – {{ .ProjectPath }}</title>
  <link rel="preconnect" href="https://fonts.googleapis.com" />
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700&display=swap" rel="stylesheet" />
  <style>
` + embeddedReportCSS + `
  </style>
</head>
<body>
  <div id="root">
    <div class="app-container">

      <!-- ── HEADER ─────────────────────────────────────────────── -->
      <div class="glass-panel header-panel">
        <h1 class="title">Dependency Analyzer</h1>
        <h2 class="subtitle repo-subtitle">
          <a href="https://github.com/hakanolgun/dependency-analyzer"
             target="_blank" rel="noreferrer" class="repo-link subtitle-link">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none"
                 stroke="currentColor" stroke-width="2"
                 stroke-linecap="round" stroke-linejoin="round">
              <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
            </svg>
            Free and Open Source
          </a>
        </h2>
        <p class="subtitle">Offline report generated by the CLI tool</p>
      </div>

      <!-- ── RESULTS PANEL ──────────────────────────────────────── -->
      <div class="glass-panel">
        <div class="header-actions">
          <div>
            <h2 class="title" style="text-align:left;font-size:2rem;margin-bottom:0.2rem;">Analysis Complete</h2>
            <p class="subtitle" style="text-align:left;margin:0;">
              Analyzed {{ .PackageCount }} packages
              {{ if .HasReactNative }}<span class="badge info" style="margin-left:0.5rem;">React Native Project Detected</span>{{ end }}
            </p>
            <p class="subtitle" style="text-align:left;margin-top:0.5rem;font-size:0.9rem;">
              Project: {{ .ProjectPath }} &middot; Generated {{ .GeneratedAt }} &middot; Scanned {{ .ScannedCount }} &middot; Failed {{ .FailedCount }}
            </p>
          </div>
          <div style="display:flex;gap:0.5rem;flex-shrink:0;">
            <button id="btn-download" class="btn" style="background:rgba(255,255,255,0.1);">
              <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
                   stroke="currentColor" stroke-width="2"
                   stroke-linecap="round" stroke-linejoin="round">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                <polyline points="7 10 12 15 17 10"></polyline>
                <line x1="12" y1="15" x2="12" y2="3"></line>
              </svg>
              Export JSON
            </button>
          </div>
        </div>

        <div class="table-container">
          <table id="results-table">
            <thead>
              <tr>
                <th data-col="name" class="sortable">
                  Package Name <span class="sort-icon" data-for="name"></span>
                </th>
                <th>Your Version</th>
                <th>Latest Version</th>
                <th data-col="weeklyDownloads" class="sortable">
                  Weekly Downloads <span class="sort-icon" data-for="weeklyDownloads"></span>
                </th>
                <th data-col="lastUpdateDate" class="sortable">
                  Last Update <span class="sort-icon" data-for="lastUpdateDate"></span>
                </th>
                <th data-col="isMaintained" class="sortable">
                  Maintained <span class="sort-icon" data-for="isMaintained"></span>
                </th>
                <th data-col="score" class="sortable">
                  Replaceability <span class="sort-icon" data-for="score"></span>
                </th>
                {{ if .HasReactNative }}
                <th data-col="newArchitecture" class="sortable">
                  New Arch Support <span class="sort-icon" data-for="newArchitecture"></span>
                </th>
                {{ end }}
              </tr>
            </thead>
            <tbody id="results-body">
            {{ range .Rows }}
              <tr>
                <td>
                  <div class="pkg-name">
                    {{ if .RepoURL }}
                      <a href="{{ .RepoURL }}" target="_blank" rel="noreferrer" class="repo-link" title="{{ .RepoURL }}">{{ .Name }} <span class="ext">↗</span></a>
                    {{ else }}
                      <span style="word-break:break-all;">{{ .Name }}</span>
                    {{ end }}
                  </div>
                  {{ if .PackageErr }}
                    <span class="pkg-err">{{ .PackageErr }}</span>
                  {{ end }}
                </td>
                <td><span class="badge gray">{{ .CurrVersion }}</span></td>
                <td>
                  {{ if .HasLatest }}<span class="badge info">{{ .LatestVersion }}</span>{{ else }}<span class="muted">-</span>{{ end }}
                </td>
                <td>{{ .WeeklyDownloads }}</td>
                <td>
                  {{ if .ShowLast }}
                    <div class="stack">
                      <span>{{ .LastPrimary }}</span>
                      <span class="sub">{{ .LastSecondary }}</span>
                    </div>
                  {{ else }}
                    <span class="muted">-</span>
                  {{ end }}
                </td>
                <td>
                  {{ if .ShowMaintained }}
                    <span class="badge {{ .MaintainedClass }}">{{ .MaintainedText }}</span>
                  {{ else }}
                    <span class="muted">-</span>
                  {{ end }}
                </td>
                <td class="cell-replace">
                  {{ if .ReplaceShow }}
                    <div class="replace-stack">
                      <span class="badge {{ .RepLabelClass }}">{{ .RepLabel }}</span>
                      <span class="sub">{{ .Score }}/100</span>
                      {{ if .ConfShow }}<span class="badge {{ .ConfClass }} conf">{{ .ConfText }}</span>{{ end }}
                    </div>
                  {{ else }}
                    <span class="muted">-</span>
                  {{ end }}
                </td>
                {{ if $.HasReactNative }}
                <td class="cell-center">
                  {{ if eq .NewArchCell "ok" }}<span class="na-yes">Yes</span>{{ else if eq .NewArchCell "no" }}<span class="na-no">No</span>{{ else }}<span class="muted">-</span>{{ end }}
                </td>
                {{ end }}
              </tr>
            {{ end }}
            </tbody>
          </table>
        </div>
      </div>

      <!-- ── MAINTENANCE LEGEND ─────────────────────────────────── -->
      <div class="maintenance-legend-section">
        <div class="legend-container">
          <h3 class="legend-title">Maintenance Status Key</h3>
          <div class="legend-grid">
            <div class="legend-item">
              <div class="legend-badge badge-yes"><span class="lg-ico">&#10003;</span><span>Yes</span></div>
              <p class="legend-description">The package is active, has recent updates, and is not officially deprecated.</p>
            </div>
            <div class="legend-item">
              <div class="legend-badge badge-unlikely"><span class="lg-ico">&#9888;</span><span>Unlikely</span></div>
              <p class="legend-description">No updates in the last 2 years. The package might be abandoned.</p>
            </div>
            <div class="legend-item">
              <div class="legend-badge badge-no"><span class="lg-ico">&#10007;</span><span>No</span></div>
              <p class="legend-description">Explicitly marked as deprecated on NPM or marked as unmaintained by the community.</p>
            </div>
          </div>
        </div>
      </div>

      <!-- ── REPLACEABILITY LEGEND ──────────────────────────────── -->
      <div class="maintenance-legend-section">
        <div class="legend-container">
          <h3 class="legend-title">Replaceability Score</h3>
          <p class="legend-description" style="text-align:center;">
          Replaceability Score measures how difficult it would be to replace a particular dependency in your project. A higher score indicates that removing or replacing the dependency would require significant effort and could introduce risk.
          If the score is low, it suggests that the dependency is easier to replace—either by implementing the functionality yourself or by using an LLM to generate an alternative solution.
          </p>
          <div class="legend-grid">
            <div class="legend-item">
              <div class="legend-badge badge-yes"><span class="lg-ico">&#9679;</span><span>Easy (0-30)</span></div>
              <p class="legend-description">Mostly small and straightforward libraries with low complexity.</p>
            </div>
            <div class="legend-item">
              <div class="legend-badge badge-unlikely"><span class="lg-ico">&#9888;</span><span>Medium (31-70)</span></div>
              <p class="legend-description">Moderate logic and dependency coupling. Replacement is possible with focused effort.</p>
            </div>
            <div class="legend-item">
              <div class="legend-badge badge-no"><span class="lg-ico">&#10007;</span><span>Hard (71-100)</span></div>
              <p class="legend-description">Native bindings, broad API surface, or complex internals make replacement difficult.</p>
            </div>
          </div>
        </div>
      </div>

      <!-- ── FOOTER ─────────────────────────────────────────────── -->
      <footer class="footer glass-panel">
        <h3 class="footer-title">Author</h3>
        <div class="author-container">
          <img src="https://github.com/hakanolgun.png" alt="Hakan Olgun" class="author-avatar" />
          <div class="author-details">
            <span class="author-name">Hakan Olgun</span>
            <div class="social-links">
              <a href="https://github.com/hakanolgun" target="_blank" rel="noreferrer" title="GitHub">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
                     stroke="currentColor" stroke-width="2"
                     stroke-linecap="round" stroke-linejoin="round">
                  <path d="M9 19c-5 1.5-5-2.5-7-3m14 6v-3.87a3.37 3.37 0 0 0-.94-2.61c3.14-.35 6.44-1.54 6.44-7A5.44 5.44 0 0 0 20 4.77 5.07 5.07 0 0 0 19.91 1S18.73.65 16 2.48a13.38 13.38 0 0 0-7 0C6.27.65 5.09 1 5.09 1A5.07 5.07 0 0 0 5 4.77a5.44 5.44 0 0 0-1.5 3.78c0 5.42 3.3 6.61 6.44 7A3.37 3.37 0 0 0 9 18.13V22"></path>
                </svg>
              </a>
              <a href="https://linkedin.com/in/hknlgn" target="_blank" rel="noreferrer" title="LinkedIn">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
                     stroke="currentColor" stroke-width="2"
                     stroke-linecap="round" stroke-linejoin="round">
                  <path d="M16 8a6 6 0 0 1 6 6v7h-4v-7a2 2 0 0 0-2-2 2 2 0 0 0-2 2v7h-4v-7a6 6 0 0 1 6-6z"></path>
                  <rect x="2" y="9" width="4" height="12"></rect>
                  <circle cx="4" cy="4" r="2"></circle>
                </svg>
              </a>
              <a href="https://x.com/kpt_hkn" target="_blank" rel="noreferrer" title="X (Twitter)">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
                     stroke="currentColor" stroke-width="2"
                     stroke-linecap="round" stroke-linejoin="round">
                  <path d="M4 4l11.733 16h4.267l-11.733-16z"></path>
                  <path d="M4 20l6.768-6.768m2.46-2.46l6.772-6.772"></path>
                </svg>
              </a>
              <a href="https://social.vivaldi.net/@mathrandom" target="_blank" rel="noreferrer" title="Mastodon" class="mastodon-icon">
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
                     stroke="currentColor" stroke-width="2"
                     stroke-linecap="round" stroke-linejoin="round">
                  <path d="M21.58 13.913c-.245 3.074-2.81 5.9-6.31 6.551-2.146.398-4.4.453-6.195.148-1.52-.259-2.85-.92-2.85-.92s-.027-1.393-.049-2.924c1.238.384 2.658.544 4.148.514 2.805-.054 4.978-.507 5.679-2.079.034-.078.066-.159.096-.24.593-1.68-.041-3.666-.041-3.666-4.636 1.139-8.497.106-9.67-2.613-.245-.583-.34-1.282-.365-2.008-.045-1.196-.062-2.52.016-3.834.135-2.261 1.258-4.502 3.619-5.496C11.597 2.456 14.542 2.5 14.542 2.5h.063s2.946-.044 4.256.848c2.361.993 3.484 3.235 3.619 5.496.113 1.956.12 4.17.065 6.069h-.963z" />
                  <path d="M16.51 10.74v.85h-1.57v-.85c0-1.07-.63-1.6-1.89-1.6-1.37 0-2.06.67-2.06 2.01v4.4h-1.54v-4.4c0-1.34-.69-2.01-2.06-2.01-1.26 0-1.89.53-1.89 1.6v.85H3.93v-1.63c0-1.57.85-2.6 2.53-3.1.55-.16 1.16-.24 1.83-.24 1.25 0 2.21.36 2.89 1.07.67-.71 1.63-1.07 2.89-1.07.67 0 1.28.08 1.83.24 1.68.5 2.53 1.53 2.53 3.1v1.63z" />
                </svg>
              </a>
            </div>
          </div>
        </div>
      </footer>

    </div><!-- /.app-container -->
  </div><!-- /#root -->

  <!-- ── EMBEDDED REPORT DATA ───────────────────────────────────── -->
  <script>
    var REPORT_DATA = {{ .ReportJSON }};
  </script>

  <!-- ── CLIENT-SIDE INTERACTIVITY ─────────────────────────────── -->
  <script>
  (function () {
    'use strict';

    // ── State ────────────────────────────────────────────────────
    var sortKey = null;
    var sortDir = 'desc'; // 'asc' | 'desc'

    // ── Sort helpers ─────────────────────────────────────────────
    var maintainedWeight = { yes: 3, unlikely: 2, no: 1 };

    function valueOf(dep, key) {
      switch (key) {
        case 'name':            return (dep.name || '').toLowerCase();
        case 'weeklyDownloads': return dep.weeklyDownloads != null ? dep.weeklyDownloads : -1;
        case 'lastUpdateDate':  return dep.lastUpdateDate ? new Date(dep.lastUpdateDate).getTime() : 0;
        case 'isMaintained':    return maintainedWeight[dep.isMaintained] || 0;
        case 'score':           return dep.score != null ? dep.score : -1;
        case 'newArchitecture': return dep.newArchitecture ? 1 : 0;
        default:                return 0;
      }
    }

    function sortedDeps() {
      var deps = REPORT_DATA.dependencies.slice();
      if (!sortKey) return deps;
      deps.sort(function (a, b) {
        var va = valueOf(a, sortKey);
        var vb = valueOf(b, sortKey);
        if (va < vb) return sortDir === 'asc' ? -1 : 1;
        if (va > vb) return sortDir === 'asc' ?  1 : -1;
        return 0;
      });
      return deps;
    }

    // ── Badge helpers ─────────────────────────────────────────────
    function maintainedBadge(status) {
      if (!status) return '<span class="muted">-</span>';
      var cls = status === 'yes' ? 'success' : status === 'unlikely' ? 'warning' : 'danger';
      var txt = status === 'yes' ? 'Yes' : status === 'unlikely' ? 'Unlikely' : 'No';
      return '<span class="badge ' + cls + '">' + txt + '</span>';
    }

    function replaceabilityCell(dep) {
      if (dep.error) return '<span class="muted">-</span>';
      var score = dep.score != null ? dep.score : 0;
      var label, cls;
      if (score >= 71)      { label = 'Hard';   cls = 'danger'; }
      else if (score >= 31) { label = 'Medium'; cls = 'warning'; }
      else                  { label = 'Easy';   cls = 'success'; }

      return '<div class="replace-stack">' +
        '<span class="badge ' + cls + '">' + label + '</span>' +
        '<span class="sub">' + score + '/100</span>' +
        '</div>';
    }

    function newArchCell(dep) {
      if (dep.newArchitecture === true)  return '<span class="na-yes">Yes</span>';
      if (dep.newArchitecture === false) return '<span class="na-no">No</span>';
      return '<span class="muted">-</span>';
    }

    function escapeHTML(s) {
      return String(s)
        .replace(/&/g,'&amp;').replace(/</g,'&lt;')
        .replace(/>/g,'&gt;').replace(/"/g,'&quot;');
    }

    function formatDownloads(n) {
      if (n == null) return '-';
      return Math.floor(n / 1000) + ' K';
    }

    function formatDate(dep) {
      if (!dep.timeSinceLastUpdate) return '<span class="muted">-</span>';
      var secondary = '';
      if (dep.lastUpdateDate) {
        var d = new Date(dep.lastUpdateDate);
        secondary = (d.getMonth()+1) + '/' + d.getDate() + '/' + d.getFullYear();
      }
      return '<div class="stack"><span>' + escapeHTML(dep.timeSinceLastUpdate) + '</span>' +
             (secondary ? '<span class="sub">' + secondary + '</span>' : '') + '</div>';
    }

    // ── Render table body ─────────────────────────────────────────
    var hasRN = REPORT_DATA.hasReactNative;

    function renderBody() {
      var deps = sortedDeps();
      var rows = deps.map(function (dep) {
        var namecell;
        if (dep.repoUrl) {
          namecell = '<a href="' + escapeHTML(dep.repoUrl) + '" target="_blank" rel="noreferrer" class="repo-link" title="' + escapeHTML(dep.repoUrl) + '">' +
                     escapeHTML(dep.name) + ' <span class="ext">↗</span></a>';
        } else {
          namecell = '<span style="word-break:break-all;">' + escapeHTML(dep.name) + '</span>';
        }
        var errCell = dep.error ? '<span class="pkg-err">' + escapeHTML(dep.error) + '</span>' : '';

        var latestCell = dep.latestVersion
          ? '<span class="badge info">' + escapeHTML(dep.latestVersion) + '</span>'
          : '<span class="muted">-</span>';

        var rnCell = hasRN ? '<td class="cell-center">' + newArchCell(dep) + '</td>' : '';

        return '<tr>' +
          '<td><div class="pkg-name">' + namecell + '</div>' + errCell + '</td>' +
          '<td><span class="badge gray">' + escapeHTML(dep.version.replace(/[\^~]/,'')) + '</span></td>' +
          '<td>' + latestCell + '</td>' +
          '<td>' + formatDownloads(dep.weeklyDownloads) + '</td>' +
          '<td>' + formatDate(dep) + '</td>' +
          '<td>' + maintainedBadge(dep.isMaintained) + '</td>' +
          '<td class="cell-replace">' + replaceabilityCell(dep) + '</td>' +
          rnCell +
          '</tr>';
      });
      document.getElementById('results-body').innerHTML = rows.join('');
    }

    // ── Sort-icon update ──────────────────────────────────────────
    function renderSortIcons() {
      document.querySelectorAll('.sort-icon').forEach(function (el) {
        var col = el.getAttribute('data-for');
        if (col === sortKey) {
          el.textContent = sortDir === 'asc' ? ' ↑' : ' ↓';
        } else {
          el.textContent = '';
        }
      });
    }

    // ── Header click handlers ─────────────────────────────────────
    document.querySelectorAll('th[data-col]').forEach(function (th) {
      th.addEventListener('click', function () {
        var col = th.getAttribute('data-col');
        if (sortKey === col) {
          if (sortDir === 'asc') {
            sortDir = 'desc';
          } else {
            // Third state: back to original order
            sortKey = null;
          }
        } else {
          // First state: ASC for all columns (including Name)
          sortKey = col;
          sortDir = 'asc';
        }
        renderSortIcons();
        renderBody();
      });
    });

    // ── Download JSON ─────────────────────────────────────────────
    document.getElementById('btn-download').addEventListener('click', function () {
      var timestamp = new Date().toISOString().replace(/[:.]/g, '-');
      var payload = {
        generatedAt: new Date().toISOString(),
        ecosystem: 'npm',
        sort: { key: sortKey, direction: sortDir },
        results: sortedDeps()
      };
      var blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
      var url = URL.createObjectURL(blob);
      var a = document.createElement('a');
      a.href = url;
      a.download = 'dep-analysis-' + timestamp + '.json';
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    });

    // Initial render syncs with the server-rendered HTML (no-op on first load,
    // but ensures body is always driven by JS after page paint).
    renderSortIcons();

  })();
  </script>
</body>
</html>
`

const embeddedReportCSS = `
:root {
  --bg-color: #0f172a;
  --text-color: #f8fafc;
  --text-muted: #94a3b8;
  --primary: #3b82f6;
  --success: #10b981;
  --danger: #ef4444;
  --warning: #f59e0b;
  --glass-bg: rgba(30, 41, 59, 0.7);
  --glass-border: rgba(255, 255, 255, 0.1);
  --glass-shadow: 0 8px 32px 0 rgba(0, 0, 0, 0.37);
  font-family: "Outfit", system-ui, -apple-system, sans-serif;
  line-height: 1.5;
  color-scheme: dark;
  background-color: var(--bg-color);
  color: var(--text-color);
  background-image:
    radial-gradient(circle at 15% 50%, rgba(59, 130, 246, 0.15), transparent 25%),
    radial-gradient(circle at 85% 30%, rgba(16, 185, 129, 0.1), transparent 25%);
  background-attachment: fixed;
  min-height: 100vh;
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  margin: 0;
  display: flex;
  place-items: flex-start center;
  min-height: 100vh;
  padding: 2rem;
}
#root {
  width: 100%;
  max-width: 1200px;
  margin: 0 auto;
}
.app-container {
  display: flex;
  flex-direction: column;
  gap: 2rem;
  width: 100%;
}
.glass-panel {
  background: var(--glass-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--glass-border);
  border-radius: 16px;
  box-shadow: var(--glass-shadow);
  padding: 2rem;
}
/* ── Header panel ── */
.header-panel {
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
}
.repo-subtitle {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  margin-top: -0.5rem;
}
.subtitle-link {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: inherit;
  text-decoration: none;
  transition: color 0.2s ease;
}
.subtitle-link:hover { color: #fff; }
/* ── Typography ── */
.title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 0.5rem;
  background: linear-gradient(135deg, #60a5fa, #34d399);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
}
.subtitle {
  color: var(--text-muted);
  margin-bottom: 2rem;
  font-size: 1rem;
}
/* ── Results header ── */
.header-actions {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
  gap: 1rem;
  flex-wrap: wrap;
}
/* ── Button ── */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.6rem 1.2rem;
  border: 1px solid var(--glass-border);
  border-radius: 8px;
  color: var(--text-color);
  font-family: inherit;
  font-size: 0.9rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.2s ease, border-color 0.2s ease;
  white-space: nowrap;
}
.btn:hover { background: rgba(255,255,255,0.15); border-color: rgba(255,255,255,0.2); }
/* ── Table ── */
.table-container {
  width: 100%;
  overflow-x: auto;
  border-radius: 8px;
  border: 1px solid var(--glass-border);
  background: rgba(15, 23, 42, 0.4);
}
table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}
th {
  background: rgba(255, 255, 255, 0.05);
  padding: 1rem;
  font-weight: 600;
  font-size: 0.85rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  border-bottom: 1px solid var(--glass-border);
  white-space: nowrap;
}
th.sortable { cursor: pointer; user-select: none; }
th.sortable:hover { color: var(--text-color); }
td {
  padding: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  vertical-align: middle;
}
tr:last-child td { border-bottom: none; }
tr { transition: background-color 0.2s ease; }
tr:hover { background: rgba(255, 255, 255, 0.03); }
.sort-icon { display: inline-block; width: 1.2em; }
/* ── Cells ── */
.muted { color: var(--text-muted); }
.sub { font-size: 0.75rem; color: var(--text-muted); }
.stack { display: flex; flex-direction: column; gap: 2px; }
.pkg-err {
  color: var(--danger);
  font-size: 0.8rem;
  margin-top: 4px;
  display: block;
}
.cell-center { text-align: center; }
.cell-replace { text-align: center; }
.replace-stack {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
}
.badge.conf { font-size: 0.65rem; }
.na-yes { color: var(--success); font-weight: 600; }
.na-no  { color: var(--danger);  font-weight: 600; }
/* ── Badges ── */
.badge {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.1);
}
.badge.success { background: rgba(16,  185, 129, 0.2); color: #34d399; }
.badge.warning { background: rgba(245, 158,  11, 0.2); color: #fbbf24; }
.badge.danger  { background: rgba(239,  68,  68, 0.2); color: #f87171; }
.badge.info    { background: rgba(59,  130, 246, 0.2); color: #60a5fa; }
.badge.gray    { background: rgba(148, 163, 184, 0.2); color: #94a3b8; }
/* ── Package name ── */
.pkg-name {
  font-weight: 600;
  color: #e2e8f0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}
.repo-link { color: inherit; text-decoration: none; word-break: break-all; }
.repo-link:hover { color: #fff; }
.ext { opacity: 0.6; font-size: 0.75rem; }
/* ── Legend sections ── */
.maintenance-legend-section {
  padding: 1.5rem 2rem;
  background: var(--glass-bg);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  border: 1px solid var(--glass-border);
  border-radius: 16px;
  box-shadow: var(--glass-shadow);
}
.legend-container { display: flex; flex-direction: column; gap: 1.5rem; }
.legend-title {
  font-size: 1rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--text-muted);
  text-align: center;
  border-bottom: 1px solid var(--glass-border);
  padding-bottom: 0.75rem;
  margin: 0 auto;
  width: 100%;
  max-width: 400px;
}
.legend-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 2rem;
}
.legend-item {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding: 1rem;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid transparent;
}
.legend-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.4rem 1rem;
  border-radius: 9999px;
  font-size: 0.85rem;
  font-weight: 700;
  width: fit-content;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.badge-yes      { background: rgba(16, 185, 129, 0.15); color: #34d399; border: 1px solid rgba(16, 185, 129, 0.2); }
.badge-unlikely { background: rgba(245,158,  11, 0.15); color: #fbbf24; border: 1px solid rgba(245,158,  11, 0.2); }
.badge-no       { background: rgba(239, 68,  68, 0.15); color: #f87171; border: 1px solid rgba(239, 68,  68, 0.2); }
.legend-description { font-size: 0.9rem; color: var(--text-muted); line-height: 1.6; }
.lg-ico { font-size: 0.9rem; }
/* ── Footer ── */
.footer { margin-top: 0; display:flex; flex-direction: column; align-items: center; justify-content: center; }
.footer-title {
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  color: var(--text-muted);
  margin-bottom: 1rem;
}
.author-container {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 1rem;
}
.author-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 2px solid var(--glass-border);
  object-fit: cover;
}
.author-details {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 0.8rem;
}
.author-name { font-weight: 600; color: var(--text-color); font-size: 1rem; }
.social-links {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}
.social-links a {
  color: var(--text-muted);
  text-decoration: none;
  transition: color 0.2s ease;
  display: flex;
  align-items: center;
}
.social-links a:hover { color: var(--text-color); }
/* ── Responsive ── */
@media (max-width: 768px) {
  body { padding: 1rem; }
  .legend-grid { grid-template-columns: 1fr; gap: 1rem; }
  .header-actions { flex-direction: column; align-items: flex-start; }
}
`
