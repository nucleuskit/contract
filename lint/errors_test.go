package lint

import (
	"testing"
)

func TestLintErrorsFindsUndeclaredHandlerCode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
  - code: 1
    message: internal error
    http_status: 500
`)
	writeFile(t, dir, "internal/adapter/http/handler.go", `package http

import nucleuserrors "github.com/nucleuskit/core/errors"

func getOrder() error {
	return nucleuserrors.New(nucleuserrors.CodeNotFound, "")
}
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L002") {
		t.Fatalf("expected L002 finding, got %#v", findings)
	}
}

func TestLintErrorsAcceptsDeclaredHandlerCode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities: []
`)
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
  - code: 1
    message: internal error
    http_status: 500
  - code: 3
    message: not found
    http_status: 404
`)
	writeFile(t, dir, "internal/adapter/http/handler.go", `package http

import coreerrors "github.com/nucleuskit/core/errors"

func getOrder() error {
	return coreerrors.Wrap(coreerrors.CodeNotFound, "", nil)
}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L002") {
		t.Fatalf("did not expect L002 finding, got %#v", findings)
	}
}
