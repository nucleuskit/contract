package schema_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReportSchemaIsValidJSON(t *testing.T) {
	path := filepath.Join("report.schema.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("decode report schema: %v", err)
	}
	if schema["title"] != "Nucleus Report Result" {
		t.Fatalf("title = %#v, want Nucleus Report Result", schema["title"])
	}
	if schema["x-nucleus-schema-version"] != "report.v1" {
		t.Fatalf("schema version = %#v, want report.v1", schema["x-nucleus-schema-version"])
	}
	required := schema["required"].([]any)
	for _, want := range []string{"result_kind", "schema_version", "schema_ref", "ok", "mode", "summary", "diagnostics"} {
		if !contains(required, want) {
			t.Fatalf("required = %#v, want %s", required, want)
		}
	}
	defs := schema["$defs"].(map[string]any)
	if _, ok := defs["ai_quality"].(map[string]any); !ok {
		t.Fatalf("$defs.ai_quality missing in %#v", defs)
	}
	if _, ok := defs["platform_readiness"].(map[string]any); !ok {
		t.Fatalf("$defs.platform_readiness missing in %#v", defs)
	}
	for _, name := range []string{"platform_upload_payload", "release_dry_run"} {
		def, ok := defs[name].(map[string]any)
		if !ok {
			t.Fatalf("$defs.%s missing in %#v", name, defs)
		}
		if def["additionalProperties"] != false {
			t.Fatalf("$defs.%s.additionalProperties = %#v, want false", name, def["additionalProperties"])
		}
	}
	for _, name := range []string{"service", "endpoint", "grpc_service", "grpc_method", "grpc_http_rule"} {
		def, ok := defs[name].(map[string]any)
		if !ok {
			t.Fatalf("$defs.%s missing in %#v", name, defs)
		}
		if def["additionalProperties"] != false {
			t.Fatalf("$defs.%s.additionalProperties = %#v, want false", name, def["additionalProperties"])
		}
	}
}
