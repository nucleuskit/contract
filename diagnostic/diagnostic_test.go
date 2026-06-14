package diagnostic

import "testing"

func TestDiagnosticsFailedWhenErrorsExist(t *testing.T) {
	diagnostics := Diagnostics{
		{Severity: SeverityWarning, Code: "manifest.owner_missing"},
		{Severity: SeverityError, Code: "manifest.service_name_required"},
	}
	if !diagnostics.Failed() {
		t.Fatal("Failed() = false, want true")
	}
}

func TestDiagnosticsSortByPathThenCode(t *testing.T) {
	diagnostics := Diagnostics{
		{Path: "nucleus.yaml", Code: "b"},
		{Path: "api/openapi.yaml", Code: "z"},
		{Path: "nucleus.yaml", Code: "a"},
	}
	diagnostics.Sort()
	got := []string{
		diagnostics[0].Path + ":" + diagnostics[0].Code,
		diagnostics[1].Path + ":" + diagnostics[1].Code,
		diagnostics[2].Path + ":" + diagnostics[2].Code,
	}
	want := []string{"api/openapi.yaml:z", "nucleus.yaml:a", "nucleus.yaml:b"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestDiagnosticsCountBySeverity(t *testing.T) {
	diagnostics := Diagnostics{
		{Severity: SeverityWarning},
		{Severity: SeverityError},
		{Severity: SeverityError},
	}
	if got := diagnostics.Count(SeverityError); got != 2 {
		t.Fatalf("Count(error) = %d, want 2", got)
	}
}
