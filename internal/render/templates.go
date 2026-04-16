package render

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/jomakori/TF_summarize/internal"
)

// Template constants for rendering.
const (
	summaryTemplate = `## Terraform {{ if eq .Phase "apply" }}Apply{{ else if .IsDestroy }}Destroy{{ else }}Plan{{ end }}

{{ if eq .Phase "apply" -}}
{{ if .ApplySucceeded }}✅ Changes applied successfully{{ else }}❌ Apply failed{{ end }}
{{- else -}}
📋 Changes found
{{- end }} for {{ .Workspace }}

{{ if .DriftDetected -}}
> [!WARNING]
> **Drift detected!** Objects have changed outside of Terraform.

{{ end -}}
{{ if and (gt .ToDestroy 0) (eq .Phase "plan") -}}
> [!CAUTION]
> **Terraform will delete resources!**
> This plan contains resource delete operations. Please check the plan result very carefully.

{{ end -}}
{{ if eq .Phase "apply" -}}
**Result:** 
{{- if gt .ToAdd 0 }} **{{ .ToAdd }}** added{{- end }}
{{- if gt .ToChange 0 }}, **{{ .ToChange }}** changed{{- end }}
{{- if gt .ToDestroy 0 }}, **{{ .ToDestroy }}** destroyed{{- end }}
{{- if gt .Failed 0 }}, **{{ .Failed }}** failed{{- end }}
{{ else -}}
**Plan:** 
{{- if gt .ToAdd 0 }} **{{ .ToAdd }}** to add{{- end }}
{{- if gt .ToChange 0 }}, **{{ .ToChange }}** to change{{- end }}
{{- if gt .ToDestroy 0 }}, **{{ .ToDestroy }}** to destroy{{- end }}
{{- if gt .ToImport 0 }}, **{{ .ToImport }}** to import{{- end }}
{{ end }}`

	badgesTemplate = `![Terraform](https://img.shields.io/badge/Terraform-{{ if eq .Phase "apply" }}Apply{{ else if .IsDestroy }}Destroy{{ else }}Plan{{ end }}-{{ if eq .Phase "apply" }}{{ if .ApplySucceeded }}28a745{{ else }}dc3545{{ end }}{{ else if .IsDestroy }}dc3545{{ else }}007bff{{ end }}){{ if gt .ToAdd 0 }} ![](https://img.shields.io/badge/-Create%20%28{{ .ToAdd }}%29-28a745){{ end }}{{ if gt .ToChange 0 }} ![](https://img.shields.io/badge/-Modify%20%28{{ .ToChange }}%29-FFC107){{ end }}{{ if gt .ToDestroy 0 }} ![](https://img.shields.io/badge/-Remove%20%28{{ .ToDestroy }}%29-dc3545){{ end }}{{ if gt .ToImport 0 }} ![](https://img.shields.io/badge/-Import%20%28{{ .ToImport }}%29-6f42c1){{ end }}{{ if .DriftDetected }} ![](https://img.shields.io/badge/-Drift%20Detected-fd7e14){{ end }}`

	resourcesTemplate = `{{ range .Creates -}}
<details>
<summary><b>Create</b> <b>({{ len .Creates }})</b></summary>

` + "```diff\n" + `{{ range .Creates -}}
+ {{ .Address }}
{{ end -}}
` + "```\n" + `
</details>

{{ end -}}`

	fullTemplate = `## Terraform {{ if eq .Phase "apply" }}Apply{{ else if .IsDestroy }}Destroy{{ else }}Plan{{ end }}

{{ if eq .Phase "apply" -}}
{{ if .ApplySucceeded }}✅ Changes applied successfully{{ else }}❌ Apply failed{{ end }}
{{- else -}}
📋 Changes found
{{- end }} for {{ .Workspace }}

{{ if .DriftDetected -}}
> [!WARNING]
> **Drift detected!** Objects have changed outside of Terraform.

{{ end -}}
{{ if and (gt .ToDestroy 0) (eq .Phase "plan") -}}
> [!CAUTION]
> **Terraform will delete resources!**
> This plan contains resource delete operations. Please check the plan result very carefully.

{{ end -}}
{{ if eq .Phase "apply" -}}
**Result:** 
{{- if gt .ToAdd 0 }} **{{ .ToAdd }}** added{{- end }}
{{- if gt .ToChange 0 }}, **{{ .ToChange }}** changed{{- end }}
{{- if gt .ToDestroy 0 }}, **{{ .ToDestroy }}** destroyed{{- end }}
{{- if gt .Failed 0 }}, **{{ .Failed }}** failed{{- end }}
{{ else -}}
**Plan:** 
{{- if gt .ToAdd 0 }} **{{ .ToAdd }}** to add{{- end }}
{{- if gt .ToChange 0 }}, **{{ .ToChange }}** to change{{- end }}
{{- if gt .ToDestroy 0 }}, **{{ .ToDestroy }}** to destroy{{- end }}
{{- if gt .ToImport 0 }}, **{{ .ToImport }}** to import{{- end }}
{{ end }}

{{ if .RawOutput -}}
<details>
<summary><b>Terraform {{ if eq .Phase "apply" }}Apply{{ else }}Plan{{ end }} Output</b></summary>

` + "```\n" + `{{ .RawOutput }}
` + "```\n" + `
</details>
{{ end -}}`
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

func (tr *TemplateRenderer) loadDefaultTemplates() {
	tr.templates["summary"] = template.Must(template.New("summary").Parse(summaryTemplate))
	tr.templates["badges"] = template.Must(template.New("badges").Parse(badgesTemplate))
	tr.templates["resources"] = template.Must(template.New("resources").Parse(resourcesTemplate))
	tr.templates["full"] = template.Must(template.New("full").Parse(fullTemplate))
}

// RenderSummary renders the summary section using template.
func (tr *TemplateRenderer) RenderSummary(s *internal.Summary) (string, error) {
	data := buildSummaryData(s)
	return tr.executeTemplate("summary", data)
}

// RenderResources renders the resource sections using template.
func (tr *TemplateRenderer) RenderResources(s *internal.Summary) (string, error) {
	data := buildResourcesData(s)
	return tr.executeTemplate("resources", data)
}

// RenderFull renders the complete output using template.
func (tr *TemplateRenderer) RenderFull(s *internal.Summary, rawOutput string) (string, error) {
	data := buildFullData(s, rawOutput)
	return tr.executeTemplate("full", data)
}

// RegisterTemplate registers a custom template.
func (tr *TemplateRenderer) RegisterTemplate(name string, tmpl *template.Template) {
	tr.templates[name] = tmpl
}

func (tr *TemplateRenderer) executeTemplate(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tr.templates[name].Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering %s template: %w", name, err)
	}
	return buf.String(), nil
}

func buildSummaryData(s *internal.Summary) map[string]interface{} {
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

func buildResourcesData(s *internal.Summary) map[string]interface{} {
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

func buildFullData(s *internal.Summary, rawOutput string) map[string]interface{} {
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
