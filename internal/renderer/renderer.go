package renderer

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"

	"github.com/repomap/repomap/internal/analyzer"
	"github.com/repomap/repomap/internal/scanner"
)

//go:embed templates/report.html
var reportTemplate string

// ReportBundle holds the compiled frontend assets.
// These are populated at build time via go:embed in the embedding file.
var (
	BundleJS  string
	StylesCSS string
)

// ReportData is the complete data structure injected into the HTML report.
type ReportData struct {
	Summaries    []analyzer.ModuleSummary        `json:"summaries"`
	Architecture *analyzer.ArchitectureSynthesis  `json:"architecture,omitempty"`
	DocWarnings  []analyzer.DocWarning            `json:"doc_warnings,omitempty"`
	Stats        analyzer.PipelineStats           `json:"stats"`
	Graph        *scanner.RepoGraph               `json:"graph"`
	Metadata     *scanner.RepoMetadata            `json:"metadata"`
	GitMeta      *scanner.GitMetadata             `json:"git_meta,omitempty"`
}

type templateData struct {
	Title  string
	Styles template.CSS
	Data   template.JS
	Bundle template.JS
}

// Render produces a self-contained HTML report file.
func Render(data *ReportData, outputPath string) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshaling report data: %w", err)
	}

	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("parsing report template: %w", err)
	}

	td := templateData{
		Title:  data.Metadata.Name,
		Styles: template.CSS(StylesCSS),
		Data:   template.JS(jsonData),
		Bundle: template.JS(BundleJS),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, td); err != nil {
		return fmt.Errorf("rendering report template: %w", err)
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating output directory %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing report to %s: %w", outputPath, err)
	}

	return nil
}
