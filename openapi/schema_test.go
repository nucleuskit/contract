package openapi

import "testing"

func TestResolveSchemasResolvesLocalRefsAndExamples(t *testing.T) {
	schemas := map[string]Schema{
		"UserID": {
			Type:    "string",
			Example: "user-1",
		},
		"User": {
			Type:     "object",
			Required: []string{"id"},
			Properties: map[string]Schema{
				"id": {Ref: "#/components/schemas/UserID"},
				"tags": {
					Type: "array",
					Items: &Schema{
						Type: "string",
						Enum: []any{"beta", "alpha"},
					},
				},
			},
		},
	}

	resolved, err := ResolveSchemas(schemas)
	if err != nil {
		t.Fatalf("ResolveSchemas: %v", err)
	}
	user := resolved["User"]
	if got := user.Properties["id"].Type; got != "string" {
		t.Fatalf("id type = %q, want string", got)
	}
	example, ok := ExampleForSchema(user)
	if !ok {
		t.Fatal("ExampleForSchema returned false")
	}
	value, ok := example.Value.(map[string]any)
	if !ok {
		t.Fatalf("example value has type %T, want map", example.Value)
	}
	if value["id"] != "user-1" {
		t.Fatalf("example id = %#v, want user-1", value["id"])
	}
}

func TestResolveSchemasRejectsCircularRefs(t *testing.T) {
	_, err := ResolveSchemas(map[string]Schema{
		"A": {Ref: "#/components/schemas/B"},
		"B": {Ref: "#/components/schemas/A"},
	})
	if err == nil {
		t.Fatal("ResolveSchemas succeeded, want circular ref error")
	}
}

func TestResolveSchemasRejectsRemoteRefs(t *testing.T) {
	_, err := ResolveSchemas(map[string]Schema{
		"Remote": {Ref: "https://example.com/schema.yaml#/User"},
	})
	if err == nil {
		t.Fatal("ResolveSchemas succeeded, want unsupported ref error")
	}
}
