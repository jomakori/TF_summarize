package internal

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// GetEnv returns the value of an environment variable or a default value.
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvWithFallback returns the first non-empty env var from the list, or default.
func GetEnvWithFallback(defaultValue string, keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return defaultValue
}

// RequireEnv returns the value of an environment variable or an error if not set.
func RequireEnv(key string) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("%s not set", key)
}

// GetEnvBool returns true if the env var is "true" or "1".
func GetEnvBool(key string) bool {
	value := strings.ToLower(os.Getenv(key))
	return value == "true" || value == "1"
}

// Color constants for shields.io badges.
const (
	ColorGreen     = "28a745" // Create / Success
	ColorRed       = "dc3545" // Destroy / Failed
	ColorYellow    = "FFC107" // Modify
	ColorNoChanges = "0366d6" // No changes
	ColorImport    = "6f42c1" // Import
	ColorOrange    = "fd7e14" // Warnings / Drift
	ColorPlan      = "007bff" // Plan phase
)

// Pre-compiled regex patterns.
var (
	cidrBlockRe     = regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2}$`)
	legacyBracketRe = regexp.MustCompile(`\[[0-9;]*m`)
)

// StripANSI removes ANSI escape sequences.
func StripANSI(s string) string {
	s = ansi.Strip(s)
	s = legacyBracketRe.ReplaceAllString(s, "")
	return s
}

// CreateShieldsIOBadge creates a shields.io badge markdown image.
func CreateShieldsIOBadge(label, message, color string) string {
	encodedMessage := strings.ReplaceAll(url.QueryEscape(message), "+", "%20")
	return fmt.Sprintf("![%s](https://img.shields.io/badge/%s-%s-%s)", label, label, encodedMessage, color)
}

// IsValidResourceAddress validates a Terraform resource address.
func IsValidResourceAddress(addr string) bool {
	if addr == "" {
		return false
	}
	if strings.HasPrefix(addr, `"`) || strings.HasSuffix(addr, `"`) {
		return false
	}
	if strings.HasPrefix(addr, `'`) || strings.HasSuffix(addr, `'`) {
		return false
	}
	cleanAddr := strings.Trim(addr, `"',`)
	if cidrBlockRe.MatchString(cleanAddr) {
		return false
	}
	if strings.Contains(addr, "/") && strings.Count(addr, ".") >= 3 {
		return false
	}
	if len(addr) > 0 {
		first := addr[0]
		if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
			return false
		}
	}
	if !strings.Contains(addr, ".") {
		return false
	}
	return true
}

// ContainsResourceAddr checks if a resource address exists in the slice.
func ContainsResourceAddr(changes []ResourceChange, addr string) bool {
	for _, c := range changes {
		if c.Address == addr {
			return true
		}
	}
	return false
}

// FilterValidResources removes resources with invalid addresses.
func FilterValidResources(resources []ResourceChange) []ResourceChange {
	if len(resources) == 0 {
		return resources
	}
	valid := make([]ResourceChange, 0, len(resources))
	for _, r := range resources {
		if IsValidResourceAddress(r.Address) {
			valid = append(valid, r)
		}
	}
	return valid
}
