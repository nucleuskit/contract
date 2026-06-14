package lint

import (
	"os"
	"strings"
	"testing"
)

func TestLintSchemaVersionFindsMissingSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
`)
	writeFile(t, dir, "contract/schema/describe.schema.json", `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "example.com/describe.schema.json",
  "type": "object"
}`)

	findings := RunStrict(dir)
	if !hasRule(findings, "L012") {
		t.Fatalf("expected L012 finding, got %#v", findings)
	}
}

func TestLintSchemaVersionAcceptsExplicitSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
`)
	writeFile(t, dir, "contract/schema/describe.schema.json", `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "$id": "example.com/describe.schema.json",
  "x-nucleus-schema-version": "1.0",
  "type": "object"
}`)

	findings := RunStrict(dir)
	if hasRule(findings, "L012") {
		t.Fatalf("did not expect L012 finding, got %#v", findings)
	}
}

func TestFlowGraphSchemaDeclaresSchemaVersion(t *testing.T) {
	data, err := os.ReadFile("../schema/flow-graph.schema.json")
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("flow graph schema is not present in this module")
		}
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"x-nucleus-schema-version"`) {
		t.Fatalf("flow graph schema missing x-nucleus-schema-version:\n%s", data)
	}
}
