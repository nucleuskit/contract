package openapi

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRequestShapesResolvesParameterAndBodyExamples(t *testing.T) {
	dir := t.TempDir()
	writeRequestShapeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /users/{id}:
    get:
      operationId: getUser
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
            enum: ["u_123"]
        - name: verbose
          in: query
          required: true
          schema:
            type: boolean
            default: true
      responses:
        "200":
          description: ok
    post:
      operationId: createUser
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/CreateUser"
      responses:
        "200":
          description: ok
components:
  schemas:
    CreateUser:
      type: object
      required: [name]
      properties:
        name:
          type: string
          example: Ada
`)

	shapes, err := LoadRequestShapes(dir)
	if err != nil {
		t.Fatal(err)
	}
	getShape := shapes[RequestShapeKey("GET", "/users/{id}", "getUser")]
	if len(getShape.Parameters) != 2 {
		t.Fatalf("parameters = %#v, want two", getShape.Parameters)
	}
	if example, ok := ExampleForSchema(getShape.Parameters[0].Schema); !ok || example.Value != "u_123" || example.Source != "enum" {
		t.Fatalf("unexpected path example: %#v ok=%t", example, ok)
	}
	if example, ok := ExampleForSchema(getShape.Parameters[1].Schema); !ok || example.Value != true || example.Source != "default" {
		t.Fatalf("unexpected query example: %#v ok=%t", example, ok)
	}
	postShape := shapes[RequestShapeKey("POST", "/users/{id}", "createUser")]
	if postShape.Body == nil || !postShape.Body.Required || postShape.Body.ContentType != "application/json" {
		t.Fatalf("unexpected body shape: %#v", postShape.Body)
	}
	example, ok := ExampleForSchema(postShape.Body.Schema)
	if !ok {
		t.Fatal("expected body example")
	}
	body, ok := example.Value.(map[string]any)
	if !ok || body["name"] != "Ada" {
		t.Fatalf("unexpected body example: %#v", example)
	}
}

func writeRequestShapeFile(t *testing.T, dir, name, data string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
