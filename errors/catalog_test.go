package errors

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
)

func TestValidateReportsMalformedErrorsYAML(t *testing.T) {
	dir := writeErrorsCatalog(t, "errors: [")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "errors.parse_failed")
}

func TestValidateReportsDuplicateErrorCodes(t *testing.T) {
	dir := writeErrorsCatalog(t, `errors:
  - code: 4001
    message: invalid
    http_status: 400
  - code: 4001
    message: duplicate
    http_status: 400
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "errors.code_duplicate")
}

func TestValidateReportsInvalidHTTPStatus(t *testing.T) {
	dir := writeErrorsCatalog(t, `errors:
  - code: 4001
    message: invalid
    http_status: 99
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "errors.http_status_invalid")
}

func TestValidateReportsWhitespaceErrorMessage(t *testing.T) {
	dir := writeErrorsCatalog(t, `errors:
  - code: 4001
    message: "   "
    http_status: 400
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "errors.message_required")
}

func TestValidateAllowsMissingErrorsCatalog(t *testing.T) {
	diagnostics := ValidateDir(t.TempDir())
	if diagnostics.Failed() {
		t.Fatalf("ValidateDir() = %#v, want no failure", diagnostics)
	}
}

func writeErrorsCatalog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o700); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "errors.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write errors.yaml: %v", err)
	}
	return dir
}

func assertDiagnostic(t *testing.T, diagnostics diagnostic.Diagnostics, code string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, diagnostics)
}
