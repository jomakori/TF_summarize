package render

import (
	"fmt"
	"strings"

	"github.com/jomakori/TF_summarize/internal"
)

// MessageType represents the type of message to output (error, warning, notice).
type MessageType int

const (
	// MessageError represents an error message.
	// GHA: ::error::, Markdown: > [!ERROR]
	MessageError MessageType = iota
	// MessageWarning represents a warning message.
	// GHA: ::warning::, Markdown: > [!WARNING]
	MessageWarning
	// MessageNotice represents an informational notice.
	// GHA: ::notice::, Markdown: > [!NOTE]
	MessageNotice
	// MessageCaution represents a caution message (destructive operations).
	// GHA: ::warning::, Markdown: > [!CAUTION]
	MessageCaution
)

// writeMessage outputs a message with provider-specific formatting.
// For GHA targets (gha, pr): outputs ::error::, ::warning::, or ::notice:: workflow commands.
// For all targets: outputs markdown alerts (> [!ERROR], > [!WARNING], > [!NOTE], > [!CAUTION]).
// If context is provided (e.g., resource address), it's included in the workflow command annotation.
func writeMessage(b *strings.Builder, msgType MessageType, context, message string, target internal.OutputTarget) {
	// GHA workflow command (only for GHA/PR targets)
	if target == internal.TargetGHASummary || target == internal.TargetPR {
		var cmd string
		switch msgType {
		case MessageError:
			cmd = "error"
		case MessageWarning, MessageCaution:
			cmd = "warning"
		case MessageNotice:
			cmd = "notice"
		}
		if context != "" {
			b.WriteString(fmt.Sprintf("::%s::%s: %s\n", cmd, context, message))
		} else {
			b.WriteString(fmt.Sprintf("::%s::%s\n", cmd, message))
		}
	}

	// Markdown alert (for all targets)
	var alertType string
	switch msgType {
	case MessageError:
		alertType = "ERROR"
	case MessageWarning:
		alertType = "WARNING"
	case MessageNotice:
		alertType = "NOTE"
	case MessageCaution:
		alertType = "CAUTION"
	}
	b.WriteString(fmt.Sprintf("> [!%s]\n> %s\n\n", alertType, message))
}

// writeError writes an error message with appropriate formatting based on the target provider.
// For GHA targets (gha, pr), it outputs ::error:: workflow commands for annotations.
// For other targets (stdout), it outputs only markdown alerts.
// If context is provided (e.g., resource address), it's included in the ::error:: annotation.
func writeError(b *strings.Builder, context string, message string, target internal.OutputTarget) {
	writeMessage(b, MessageError, context, message, target)
}

// Render produces a complete markdown summary for the given Summary.
func Render(s *internal.Summary) string {
	output := RenderFull(s)
	return output.Full
}

// RenderFull produces all sections of the markdown summary and returns them separately.
func RenderFull(s *internal.Summary) *internal.RenderOutput {
	return &internal.RenderOutput{
		Summary:   RenderSummary(s),
		Details:   RenderDetails(s),
		Outputs:   RenderOutputs(s),
		RawOutput: RenderRawOutput(s),
		Full:      renderComplete(s),
	}
}

// RenderSummary produces just the header, badges, and summary line.
func RenderSummary(s *internal.Summary) string {
	var b strings.Builder
	writeHeader(&b, s)
	writeBadges(&b, s)
	writeWarnings(&b, s)
	writeErrors(&b, s)
	writeSummaryLine(&b, s)
	return b.String()
}

// RenderDetails produces the resource sections (creates, updates, destroys, etc).
func RenderDetails(s *internal.Summary) string {
	var b strings.Builder
	writeResourceSections(&b, s)
	return b.String()
}

// RenderOutputs produces terraform outputs section (if any).
func RenderOutputs(s *internal.Summary) string {
	if len(s.Outputs) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("<details>\n<summary><b>Outputs</b> (")
	b.WriteString(fmt.Sprintf("%d", len(s.Outputs)))
	b.WriteString(")</summary>\n\n```diff\n")

	for _, o := range s.Outputs {
		prefix := "+"
		switch o.Action {
		case internal.ActionCreate:
			prefix = "+"
		case internal.ActionDestroy:
			prefix = "-"
		case internal.ActionUpdate:
			prefix = "~"
		}
		b.WriteString(fmt.Sprintf("%s %s = %s\n", prefix, o.Name, o.Value))
	}

	b.WriteString("```\n\n</details>\n\n")
	return b.String()
}

// RenderRawOutput produces the collapsible raw terraform output section.
func RenderRawOutput(s *internal.Summary) string {
	var b strings.Builder
	writeRawOutput(&b, s)
	return b.String()
}

func renderComplete(s *internal.Summary) string {
	var b strings.Builder
	writeHeader(&b, s)
	writeBadges(&b, s)
	writeWarnings(&b, s)
	writeErrors(&b, s)
	writeSummaryLine(&b, s)
	writeResourceSections(&b, s)
	writeOutputs(&b, s)
	writeRawOutput(&b, s)
	return b.String()
}

func writeOutputs(b *strings.Builder, s *internal.Summary) {
	if len(s.Outputs) == 0 {
		return
	}

	b.WriteString("<details>\n<summary><b>Outputs</b> (")
	b.WriteString(fmt.Sprintf("%d", len(s.Outputs)))
	b.WriteString(")</summary>\n\n```diff\n")

	for _, o := range s.Outputs {
		prefix := "+"
		switch o.Action {
		case internal.ActionCreate:
			prefix = "+"
		case internal.ActionDestroy:
			prefix = "-"
		case internal.ActionUpdate:
			prefix = "~"
		}
		b.WriteString(fmt.Sprintf("%s %s = %s\n", prefix, o.Name, o.Value))
	}

	b.WriteString("```\n\n</details>\n\n")
}

func writeHeader(b *strings.Builder, s *internal.Summary) {
	phaseTitle := "Terraform Plan"
	if s.Phase == internal.PhaseApply {
		phaseTitle = "Terraform Apply"
	} else if s.IsDestroyPlan {
		phaseTitle = "Terraform Destroy"
	}
	b.WriteString(fmt.Sprintf("## %s\n\n", phaseTitle))
}

func writeBadges(b *strings.Builder, s *internal.Summary) {
	var badges []string

	phaseBadge := "Plan"
	phaseColor := internal.ColorPlan

	if s.Phase == internal.PhaseApply {
		phaseBadge = "Apply"
		if len(s.Failures) > 0 || (s.ApplyError != "" && !s.ApplySucceeded) {
			phaseColor = internal.ColorRed
		} else {
			phaseColor = internal.ColorGreen
		}
	} else {
		if s.IsDestroyPlan {
			phaseBadge = "Destroy"
			phaseColor = internal.ColorRed
		}
	}

	badges = append(badges, internal.CreateShieldsIOBadge("Terraform", phaseBadge, phaseColor))

	if s.ToAdd > 0 || len(s.Creates) > 0 {
		label, count := "Create", s.ToAdd
		if s.Phase == internal.PhaseApply {
			label, count = "Created", max(len(s.Creates), s.ToAdd)
		}
		badges = append(badges, internal.CreateShieldsIOBadge("", fmt.Sprintf("%s (%d)", label, count), internal.ColorGreen))
	}

	if s.ToChange > 0 {
		msg := fmt.Sprintf("Modify (%d)", s.ToChange)
		badges = append(badges, internal.CreateShieldsIOBadge("", msg, internal.ColorYellow))
	}

	if s.ToDestroy > 0 {
		msg := fmt.Sprintf("Remove (%d)", s.ToDestroy)
		badges = append(badges, internal.CreateShieldsIOBadge("", msg, internal.ColorRed))
	}

	if len(s.Replaces) > 0 {
		msg := fmt.Sprintf("Replace (%d)", len(s.Replaces))
		badges = append(badges, internal.CreateShieldsIOBadge("", msg, internal.ColorRed))
	}

	if s.ToImport > 0 {
		msg := fmt.Sprintf("Import (%d)", s.ToImport)
		badges = append(badges, internal.CreateShieldsIOBadge("", msg, internal.ColorImport))
	}

	if s.ToAdd == 0 && s.ToChange == 0 && s.ToDestroy == 0 && s.ToImport == 0 && len(s.Replaces) == 0 &&
		len(s.Creates) == 0 && len(s.Destroys) == 0 && len(s.Updates) == 0 && len(s.Failures) == 0 {
		badges = append(badges, internal.CreateShieldsIOBadge("", "No Changes", internal.ColorNoChanges))
	}

	if s.DriftDetected {
		badges = append(badges, internal.CreateShieldsIOBadge("", "Drift Detected", internal.ColorOrange))
	}

	if s.Phase == internal.PhaseApply && len(s.Failures) > 0 {
		failureMsg := fmt.Sprintf("Failed (%d)", len(s.Failures))
		badges = append(badges, internal.CreateShieldsIOBadge("", failureMsg, internal.ColorRed))
	}

	b.WriteString(strings.Join(badges, " "))
	b.WriteString("\n\n")
}

func writeWarnings(b *strings.Builder, s *internal.Summary) {
	if s.ToDestroy > 0 || len(s.Replaces) > 0 {
		if s.Phase == internal.PhasePlan {
			writeMessage(b, MessageCaution, "", "**Terraform will delete resources!** This plan contains resource delete operations. Please check the plan result very carefully.", s.TargetProvider)
		}
	}

	if s.DriftDetected {
		writeMessage(b, MessageWarning, "", "**Drift detected!** Objects have changed outside of Terraform.", s.TargetProvider)
	}

	for _, w := range s.Warnings {
		writeMessage(b, MessageWarning, "", w, s.TargetProvider)
	}
}

func writeErrors(b *strings.Builder, s *internal.Summary) {
	if s.Phase == internal.PhaseApply && len(s.Failures) > 0 {
		return
	}
	for _, e := range s.Errors {
		// Skip generic exit code messages - they're too generic and redundant
		if strings.Contains(e, "exited with code") {
			continue
		}
		writeError(b, "", e, s.TargetProvider)
	}
}

func writeSummaryLine(b *strings.Builder, s *internal.Summary) {
	if s.Phase == internal.PhaseApply {
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

func writeApplySummaryLine(b *strings.Builder, s *internal.Summary) {
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

func writeResourceSections(b *strings.Builder, s *internal.Summary) {
	if s.Phase == internal.PhaseApply {
		writeApplyResourceSections(b, s)
		return
	}

	writeColoredResourceGroup(b, "Create", "+", s.Creates, internal.ColorGreen)
	writeColoredResourceGroup(b, "Modify", "~", s.Updates, internal.ColorYellow)
	writeColoredResourceGroup(b, "Destroy", "-", s.Destroys, internal.ColorRed)
	writeColoredResourceGroup(b, "Replace", "-/+", s.Replaces, internal.ColorRed)
	writeColoredResourceGroup(b, "Import", "←", s.Imports, internal.ColorImport)
}

func writeApplyResourceSections(b *strings.Builder, s *internal.Summary) {
	writeApplyResourceGroup(b, "Created", "+", "✅", s.Creates)
	writeApplyResourceGroup(b, "Updated", "~", "✅", s.Updates)
	writeApplyResourceGroup(b, "Destroyed", "-", "✅", s.Destroys)

	if len(s.Failures) > 0 {
		b.WriteString("<details open>\n<summary><b>❌ Failed</b> (")
		b.WriteString(fmt.Sprintf("%d", len(s.Failures)))
		b.WriteString(")</summary>\n\n")

		for _, r := range s.Failures {
			b.WriteString(fmt.Sprintf("**`%s`**\n", r.Address))
			if r.Error != "" {
				writeError(b, r.Address, r.Error, s.TargetProvider)
			}
		}

		b.WriteString("</details>\n\n")
	}
}

func writeColoredResourceGroup(b *strings.Builder, title, prefix string, resources []internal.ResourceChange, color string) {
	if len(resources) == 0 {
		return
	}

	b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s</b> <b>(%d)</b></summary>\n\n```diff\n", title, len(resources)))

	for _, r := range resources {
		line := fmt.Sprintf("%s %s", prefix, r.Address)
		if r.Error != "" {
			line += fmt.Sprintf("  # ERROR: %s", r.Error)
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("```\n\n</details>\n\n")
}

func writeApplyResourceGroup(b *strings.Builder, title, prefix, icon string, resources []internal.ResourceChange) {
	if len(resources) == 0 {
		return
	}

	b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s %s</b> (%d)</summary>\n\n```diff\n", icon, title, len(resources)))

	for _, r := range resources {
		b.WriteString(fmt.Sprintf("%s %s\n", prefix, r.Address))
	}

	b.WriteString("```\n\n</details>\n\n")
}

func writeRawOutput(b *strings.Builder, s *internal.Summary) {
	raw := strings.TrimSpace(s.RawOutput)
	if raw == "" {
		return
	}

	title := "Terraform Plan Output"
	if s.Phase == internal.PhaseApply {
		title = "Terraform Apply Output"
	}

	b.WriteString(fmt.Sprintf("<details>\n<summary><b>%s</b></summary>\n\n```diff\n", title))
	b.WriteString(colorizeOutput(raw, s.Phase))
	b.WriteString("\n```\n\n</details>\n")
}

func colorizeOutput(output string, phase internal.Phase) string {
	lines := strings.Split(output, "\n")
	var result []string
	inErrorBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect Terraform error block boundaries (╷ starts, ╵ ends)
		if strings.HasPrefix(trimmed, "╷") {
			inErrorBlock = true
			result = append(result, fmt.Sprintf("- %s", line))
			continue
		}
		if strings.HasPrefix(trimmed, "╵") {
			inErrorBlock = false
			result = append(result, fmt.Sprintf("- %s", line))
			continue
		}

		// If inside an error block, highlight all lines in red
		if inErrorBlock {
			result = append(result, fmt.Sprintf("- %s", line))
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "+ "):
			result = append(result, fmt.Sprintf("+ %s", strings.TrimPrefix(trimmed, "+ ")))
		case strings.HasPrefix(trimmed, "- "):
			result = append(result, fmt.Sprintf("- %s", strings.TrimPrefix(trimmed, "- ")))
		case strings.HasPrefix(trimmed, "~ "):
			result = append(result, fmt.Sprintf("~ %s", strings.TrimPrefix(trimmed, "~ ")))
		case strings.HasPrefix(trimmed, "-/+ "):
			result = append(result, fmt.Sprintf("-/+ %s", strings.TrimPrefix(trimmed, "-/+ ")))
		case strings.Contains(trimmed, "Creating..."):
			result = append(result, fmt.Sprintf("+ %s", trimmed))
		case strings.Contains(trimmed, "Destroying..."):
			result = append(result, fmt.Sprintf("- %s", trimmed))
		case strings.Contains(trimmed, "Modifying..."):
			result = append(result, fmt.Sprintf("~ %s", trimmed))
		case strings.Contains(trimmed, "Creation complete"):
			result = append(result, fmt.Sprintf("+ %s", trimmed))
		case strings.Contains(trimmed, "Destruction complete"):
			result = append(result, fmt.Sprintf("- %s", trimmed))
		case strings.Contains(trimmed, "Modifications complete"):
			result = append(result, fmt.Sprintf("~ %s", trimmed))
		case strings.HasPrefix(trimmed, "Error:"):
			result = append(result, fmt.Sprintf("- %s", trimmed))
		case strings.HasPrefix(trimmed, "Warning:"):
			result = append(result, fmt.Sprintf("~ %s", trimmed))
		case strings.HasPrefix(trimmed, "Plan:"):
			result = append(result, trimmed)
		case strings.HasPrefix(trimmed, "Apply complete!"):
			result = append(result, trimmed)
		default:
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
