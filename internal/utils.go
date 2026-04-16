package internal

import (
	"regexp"

	"github.com/charmbracelet/x/ansi"
)

// stripANSI removes ANSI escape sequences (both ESC and bracket-only formats).
func stripANSI(s string) string {
	s = ansi.Strip(s)                                                    // ESC sequences
	s = regexp.MustCompile(`\[[0-9;]*m`).ReplaceAllString(s, "")        // Legacy bracket codes
	return s
}
