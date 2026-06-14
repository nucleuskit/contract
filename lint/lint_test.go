package lint

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nucleuskit/contract/inspect"
)

func TestRunSkipsStrictOnlyRulesByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/http"

func main() {}
`)

	findings := Run(dir)
	if hasRule(findings, "L004") {
		t.Fatalf("did not expect default lint to run strict-only L004, got %#v", findings)
	}
}

func TestRunStrictIncludesStrictOnlyRules(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/http"

func main() {}
`)

	findings := RunStrict(dir)
	if !hasRule(findings, "L004") {
		t.Fatalf("expected strict lint to run L004, got %#v", findings)
	}
}

func TestLintPublicAPIsHaveGoDoc(t *testing.T) {
	packages, err := parser.ParseDir(token.NewFileSet(), ".", nil, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	pkg, ok := packages["lint"]
	if !ok {
		t.Fatal("lint package was not parsed")
	}
	required := map[string]bool{
		"Finding":    true,
		"Run":        true,
		"RunStrict":  true,
		"UsesImport": true,
	}
	seen := map[string]bool{}
	var missing []string
	for _, file := range pkg.Files {
		for _, declaration := range file.Decls {
			switch declaration := declaration.(type) {
			case *ast.GenDecl:
				for _, spec := range declaration.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok || !required[typeSpec.Name.Name] {
						continue
					}
					seen[typeSpec.Name.Name] = true
					if declaration.Doc == nil || strings.TrimSpace(declaration.Doc.Text()) == "" {
						missing = append(missing, typeSpec.Name.Name)
					}
				}
			case *ast.FuncDecl:
				if !required[declaration.Name.Name] {
					continue
				}
				seen[declaration.Name.Name] = true
				if declaration.Doc == nil || strings.TrimSpace(declaration.Doc.Text()) == "" {
					missing = append(missing, declaration.Name.Name)
				}
			}
		}
	}
	for name := range required {
		if !seen[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("exported lint APIs missing Go doc: %s", strings.Join(missing, ", "))
	}
}

func TestSortFindingsOrdersStableAuditOutput(t *testing.T) {
	findings := sortFindings([]Finding{
		{Rule: "L010", Path: "b", Message: "b"},
		{Rule: "L001", Path: "z", Message: "z"},
		{Rule: "L001", Path: "a", Message: "z"},
		{Rule: "L001", Path: "a", Message: "a"},
	})

	got := []string{}
	for _, finding := range findings {
		got = append(got, finding.Rule+" "+finding.Path+" "+finding.Message)
	}
	want := []string{
		"L001 a a",
		"L001 a z",
		"L001 z z",
		"L010 b b",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("sorted findings = %#v, want %#v", got, want)
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

func writeFreshGeneratedMarker(t *testing.T, dir string, target string) {
	t.Helper()
	sourceHash, err := inspect.ContractSourceHash(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, filepath.Join(target, inspect.FreshnessMarker), sourceHash)
}

func hasRule(findings []Finding, rule string) bool {
	for _, finding := range findings {
		if finding.Rule == rule {
			return true
		}
	}
	return false
}
