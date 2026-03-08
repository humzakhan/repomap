package analyzer

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

//go:embed prompts
var promptFS embed.FS

// PromptTemplates holds parsed prompt templates.
type PromptTemplates struct {
	SummarizeModule      *template.Template
	SynthesizeArch       *template.Template
	IngestDocs           *template.Template
}

// LoadPromptTemplates loads and parses all prompt templates from embedded files.
func LoadPromptTemplates() (*PromptTemplates, error) {
	summarize, err := loadTemplate("prompts/summarize_module.txt")
	if err != nil {
		return nil, fmt.Errorf("loading summarize_module template: %w", err)
	}

	synthesize, err := loadTemplate("prompts/synthesize_architecture.txt")
	if err != nil {
		return nil, fmt.Errorf("loading synthesize_architecture template: %w", err)
	}

	ingest, err := loadTemplate("prompts/ingest_docs.txt")
	if err != nil {
		return nil, fmt.Errorf("loading ingest_docs template: %w", err)
	}

	return &PromptTemplates{
		SummarizeModule: summarize,
		SynthesizeArch:  synthesize,
		IngestDocs:      ingest,
	}, nil
}

func loadTemplate(path string) (*template.Template, error) {
	data, err := promptFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	tmpl, err := template.New(path).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return tmpl, nil
}

// RenderSummarize renders the summarize_module prompt template.
func (pt *PromptTemplates) RenderSummarize(data SummarizeData) (string, error) {
	return renderTemplate(pt.SummarizeModule, data)
}

// RenderSynthesize renders the synthesize_architecture prompt template.
func (pt *PromptTemplates) RenderSynthesize(data SynthesizeData) (string, error) {
	return renderTemplate(pt.SynthesizeArch, data)
}

// RenderIngestDocs renders the ingest_docs prompt template.
func (pt *PromptTemplates) RenderIngestDocs(data IngestDocsData) (string, error) {
	return renderTemplate(pt.IngestDocs, data)
}

func renderTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering template %s: %w", tmpl.Name(), err)
	}
	return buf.String(), nil
}

// SummarizeData is the data passed to the summarize_module template.
type SummarizeData struct {
	FilePath string
	Language string
	Exports  []string
	Imports  []ImportRef
	Code     string
}

// ImportRef is a simplified import reference for template rendering.
type ImportRef struct {
	Source string
}

// SynthesizeData is the data passed to the synthesize_architecture template.
type SynthesizeData struct {
	Summaries  string
	Edges      string
	RepoName   string
	TotalFiles int
	Languages  string
}

// IngestDocsData is the data passed to the ingest_docs template.
type IngestDocsData struct {
	DocContent    string
	CodeSummaries string
}
