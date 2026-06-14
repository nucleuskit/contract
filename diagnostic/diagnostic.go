// Package diagnostic defines reusable validation findings for contract packages.
package diagnostic

import "sort"

// Severity classifies whether a diagnostic should fail validation.
type Severity string

const (
	// SeverityError marks a validation failure.
	SeverityError Severity = "error"
	// SeverityWarning marks a non-fatal validation concern.
	SeverityWarning Severity = "warning"
)

// Diagnostic describes one validation finding.
type Diagnostic struct {
	Severity Severity `json:"severity"`       // Severity level
	Code     string   `json:"code"`           // Error code
	Path     string   `json:"path,omitempty"` // Error path
	Message  string   `json:"message"`        // Error message
}

// Diagnostics is a sortable collection of validation findings.
type Diagnostics []Diagnostic

// Failed reports whether any diagnostic is fatal.
func (diagnostics Diagnostics) Failed() bool {
	for _, item := range diagnostics {
		if item.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Sort orders diagnostics deterministically for CLI output and tests.
func (diagnostics Diagnostics) Sort() {
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path == diagnostics[j].Path {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Path < diagnostics[j].Path
	})
}

// Count returns the number of diagnostics with the provided severity.
func (diagnostics Diagnostics) Count(severity Severity) int {
	count := 0
	for _, item := range diagnostics {
		if item.Severity == severity {
			count++
		}
	}
	return count
}
