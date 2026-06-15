package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestContractSourceHashIsStableAcrossDirRepresentations(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "service")
	writeHashFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)

	hashAbs, err := ContractSourceHash(dir)
	if err != nil {
		t.Fatalf("ContractSourceHash(abs): %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(parent); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	hashRel, err := ContractSourceHash("service")
	if err != nil {
		t.Fatalf("ContractSourceHash(rel): %v", err)
	}
	hashSlash, err := ContractSourceHash("service" + string(filepath.Separator))
	if err != nil {
		t.Fatalf("ContractSourceHash(trailing slash): %v", err)
	}
	if hashAbs == "" {
		t.Fatal("hash is empty")
	}
	if hashAbs != hashRel || hashAbs != hashSlash {
		t.Fatalf("hash mismatch: abs=%s rel=%s slash=%s", hashAbs, hashRel, hashSlash)
	}
}

func writeHashFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
}
