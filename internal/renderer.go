package internal

import (
	"fmt"
	"net/url"
	"strings"
)

// Color constants for shields.io badges
const (
	colorGreen       = "28a745" // New changes (Create) & successful Apply
	colorRed         = "dc3545" // Deleted changes (Remove/Replace), Destroy Plan, Failed Apply
	colorYellow      = "FFC107" // Modifications (Modify) - vibrant yellow
	colorNoChanges   = "0366d6" // No changes (blue)
	colorImport      = "6f42c1" // Import (purple)
	colorOrange      = "fd7e14" // Warnings/Drift
	colorPlan        = "007bff" // Phase badge (Plan) - blue
)

// createShieldsIOBadge creates a shields.io badge URL
func createShieldsIOBadge(label, message, color string) string {
	// URL encode message for shields.io badge path (spaces become %20, not +)
	encodedMessage := strings.ReplaceAll(url.QueryEscape(message), "+", "%20")
	return fmt.Sprintf("![%s](https://img.shields.io/badge/%s-%s-%s)", label, label, encodedMessage, color)
}

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

	// Phase badge with color dependent on plan type and success
	phaseBadge := "Plan"
	phaseColor := colorPlan
	
	if s.Phase == PhaseApply {
		phaseBadge = "Apply"
		// Apply phase: green for success, red for failures
		if len(s.Failures) > 0 || (s.ApplyError != "" && !s.ApplySucceeded) {
			phaseColor = colorRed // red for failed apply
		} else {
			phaseColor = colorGreen // green for successful apply
		}
	} else {
		// Plan phase: blue for regular plan, red for destroy plan
		if s.IsDestroyPlan {
			phaseBadge = "Destroy"
			phaseColor = colorRed // red for destroy plan
		}
	}
	
	badges = append(badges, createShieldsIOBadge("Terraform", phaseBadge, phaseColor))

	// Individual action badges with counts (PascalCase with requested terminology)
	if s.ToAdd > 0 {
		msg := fmt.Sprintf("Create (%d)", s.ToAdd)
		badges = append(badges, createShieldsIOBadge("", msg, colorGreen))
	}

	if s.ToChange > 0 {
		msg := fmt.Sprintf("Modify (%d)", s.ToChange)
		badges = append(badges, createShieldsIOBadge("", msg, colorYellow))
	}

	if s.ToDestroy > 0 {
		msg := fmt.Sprintf("Remove (%d)", s.ToDestroy)
		badges = append(badges, createShieldsIOBadge("", msg, colorRed))
	}

	if len(s.Replaces) > 0 {
		msg := fmt.Sprintf("Replace (%d)", len(s.Replaces))
		badges = append(badges, createShieldsIOBadge("", msg, colorRed))
	}

	if s.ToImport > 0 {
		msg := fmt.Sprintf("Import (%d)", s.ToImport)
		badges = append(badges, createShieldsIOBadge("", msg, colorImport))
	}

	// Show NO CHANGES badge only if there are truly no changes (blue color)
	if s.ToAdd == 0 && s.ToChange == 0 && s.ToDestroy == 0 && s.ToImport == 0 && len(s.Replaces) == 0 {
		badges = append(badges, createShieldsIOBadge("", "No Changes", colorNoChanges))
	}

	// Drift detected badge
	if s.DriftDetected {
		badges = append(badges, createShieldsIOBadge("", "Drift Detected", colorOrange))
	}

	// Failures badge for apply phase
	if s.Phase == PhaseApply && len(s.Failures) > 0 {
		failureMsg := fmt.Sprintf("Failed (%d)", len(s.Failures))
		badges = append(badges, createShieldsIOBadge("", failureMsg, colorRed))
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

	writeColoredResourceGroup(b, "Create", "+", s.Creates, colorGreen)
	writeColoredResourceGroup(b, "Modify", "~", s.Updates, colorYellow)
	writeColoredResourceGroup(b, "Destroy", "-", s.Destroys, colorRed)
	writeColoredResourceGroup(b, "Replace", "-/+", s.Replaces, colorRed)
	writeColoredResourceGroup(b, "Import", "←", s.Imports, colorImport)
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

func writeColoredResourceGroup(b *strings.Builder, title, prefix string, resources []ResourceChange, color string) {
	if len(resources) == 0 {
		return
	}

	badge := createShieldsIOBadge("", fmt.Sprintf("%s (%d)", title, len(resources)), color)
	b.WriteString(fmt.Sprintf("<details>\n<summary>%s</summary>\n\n```diff\n", badge))

	for _, r := range resources {
		line := fmt.Sprintf("%s %s", prefix, r.Address)
		if r.Error != "" {
			line += fmt.Sprintf("  # ERROR: %s", r.Error)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("```\n\n</details>\n\n")
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
