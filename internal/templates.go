package internal

// summaryTemplate renders the summary section with header and counts.
const summaryTemplate = `## Terraform {{ if eq .Phase "apply" }}Apply{{ else if .IsDestroy }}Destroy{{ else }}Plan{{ end }}

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

// badgesTemplate renders color-coded shields.io badges.
const badgesTemplate = `![Terraform](https://img.shields.io/badge/Terraform-{{ if eq .Phase "apply" }}Apply{{ else if .IsDestroy }}Destroy{{ else }}Plan{{ end }}-{{ if eq .Phase "apply" }}{{ if .ApplySucceeded }}28a745{{ else }}dc3545{{ end }}{{ else if .IsDestroy }}dc3545{{ else }}007bff{{ end }}){{ if gt .ToAdd 0 }} ![](https://img.shields.io/badge/-Create%20%28{{ .ToAdd }}%29-28a745){{ end }}{{ if gt .ToChange 0 }} ![](https://img.shields.io/badge/-Modify%20%28{{ .ToChange }}%29-FFC107){{ end }}{{ if gt .ToDestroy 0 }} ![](https://img.shields.io/badge/-Remove%20%28{{ .ToDestroy }}%29-dc3545){{ end }}{{ if gt .ToImport 0 }} ![](https://img.shields.io/badge/-Import%20%28{{ .ToImport }}%29-6f42c1){{ end }}{{ if .DriftDetected }} ![](https://img.shields.io/badge/-Drift%20Detected-fd7e14){{ end }}`

// resourcesTemplate renders resource lists grouped by action type.
const resourcesTemplate = `{{ range .Creates -}}
<details>
<summary><b>Create</b> <b>({{ len .Creates }})</b></summary>

` + "```diff\n" + `{{ range .Creates -}}
+ {{ .Address }}
{{ end -}}
` + "```\n" + `
</details>

{{ end -}}`

// fullTemplate renders the complete markdown output with all sections.
const fullTemplate = `## Terraform {{ if eq .Phase "apply" }}Apply{{ else if .IsDestroy }}Destroy{{ else }}Plan{{ end }}

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
