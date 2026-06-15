package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGeneratedFreshnessForDirRejectsPathTraversalTarget(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "service")
	if err := os.MkdirAll(filepath.Join(dir, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeInspectFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
info:
  title: demo
  version: 0.1.0
paths:
  /healthz:
    get:
      operationId: getHealthz
      responses:
        "204":
          description: ok
`)
	writeInspectFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
ai:
  generated:
    - ../outside
`)

	sourceHash, err := ContractSourceHash(dir)
	if err != nil {
		t.Fatalf("ContractSourceHash(): %v", err)
	}
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, FreshnessMarker), []byte(sourceHash+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	items, err := GeneratedFreshnessForDir(dir)
	if err != nil {
		t.Fatalf("GeneratedFreshnessForDir(): %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("GeneratedFreshnessForDir() = %#v, want one item", items)
	}
	if items[0].Fresh {
		t.Fatalf("GeneratedFreshnessForDir() = %#v, traversal target must not be fresh", items)
	}
	if items[0].Reason != generatedFreshnessReasonInvalidTarget {
		t.Fatalf("reason = %q, want %q", items[0].Reason, generatedFreshnessReasonInvalidTarget)
	}
}

func TestDefaultVerificationUsesAISafeLoopEntryCommands(t *testing.T) {
	verification := defaultVerification()
	want := []string{
		"nucleus validate --dir .",
		"nucleus lint --dir . --strict",
		"nucleus verify --dir . --json",
	}
	if len(verification.Commands) != len(want) {
		t.Fatalf("commands = %#v, want %#v", verification.Commands, want)
	}
	for index, command := range want {
		if verification.Commands[index] != command {
			t.Fatalf("commands[%d] = %q, want %q", index, verification.Commands[index], command)
		}
	}
}

func writeInspectFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
