package manifest

import (
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
)

func TestValidateReportsRequiredFields(t *testing.T) {
	diagnostics := ValidateDiagnostics(Manifest{})
	assertDiagnostic(t, diagnostics, "manifest.schema_version_required")
	assertDiagnostic(t, diagnostics, "manifest.service_name_required")
	assertDiagnostic(t, diagnostics, "manifest.service_version_required")
}

func TestValidateReportsWhitespaceRequiredFields(t *testing.T) {
	diagnostics := ValidateDiagnostics(Manifest{
		SchemaVersion: "   ",
		Service:       Service{Name: "   ", Version: "   "},
	})
	assertDiagnostic(t, diagnostics, "manifest.schema_version_required")
	assertDiagnostic(t, diagnostics, "manifest.service_name_required")
	assertDiagnostic(t, diagnostics, "manifest.service_version_required")
}

func TestValidateWarnsForMissingAISurface(t *testing.T) {
	diagnostics := ValidateDiagnostics(Manifest{
		SchemaVersion: "1.0",
		Service:       Service{Name: "demo", Version: "0.1.0"},
	})
	assertDiagnostic(t, diagnostics, "manifest.ai_intent_missing")
}

func TestValidateReportsInvalidEditSurfacePath(t *testing.T) {
	diagnostics := ValidateDiagnostics(Manifest{
		SchemaVersion: "1.0",
		Service:       Service{Name: "demo", Version: "0.1.0"},
		AI: AI{
			Intent:         "test",
			AllowedChanges: []string{"../outside"},
		},
	})
	assertDiagnostic(t, diagnostics, "manifest.edit_surface_path_invalid")
}

func TestValidateReportsDependencyFields(t *testing.T) {
	diagnostics := ValidateDiagnostics(Manifest{
		SchemaVersion: "1.0",
		Service:       Service{Name: "demo", Version: "0.1.0"},
		AI:            AI{Intent: "test"},
		Dependencies:  []Dependency{{Required: true}},
	})
	assertDiagnostic(t, diagnostics, "manifest.dependency_name_required")
	assertDiagnostic(t, diagnostics, "manifest.dependency_contract_required")
}

func TestValidateKeepsLegacyErrors(t *testing.T) {
	errs := Validate(Manifest{})
	if len(errs) == 0 {
		t.Fatal("Validate() returned no errors, want required field errors")
	}
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
