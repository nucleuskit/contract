package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestServeSchemaIsValidJSON(t *testing.T) {
	path := filepath.Join("serve.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode serve schema: %v", err)
	}
	if schema["title"] != "Nucleus Serve Result" {
		t.Fatalf("title = %#v, want Nucleus Serve Result", schema["title"])
	}
	if schema["x-nucleus-schema-version"] != "serve.v1" {
		t.Fatalf("schema version = %#v, want serve.v1", schema["x-nucleus-schema-version"])
	}
	required := schema["required"].([]any)
	for _, want := range []string{"result_kind", "schema_version", "schema_ref", "ok", "mode", "summary", "diagnostics", "server"} {
		if !contains(required, want) {
			t.Fatalf("required = %#v, want %s", required, want)
		}
	}
	defs := schema["$defs"].(map[string]any)
	for _, name := range []string{"summary", "server", "metadata_endpoint", "diagnostic"} {
		def, ok := defs[name].(map[string]any)
		if !ok {
			t.Fatalf("$defs.%s missing in %#v", name, defs)
		}
		if def["additionalProperties"] != false {
			t.Fatalf("$defs.%s.additionalProperties = %#v, want false", name, def["additionalProperties"])
		}
	}
}
