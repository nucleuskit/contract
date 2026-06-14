package openapi

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
)

func TestValidateReportsMalformedOpenAPI(t *testing.T) {
	dir := writeOpenAPI(t, "openapi: [")
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.parse_failed")
}

func TestValidateReportsPathParameterWithoutDefinition(t *testing.T) {
	dir := writeOpenAPI(t, `openapi: 3.1.0
paths:
  /widgets/{id}:
    get:
      operationId: get_widget
      responses:
        "200":
          description: ok
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.path_parameter_missing")
}

func TestValidateReportsDuplicateOperationIDs(t *testing.T) {
	dir := writeOpenAPI(t, `openapi: 3.1.0
paths:
  /a:
    get:
      operationId: duplicate
      responses:
        "200":
          description: ok
  /b:
    get:
      operationId: duplicate
      responses:
        "200":
          description: ok
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.operation_id_duplicate")
}

func TestValidateReportsMissingResponses(t *testing.T) {
	dir := writeOpenAPI(t, `openapi: 3.1.0
paths:
  /widgets:
    get:
      operationId: get_widgets
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.responses_required")
}

func TestValidateReportsWhitespaceOperationID(t *testing.T) {
	dir := writeOpenAPI(t, `openapi: 3.1.0
paths:
  /widgets:
    get:
      operationId: "   "
      responses:
        "200":
          description: ok
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "openapi.operation_id_required")
}

func writeOpenAPI(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o700); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "openapi.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write openapi.yaml: %v", err)
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
