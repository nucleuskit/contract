package lint

import (
	"strings"
	"testing"
)

func TestLintDomainImportFindsForbiddenFrameworkImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "internal/domain/order/usecase.go", `package order

import _ "github.com/gin-gonic/gin"
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L003") {
		t.Fatalf("expected L003 finding, got %#v", findings)
	}
}

func TestLintCriticalServiceRejectsLegacyBridgeImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
  tier: critical
capabilities: []
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/bridge/legacy/opentracing"

func main() {}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L007") {
		t.Fatalf("expected L007 finding, got %#v", findings)
	}
}

func TestLintImportParseErrorUsesRelativePath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "internal/domain/order/bad.go", `package order

import (
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L003") {
		t.Fatalf("expected L003 finding, got %#v", findings)
	}
	for _, finding := range findings {
		if finding.Rule != "L003" {
			continue
		}
		if strings.Contains(finding.Path, dir) || strings.Contains(finding.Message, dir) {
			t.Fatalf("finding leaks service dir %q: %#v", dir, finding)
		}
		if finding.Path != "internal/domain/order/bad.go" {
			t.Fatalf("finding path = %q, want relative source path", finding.Path)
		}
	}
}
