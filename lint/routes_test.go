package lint

import (
	"testing"
)

func TestLintRoutesFindsRegisteredRouteMissingFromOpenAPI(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func main() {
	server := runtimehttp.NewServer()
	server.Handle(http.MethodGet, "/internal", nil)
}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L001") {
		t.Fatalf("expected L001 finding, got %#v", findings)
	}
}

func TestLintRoutesFindsOpenAPIRouteMissingFromRegistration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func main() {
	server := runtimehttp.NewServer()
	server.Handle(http.MethodPost, "/healthz", nil)
}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L001") {
		t.Fatalf("expected L001 finding, got %#v", findings)
	}
}

func TestLintRoutesAcceptsMatchingOpenAPIAndRegistration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func main() {
	server := runtimehttp.NewServer()
	server.Handle(http.MethodGet, "/healthz", nil)
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L001") {
		t.Fatalf("did not expect L001 finding, got %#v", findings)
	}
}

func TestLintRoutesIgnoresTestFileRegistration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main_test.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func TestRoutes() {
	server := runtimehttp.NewServer()
	server.Handle(http.MethodGet, "/healthz", nil)
}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L001") {
		t.Fatalf("expected L001 finding when route is only registered in _test.go, got %#v", findings)
	}
}

func TestLintRoutesAcceptsRawStringRoutePath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main.go", "package main\n\nimport (\n\t\"net/http\"\n\n\truntimehttp \"github.com/nucleuskit/http\"\n)\n\nfunc main() {\n\tserver := runtimehttp.NewServer()\n\tserver.Handle(http.MethodGet, `/healthz`, nil)\n}\n")

	findings := Run(dir, true)
	if hasRule(findings, "L001") {
		t.Fatalf("did not expect L001 finding for raw string route path, got %#v", findings)
	}
}

func TestLintRoutesAcceptsEscapedStringRoutePath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func main() {
	server := runtimehttp.NewServer()
	server.Handle(http.MethodGet, "/\u0068ealthz", nil)
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L001") {
		t.Fatalf("did not expect L001 finding for escaped string route path, got %#v", findings)
	}
}

func TestLintRoutesAcceptsInternalAppRouterRegistration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
  - log
nucleus:
  providers:
    log:
      provider: noop
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.1.0
paths:
  /hello/{name}:
    get:
      operationId: getHello
`)
	writeFile(t, dir, "internal/app/routes.go", `package app

import "net/http"

type Router struct{}

func (Router) Handle(method string, path string, handler func()) {}

type Logger struct{}

func (Logger) Info() {}

var log Logger

func RegisterRoutes(router Router) {
	router.Handle(http.MethodGet, "/hello/{name}", func() {
		log.Info()
	})
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L001") || hasRule(findings, "L004") {
		t.Fatalf("did not expect L001 or L004 finding, got %#v", findings)
	}
}

func TestLintRoutesIgnoresUnrelatedHandleMethod(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.1.0
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "internal/app/cache.go", `package app

import "net/http"

type Widget struct{}

func (Widget) Handle(method string, path string, handler func()) {}

func Configure(widget Widget) {
	widget.Handle(http.MethodGet, "/healthz", func() {})
}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L001") {
		t.Fatalf("expected L001 finding for unrelated Handle method, got %#v", findings)
	}
}

func TestLintRoutesAcceptsRegisterRoutesComposite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func main() {
	server := runtimehttp.NewServer()
	server.RegisterRoutes([]runtimehttp.Route{{
		Method: http.MethodGet,
		Path: "/healthz",
	}})
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L001") {
		t.Fatalf("did not expect L001 finding, got %#v", findings)
	}
}

func TestLintRoutesAcceptsGeneratedHTTPBinder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /healthz:
    get:
      operationId: getHealthz
`)
	writeFreshGeneratedMarker(t, dir, "internal/adapter/http/gen")
	writeFile(t, dir, "internal/app/routes.go", `package app

import (
	runtimehttp "github.com/nucleuskit/http"
	httpgen "example.com/demo/internal/adapter/http/gen"
)

func register(server *runtimehttp.Server, handler httpgen.Handler) {
	httpgen.RegisterRoutes(server, handler)
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L001") {
		t.Fatalf("did not expect L001 finding, got %#v", findings)
	}
}

func TestLintRoutesAcceptsWellKnownRegistration(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - http
`)
	writeFile(t, dir, "api/openapi.yaml", `openapi: 3.0.3
paths:
  /.well-known/nucleus.json:
    get:
      operationId: getNucleusWellKnown
`)
	writeFile(t, dir, "cmd/demo/main.go", `package main

import (
	"net/http"

	runtimehttp "github.com/nucleuskit/http"
)

func main() {
	server := runtimehttp.NewServer()
	server.RegisterWellKnown(func(*http.Request) (runtimehttp.WellKnown, error) {
		return runtimehttp.WellKnown{}, nil
	})
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L001") {
		t.Fatalf("did not expect L001 finding, got %#v", findings)
	}
}
