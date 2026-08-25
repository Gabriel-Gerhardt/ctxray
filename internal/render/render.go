// Package render turns an analyze.Report into a single self-contained
// HTML file: inline CSS, inline SVG, zero external requests. It opens the
// same whether you double-click it or serve it from a CDN.
package render

import (
	"embed"
	"html/template"
	"io"

	"github.com/Gabriel-Gerhardt/ctxray/internal/analyze"
)

//go:embed template.html.tmpl
var templateFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "template.html.tmpl"))

// Render writes the report as HTML to w.
func Render(report analyze.Report, w io.Writer) error {
	return tmpl.Execute(w, buildViewModel(report))
}
