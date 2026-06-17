package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestEvidenceSchemaIsValidJSON(t *testing.T) {
	path := filepath.Join("evidence.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode evidence schema: %v", err)
	}
	if schema["title"] != "Nucleus Evidence" {
		t.Fatalf("title = %#v, want Nucleus Evidence", schema["title"])
	}
	if _, ok := schema["oneOf"].([]any); !ok {
		t.Fatalf("oneOf has type %T, want []any", schema["oneOf"])
	}
	defs := schema["$defs"].(map[string]any)
	execution := defs["execution_evidence"].(map[string]any)
	allOf := execution["allOf"].([]any)
	then := allOf[0].(map[string]any)["then"].(map[string]any)
	required := then["required"].([]any)
	for _, want := range []string{"schema_ref", "assertion_results", "redaction_applied", "executed_by_package"} {
		if !contains(required, want) {
			t.Fatalf("http_scenario required = %#v, want %s", required, want)
		}
	}
}

func contains(values []any, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
