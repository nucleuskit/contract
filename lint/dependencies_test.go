package lint

import (
	"strings"
	"testing"
)

func TestLintDependenciesFindsMissingContract(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: payment-api
    contract: contract/deps/payment.openapi.yaml
    required: true
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L005") {
		t.Fatalf("expected L005 finding, got %#v", findings)
	}
}

func TestLintDependenciesAcceptsOpenAPIContract(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: payment-api
    contract: contract/deps/payment.openapi.yaml
    required: true
`)
	writeFile(t, dir, "contract/deps/payment.openapi.yaml", `openapi: 3.0.3
paths: {}
`)

	findings := Run(dir, true)
	if hasRule(findings, "L005") {
		t.Fatalf("did not expect L005 finding, got %#v", findings)
	}
}

func TestLintDependenciesAcceptsOpenAPIContractFragment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: payment-api
    contract: contract/deps/payment.openapi.yaml#/paths/~1payments
    required: true
`)
	writeFile(t, dir, "contract/deps/payment.openapi.yaml", `openapi: 3.0.3
paths:
  /payments:
    get:
      operationId: listPayments
`)

	findings := Run(dir, true)
	if hasRule(findings, "L005") {
		t.Fatalf("did not expect L005 finding, got %#v", findings)
	}
}

func TestLintDependenciesAcceptsRemoteContractRef(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: payment-api
    contract: https://contracts.example.com/payment.openapi.yaml#/paths/~1payments
    required: true
`)

	findings := Run(dir, true)
	if hasRule(findings, "L005") {
		t.Fatalf("did not expect L005 finding for remote contract ref, got %#v", findings)
	}
}

func TestLintDependenciesRejectsContractOutsideServiceDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: payment-api
    contract: ../payment.openapi.yaml
    required: true
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L005") {
		t.Fatalf("expected L005 finding for dependency path escape, got %#v", findings)
	}
	for _, finding := range findings {
		if strings.Contains(finding.Path, dir) || strings.Contains(finding.Message, dir) {
			t.Fatalf("finding leaks service dir %q: %#v", dir, finding)
		}
	}
}

func TestLintDependenciesFindsMissingOpenAPIContractFragment(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
dependencies:
  - name: payment-api
    contract: contract/deps/payment.openapi.yaml#/paths/~1missing
    required: true
`)
	writeFile(t, dir, "contract/deps/payment.openapi.yaml", `openapi: 3.0.3
paths:
  /payments:
    get:
      operationId: listPayments
`)

	findings := Run(dir, true)
	if !hasRule(findings, "L005") {
		t.Fatalf("expected L005 finding for missing fragment, got %#v", findings)
	}
}
