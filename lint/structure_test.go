package lint

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintTopLevelDirsUsesAgentsRegistry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: alert-service
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "AGENTS.md", "# Service Rules\n\n## Top-Level Directories\n\n- `api`\n- `internal`\n")
	writeFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeFile(t, dir, "api/errors.yaml", "errors: []\n")
	writeFile(t, dir, "internal/alerts/service.go", "package alerts\n")

	findings := RunStrict(dir)
	if hasRule(findings, "L011") {
		t.Fatalf("expected AGENTS.md top-level registry to allow business service dirs, got %#v", findings)
	}
}

func TestLintTopLevelDirsFindsDirsMissingFromAgentsRegistry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: alert-service
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "AGENTS.md", "# Service Rules\n\n## Top-Level Directories\n\n- `api`\n- `cmd`\n- `internal`\n")
	for _, name := range []string{"api", "cmd", "internal", "store"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	findings := RunStrict(dir)
	if !hasRule(findings, "L011") {
		t.Fatalf("expected AGENTS.md top-level registry to enforce L011, got %#v", findings)
	}
}

func TestLintTopLevelDirsSkipsWhenAgentsHasNoRegistry(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: alert-service
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "AGENTS.md", "# Service Rules\n\nNo directory registry yet.\n")
	if err := os.MkdirAll(filepath.Join(dir, "store"), 0o755); err != nil {
		t.Fatal(err)
	}

	findings := RunStrict(dir)
	if hasRule(findings, "L011") {
		t.Fatalf("expected L011 to stay disabled without an AGENTS.md registry section, got %#v", findings)
	}
}
