package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nucleuskit/contract/openapi"
)

func TestExportDocsAndTypeScriptFromOpenAPI(t *testing.T) {
	dir := t.TempDir()
	writeExportFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{id}:
    get:
      operationId: getUser
components:
  schemas:
    User:
      type: object
      required: [id]
      properties:
        id:
          type: string
          description: User ID
        age:
          type: integer
        tags:
          type: array
          items:
            type: string
`)

	docs, err := ExportDocs(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"# Nucleus Contract",
		"## Endpoints",
		"GET /users/{id}",
		"## Schemas",
		"### User",
		"- `id` string required - User ID",
	} {
		if !strings.Contains(string(docs), want) {
			t.Fatalf("docs missing %q:\n%s", want, docs)
		}
	}

	ts, err := ExportTypeScript(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"export interface User {",
		"id: string;",
		"age?: number;",
		"tags?: string[];",
	} {
		if !strings.Contains(string(ts), want) {
			t.Fatalf("typescript missing %q:\n%s", want, ts)
		}
	}
}

func TestExportClientBundleGeneratesMultipleLanguages(t *testing.T) {
	dir := t.TempDir()
	writeExportFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{id}:
    get:
      operationId: getUser
  /orders:
    post:
      operationId: createOrder
`)

	clients, err := ExportClientBundle(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"typescript", "dart", "java", "kotlin"} {
		client, ok := clients[language]
		if !ok {
			t.Fatalf("expected %s client in bundle: %#v", language, clients)
		}
		if !strings.Contains(string(client), "getUser") || !strings.Contains(string(client), "createOrder") {
			t.Fatalf("%s client missing operations:\n%s", language, client)
		}
	}
}

func TestExportClientRejectsDuplicateNormalizedOperationNames(t *testing.T) {
	_, err := ExportClient([]openapi.Route{
		{Method: "GET", Path: "/a", OperationID: "get-user"},
		{Method: "GET", Path: "/b", OperationID: "get_user"},
	}, "typescript")
	if err == nil {
		t.Fatal("ExportClient succeeded, want duplicate operation name error")
	}
	if !strings.Contains(err.Error(), "duplicate generated client operation name") {
		t.Fatalf("error = %v, want duplicate generated client operation name", err)
	}
}

func writeExportFile(t *testing.T, dir, name, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
