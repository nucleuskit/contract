package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWritesArtifactsAndMarker(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	result, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 8 {
		t.Fatalf("expected 8 generated files, got %d", len(result.Files))
	}
	for _, name := range []string{"errors.go", "endpoints.go", "contract_source.go", ".nucleus-source.sha256"} {
		if _, err := os.Stat(filepath.Join(dir, "contract", "gen", name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
	for _, name := range []string{"handler.gen.go", "types.gen.go", "routes.gen.go", ".nucleus-source.sha256"} {
		if _, err := os.Stat(filepath.Join(dir, "internal", "adapter", "http", "gen", name)); err != nil {
			t.Fatalf("expected HTTP adapter generated %s to exist: %v", name, err)
		}
	}
}

func TestGenerateWritesHTTPServerClientAndSourceArtifacts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	result, err := GenerateWithOptions(dir, Options{HTTP: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 7 {
		t.Fatalf("expected 7 generated files, got %d: %#v", len(result.Files), result.Files)
	}
	for _, name := range []string{"endpoints.go", "contract_source.go", ".nucleus-source.sha256"} {
		if _, err := os.Stat(filepath.Join(dir, "contract", "gen", name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}
	assertFileContains(t, filepath.Join(dir, "internal", "adapter", "http", "gen", "handler.gen.go"), "type Handler interface")
	assertFileContains(t, filepath.Join(dir, "internal", "adapter", "http", "gen", "routes.gen.go"), "func RegisterRoutes")
	assertFileContains(t, filepath.Join(dir, "contract", "gen", "contract_source.go"), "EmbeddedContractSources")
	assertFileContains(t, filepath.Join(dir, "contract", "gen", "contract_source.go"), result.Hash)
}

func TestGenerateWritesHTTPRouteAndClientMetadata(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{user_id}:
    parameters:
      - name: user_id
        in: path
        required: true
        schema:
          type: string
    post:
      operationId: createUser
      x-nucleus-priority: 8
      parameters:
        - name: dry_run
          in: query
          required: false
          schema:
            type: boolean
      requestBody:
        required: true
      responses:
        "200":
          description: ok
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)

	if _, err := GenerateWithOptions(dir, Options{HTTP: true}); err != nil {
		t.Fatal(err)
	}

	typesPath := filepath.Join(dir, "internal", "adapter", "http", "gen", "types.gen.go")
	assertFileContains(t, typesPath, "type RouteParameter struct")
	assertFileContains(t, typesPath, "Parameters          []RouteParameter")
	assertFileContains(t, typesPath, `RequestBodyRequired: true`)
	assertFileContains(t, typesPath, `Priority: 8`)
	assertFileContains(t, typesPath, `{Name: "user_id", In: "path", Required: true, SchemaType: "string"}`)
	assertFileContains(t, typesPath, `{Name: "dry_run", In: "query", Required: false, SchemaType: "boolean"}`)

	routesPath := filepath.Join(dir, "internal", "adapter", "http", "gen", "routes.gen.go")
	assertFileContains(t, routesPath, "func RegisterRoutes(server *runtimehttp.Server, handler Handler)")
	assertFileContains(t, routesPath, `runtimehttp "github.com/nucleuskit/http"`)
	assertFileContains(t, filepath.Join(dir, "internal", "adapter", "http", "gen", "handler.gen.go"), "CreateUser(request *http.Request) (any, error)")
	assertFileContains(t, routesPath, `Method: "POST", Path: "/users/{user_id}", OperationID: "createUser"`)
	assertFileContains(t, routesPath, "return handler.CreateUser(request)")

	if _, err := GenerateWithOptions(dir, Options{Clients: true}); err != nil {
		t.Fatal(err)
	}
	clientPath := filepath.Join(dir, "sdk", "go", "client.gen.go")
	assertFileContains(t, clientPath, "type ClientOperation struct")
	assertFileContains(t, clientPath, "var ClientOperations = []ClientOperation")
	assertFileContains(t, clientPath, `{Method: "POST", Path: "/users/{user_id}", OperationID: "createUser", MethodName: "CreateUser", RequestBodyRequired: true}`)
}

func TestGenerateRejectsDuplicateNormalizedHandlerNames(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/a:
    get:
      operationId: get-user
  /users/b:
    get:
      operationId: get_user
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 1001
    message: failed
    http_status: 500
`)

	_, err := GenerateWithOptions(dir, Options{HTTP: true})
	if err == nil {
		t.Fatal("GenerateWithOptions succeeded, want duplicate handler name error")
	}
	if !strings.Contains(err.Error(), "duplicate generated handler name") {
		t.Fatalf("error = %v, want duplicate generated handler name", err)
	}
}

func TestGenerateRejectsDuplicateNormalizedErrorConstNames(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 1001
    message: user-missing
    http_status: 404
  - code: 1002
    message: user_missing
    http_status: 404
`)

	_, err := GenerateWithOptions(dir, Options{Errors: true})
	if err == nil {
		t.Fatal("GenerateWithOptions succeeded, want duplicate error const name error")
	}
	if !strings.Contains(err.Error(), "duplicate generated error code name") {
		t.Fatalf("error = %v, want duplicate generated error code name", err)
	}
}

func writeFile(t *testing.T, dir string, name string, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("expected %s to contain %q, got:\n%s", path, want, string(data))
	}
}
