package internal

import (
	"fmt"
	"strings"
)

// Render produces a markdown summary for the given Summary.
func Render(s *Summary) string {
	var b strings.Builder

	writeHeader(&b, s)
	writeBadges(&b, s)
	writeWarnings(&b, s)
	writeErrors(&b, s)
	writeSummaryLine(&b, s)
	writeResourceSections(&b, s)
	writeRawOutput(&b, s)

	return b.String()
}

func writeHeader(b *strings.Builder, s *Summary) {
	switch s.Phase {
	case PhaseApply:
		if s.ApplySucceeded && len(s.Errors) == 0 {
			b.WriteString(fmt.Sprintf("## ✅ Changes applied successfully for `%s`\n\n", s.Workspace))
		} else if len(s.Errors) > 0 {
			b.WriteString(fmt.Sprintf("## ❌ Apply failed for `%s`\n\n", s.Workspace))
		} else {
			b.WriteString(fmt.Sprintf("## ⏳ Apply results for `%s`\n\n", s.Workspace))
		}
	default:
		if s.ToAdd == 0 && s.ToChange == 0 && s.ToDestroy == 0 && s.ToImport == 0 {
			b.WriteString(fmt.Sprintf("## ✅ No changes found for `%s`\n\n", s.Workspace))
		} else {
			b.WriteString(fmt.Sprintf("## 📋 Changes found for `%s`\n\n", s.Workspace))
		}
	}
}

func writeBadges(b *strings.Builder, s *Summary) {
	var badges []string

	phaseBadge := "PLAN"
	if s.Phase == PhaseApply {
		phaseBadge = "APPLY"
	}
	badges = append(badges, fmt.Sprintf("![phase](https://img.shields.io/badge/%s-grey)", phaseBadge))

	// Action badge — pick the most significant action
	switch {
	case len(s.Replaces) > 0 || (s.ToDestroy > 0 && s.ToAdd > 0):
		badges = append(badges, "![action](https://img.shields.io/badge/REPLACE-e74c3c)")
	case s.ToDestroy > 0:
		badges = append(badges, "![action](https://img.shields.io/badge/DESTROY-e74c3c)")
	case s.ToChange > 0 && s.ToAdd > 0:
		badges = append(badges, "![action](https://img.shields.io/badge/UPDATE%20%2B%20CREATE-2ecc71)")
	case s.ToChange > 0:
		badges = append(badges, "![action](https://img.shields.io/badge/UPDATE-f39c12)")
	case s.ToAdd > 0:
		badges = append(badges, "![action](https://img.shields.io/badge/CREATE-2ecc71)")
	case s.ToImport > 0:
		badges = append(badges, "![action](https://img.shields.io/badge/IMPORT-3498db)")
	default:
		badges = append(badges, "![action](https://img.shields.io/badge/NO%20CHANGES-grey)")
	}

	if s.DriftDetected {
		badges = append(badges, "![drift](https://img.shields.io/badge/DRIFT%20DETECTED-e67e22)")
	}

	if s.Phase == PhaseApply && len(s.Failures) > 0 {
		badges = append(badges, fmt.Sprintf("![failures](https://img.shields.io/badge/%d%%20FAILED-e74c3c)", len(s.Failures)))
	}

	b.WriteString(strings.Join(badges, " "))
	b.WriteString("\n\n")
}

func writeWarnings(b *strings.Builder, s *Summary) {
	if s.ToDestroy > 0 || len(s.Replaces) > 0 {
		if s.Phase == PhasePlan {
			b.WriteString("> [!CAUTION]\n")
			b.WriteString("> **Terraform will delete resources!**\n")
			b.WriteString("> This plan contains resource delete operations. Please check the plan result very carefully.\n\n")
		}
	}

	if s.DriftDetected {
		b.WriteString("> [!WARNING]\n")
		b.WriteString("> **Drift detected!** Objects have changed outside of Terraform.\n\n")
	}

	for _, w := range s.Warnings {
		b.WriteString(fmt.Sprintf("> [!WARNING]\n> %s\n\n", w))
	}
}

func writeErrors(b *strings.Builder, s *Summary) {
	if s.Phase == PhaseApply && len(s.Failures) > 0 {
		// Detailed per-resource errors
		return // rendered in resource sections instead
	}
	// Fallback: show raw errors if we couldn't associate them with resources
	for _, e := range s.Errors {
		b.WriteString(fmt.Sprintf("> [!CAUTION]\n> **Error:** %s\n\n", e))
	}
}

func writeSummaryLine(b *strings.Builder, s *Summary) {
	if s.Phase == PhaseApply {
		writeApplySummaryLine(b, s)
		return
	}

	parts := []string{}
	if s.ToAdd > 0 {
		parts = append(parts, fmt.Sprintf("**%d** to add", s.ToAdd))
	}
	if s.ToChange > 0 {
		parts = append(parts, fmt.Sprintf("**%d** to change", s.ToChange))
	}
	if s.ToDestroy > 0 {
		parts = append(parts, fmt.Sprintf("**%d** to destroy", s.ToDestroy))
	}
	if s.ToImport > 0 {
		parts = append(parts, fmt.Sprintf("**%d** to import", s.ToImport))
	}

	if len(parts) == 0 {
		b.WriteString("Infrastructure is up-to-date. No changes needed.\n\n")
		return
	}

	b.WriteString(fmt.Sprintf("**Plan:** %s\n\n", strings.Join(parts, ", ")))
}

func writeApplySummaryLine(b *strings.Builder, s *Summary) {
	parts := []string{}
	if s.ToAdd > 0 {
		parts = append(parts, fmt.Sprintf("**%d** added", s.ToAdd))
	}
	if s.ToChange > 0 {
		parts = append(parts, fmt.Sprintf("**%d** changed", s.ToChange))
	}
	if s.ToDestroy > 0 {
		parts = append(parts, fmt.Sprintf("**%d** destroyed", s.ToDestroy))
	}
	if len(s.Failures) > 0 {
		parts = append(parts, fmt.Sprintf("**%d** failed", len(s.Failures)))
	}

	if len(parts) == 0 && s.ApplySucceeded {
		b.WriteString("No resource changes were needed.\n\n")
		return
	}

	if len(parts) > 0 {
		b.WriteString(fmt.Sprintf("**Result:** %s\n\n", strings.Join(parts, ", ")))
	}
}

func writeResourceSections(b *strings.Builder, s *Summary) {
	if s.Phase == PhaseApply {
		writeApplyResourceSections(b, s)
		return
	}

	writeResourceGroup(b, "Create", "+", s.Creates)
	writeResourceGroup(b, "Update", "~", s.Updates)
	writeResourceGroup(b, "Destroy", "-", s.Destroys)
	writeResourceGroup(b, "Replace", "-/+", s.Replaces)
	writeResourceGroup(b, "Import", "←", s.Imports)
	writeResourceGroup(b, "Read", "<=", s.Reads)
}

func writeApplyResourceSections(b *strings.Builder, s *Summary) {
	// Succeeded resources
	writeApplyResourceGroup(b, "Created", "+", "✅", s.Creates)
	writeApplyResourceGroup(b, "Updated", "~", "✅", s.Updates)
	writeApplyResourceGroup(b, "Destroyed", "-", "✅", s.Destroys)

	// Failed resources with error details
	if len(s.Failures) > 0 {
		b.WriteString("<details open>\n<summary><b>❌ Failed</b> (")
		b.WriteString(fmt.Sprintf("%d", len(s.Failures)))
		b.WriteString(")</summary>\n\n")

		for _, r := range s.Failures {
			b.WriteString(fmt.Sprintf("**`%s`**\n", r.Address))
			if r.Error != "" {
				b.WriteString(fmt.Sprintf("> %s\n", r.Error))
			}
			b.WriteString("\n")
		}

		b.WriteString("</details>\n\n")
	}
}

func writeResourceGroup(b *strings.Builder, title, prefix string, resources []ResourceChange) {
	if len(resources) == 0 {
		return
	}

	b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s</b> (%d)</summary>\n\n```diff\n", title, len(resources)))

	for _, r := range resources {
		line := fmt.Sprintf("%s %s", prefix, r.Address)
		if r.Error != "" {
			line += fmt.Sprintf("  # ERROR: %s", r.Error)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("```\n\n</details>\n\n")
}

func writeApplyResourceGroup(b *strings.Builder, title, prefix, icon string, resources []ResourceChange) {
	if len(resources) == 0 {
		return
	}

	b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s %s</b> (%d)</summary>\n\n```diff\n", icon, title, len(resources)))

	for _, r := range resources {
		b.WriteString(fmt.Sprintf("%s %s\n", prefix, r.Address))
	}

	b.WriteString("```\n\n</details>\n\n")
}

func writeRawOutput(b *strings.Builder, s *Summary) {
	raw := strings.TrimSpace(s.RawOutput)
	if raw == "" {
		return
	}

	title := "Terraform Plan Output"
	if s.Phase == PhaseApply {
		title = "Terraform Apply Output"
	}

	b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s</b></summary>\n\n```\n", title))
	b.WriteString(raw)
	b.WriteString("\n```\n\n</details>\n")
}
