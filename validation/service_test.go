package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
)

func TestValidateServiceReportsMissingDependencyContractFile(t *testing.T) {
	dir := writeManifest(t, `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: upstream
    contract: api/missing.yaml#/paths/~1widgets
    required: true
ai:
  intent: test
`)
	diagnostics := ValidateService(dir)
	assertDiagnostic(t, diagnostics, "dependency.contract_missing")
}

func TestValidateServiceReportsHTTPCapabilityWithoutContract(t *testing.T) {
	dir := writeManifest(t, `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
ai:
  intent: test
`)
	diagnostics := ValidateService(dir)
	assertDiagnostic(t, diagnostics, "capability.http_contract_missing")
}

func TestValidateServiceDoesNotReportMissingHTTPCapabilityWhenOpenAPIIsMalformed(t *testing.T) {
	dir := writeManifest(t, `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
ai:
  intent: test
`)
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o700); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "openapi.yaml"), []byte("openapi: ["), 0o600); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
	diagnostics := ValidateService(dir)
	assertDiagnostic(t, diagnostics, "openapi.parse_failed")
	assertNoDiagnostic(t, diagnostics, "capability.http_contract_missing")
}

func TestValidateServiceAllowsHelloHTTPShape(t *testing.T) {
	dir := writeManifest(t, `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
ai:
  intent: test
`)
	apiDir := filepath.Join(dir, "api")
	if err := os.MkdirAll(apiDir, 0o700); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "openapi.yaml"), []byte(`openapi: 3.1.0
paths:
  /hello/{name}:
    parameters:
      - name: name
        in: path
        required: true
        schema:
          type: string
    get:
      operationId: get_hello
      responses:
        "200":
          description: ok
`), 0o600); err != nil {
		t.Fatalf("write openapi: %v", err)
	}
	diagnostics := ValidateService(dir)
	if diagnostics.Failed() {
		t.Fatalf("ValidateService() = %#v, want no failure", diagnostics)
	}
}

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nucleus.yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
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

func assertNoDiagnostic(t *testing.T, diagnostics diagnostic.Diagnostics, code string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == code {
			t.Fatalf("diagnostic %q found in %#v", code, diagnostics)
		}
	}
}
