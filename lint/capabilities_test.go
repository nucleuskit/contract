package lint

import (
	"testing"
)

func TestLintCapabilityGraphFindsUndeclaredImport(t *testing.T) {
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

	findings := Run(dir, true)
	if !hasRule(findings, "L004") {
		t.Fatalf("expected L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphFindsP1P2UndeclaredImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/cap/errortracker"

func main() {}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L004") {
		t.Fatalf("expected L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphFindsTransportUndeclaredImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/cap/transport"

func main() {}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L004") {
		t.Fatalf("expected L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphFindsDeclaredP1P2CapabilityWithoutImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - sentinel
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L004") {
		t.Fatalf("expected L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphAcceptsDeclaredP1P2Import(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - lock
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/cap/lock"

func main() {}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L004") {
		t.Fatalf("did not expect L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphAcceptsDeclaredTransportImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - transport
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/cap/transport"

func main() {}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L004") {
		t.Fatalf("did not expect L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphAcceptsDeclaredDiscoveryImport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - discovery
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import _ "github.com/nucleuskit/cap/discovery"

func main() {}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L004") {
		t.Fatalf("did not expect L004 finding, got %#v", findings)
	}
}

func TestLintCapabilityGraphAcceptsManifestProviderConfiguration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - log
nucleus:
  providers:
    log:
      provider: noop
`)

	findings := Run(dir, true)
	if hasRule(findings, "L004") {
		t.Fatalf("did not expect L004 finding for provider-backed capability, got %#v", findings)
	}
}

func TestLintCapabilityGraphDoesNotAcceptHTTPProviderWithoutRoutes(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
nucleus:
  providers:
    http:
      provider: noop
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L004") {
		t.Fatalf("expected L004 finding for provider-only http capability, got %#v", findings)
	}
}

func TestLintCapabilityGraphAcceptsFreshGeneratedHTTPBinder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
ai:
  generated:
    - internal/adapter/http/gen
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths: {}
`)
	writeFreshGeneratedMarker(t, dir, "internal/adapter/http/gen")
	writeFile(t, dir, "internal/app/routes.go", `package app

import httpgen "example.com/demo/internal/adapter/http/gen"

func register(router any, handler httpgen.Handler) {
	httpgen.RegisterRoutes(router, handler)
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L004") {
		t.Fatalf("did not expect L004 finding for generated HTTP binder, got %#v", findings)
	}
}
