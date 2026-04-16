package internal

import (
	"bytes"
	"fmt"
	"text/template"
)

// TemplateRenderer renders output using Go templates.
type TemplateRenderer struct {
	templates map[string]*template.Template
}

// NewTemplateRenderer creates a new template renderer with default templates.
func NewTemplateRenderer() *TemplateRenderer {
	tr := &TemplateRenderer{
		templates: make(map[string]*template.Template),
	}
	tr.loadDefaultTemplates()
	return tr
}

// loadDefaultTemplates loads built-in templates.
func (tr *TemplateRenderer) loadDefaultTemplates() {
	tr.templates["summary"] = template.Must(template.New("summary").Parse(summaryTemplate))
	tr.templates["badges"] = template.Must(template.New("badges").Parse(badgesTemplate))
	tr.templates["resources"] = template.Must(template.New("resources").Parse(resourcesTemplate))
	tr.templates["full"] = template.Must(template.New("full").Parse(fullTemplate))
}

// RenderSummary renders the summary section using template.
func (tr *TemplateRenderer) RenderSummary(s *Summary) (string, error) {
	data := buildSummaryData(s)
	return tr.executeTemplate("summary", data)
}

// RenderResources renders the resource sections using template.
func (tr *TemplateRenderer) RenderResources(s *Summary) (string, error) {
	data := buildResourcesData(s)
	return tr.executeTemplate("resources", data)
}

// RenderFull renders the complete output using template.
func (tr *TemplateRenderer) RenderFull(s *Summary, rawOutput string) (string, error) {
	data := buildFullData(s, rawOutput)
	return tr.executeTemplate("full", data)
}

// RegisterTemplate registers a custom template.
func (tr *TemplateRenderer) RegisterTemplate(name string, tmpl *template.Template) {
	tr.templates[name] = tmpl
}

// executeTemplate executes a template and returns the result.
func (tr *TemplateRenderer) executeTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tr.templates[name].Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering %s template: %w", name, err)
	}
	return buf.String(), nil
}

// buildSummaryData builds template data for summary rendering.
func buildSummaryData(s *Summary) map[string]interface{} {
	return map[string]interface{}{
		"Phase":         s.Phase,
		"Workspace":     s.Workspace,
		"ToAdd":         s.ToAdd,
		"ToChange":      s.ToChange,
		"ToDestroy":     s.ToDestroy,
		"ToImport":      s.ToImport,
		"IsDestroy":     s.IsDestroyPlan,
		"DriftDetected": s.DriftDetected,
		"Warnings":      s.Warnings,
		"Errors":        s.Errors,
	}
}

// buildResourcesData builds template data for resources rendering.
func buildResourcesData(s *Summary) map[string]interface{} {
	return map[string]interface{}{
		"Phase":    s.Phase,
		"Creates":  s.Creates,
		"Updates":  s.Updates,
		"Destroys": s.Destroys,
		"Replaces": s.Replaces,
		"Reads":    s.Reads,
		"Imports":  s.Imports,
		"Failures": s.Failures,
	}
}

// buildFullData builds template data for full rendering.
func buildFullData(s *Summary, rawOutput string) map[string]interface{} {
	return map[string]interface{}{
		"Phase":          s.Phase,
		"Workspace":      s.Workspace,
		"ToAdd":          s.ToAdd,
		"ToChange":       s.ToChange,
		"ToDestroy":      s.ToDestroy,
		"ToImport":       s.ToImport,
		"IsDestroy":      s.IsDestroyPlan,
		"DriftDetected":  s.DriftDetected,
		"Warnings":       s.Warnings,
		"Errors":         s.Errors,
		"Creates":        s.Creates,
		"Updates":        s.Updates,
		"Destroys":       s.Destroys,
		"Replaces":       s.Replaces,
		"Reads":          s.Reads,
		"Imports":        s.Imports,
		"Failures":       s.Failures,
		"RawOutput":      rawOutput,
		"ApplySucceeded": s.ApplySucceeded,
		"Applied":        s.Applied,
		"Failed":         s.Failed,
	}
}
