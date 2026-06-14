package lint

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nucleuskit/contract/inspect"
)

func TestLintGeneratedFreshnessFindsStaleMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeFile(t, dir, "api/errors.yaml", "errors: []\n")
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  generated:
    - contract/gen
`)
	writeFile(t, dir, "contract/gen/.nucleus-source.sha256", "stale")

	findings := RunStrict(dir)
	if !hasRule(findings, "L010") {
		t.Fatalf("expected L010 finding, got %#v", findings)
	}
}

func TestLintGeneratedFreshnessRejectsTargetOutsideServiceDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/openapi.yaml", "openapi: 3.0.3\npaths: {}\n")
	writeFile(t, dir, "api/errors.yaml", "errors: []\n")
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  generated:
    - ../outside-gen
`)
	sourceHash, err := inspect.ContractSourceHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	outsideDir := filepath.Join(dir, "..", "outside-gen")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideDir, inspect.FreshnessMarker), []byte(sourceHash), 0o644); err != nil {
		t.Fatal(err)
	}

	findings := RunStrict(dir)
	if !hasRule(findings, "L010") {
		t.Fatalf("expected L010 finding for generated target path escape, got %#v", findings)
	}
}
