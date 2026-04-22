package report

import (
	"embed"
	"html/template"
)

//go:embed assets/report.css
var reportCSS string

//go:embed assets/report-npm.js
var reportNpmJS string

//go:embed templates/*.gohtml templates/partials/*.gohtml
var tmplFS embed.FS

var reportTemplates *template.Template

func init() {
	var err error
	reportTemplates, err = template.ParseFS(tmplFS,
		"templates/npm.gohtml",
		"templates/partials/head.gohtml",
		"templates/partials/header.gohtml",
		"templates/partials/footer.gohtml",
		"templates/partials/shell_close.gohtml",
		"templates/partials/tail.gohtml",
		"templates/partials/maintenance_npm.gohtml",
		"templates/partials/replaceability_npm.gohtml",
	)
	if err != nil {
		panic("report templates: " + err.Error())
	}
}
