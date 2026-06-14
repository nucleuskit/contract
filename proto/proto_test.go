package proto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nucleuskit/contract/diagnostic"
)

func TestValidateReportsUnbalancedServiceBlock(t *testing.T) {
	dir := writeProto(t, "greeting.proto", `syntax = "proto3";
service Greeting {
  rpc SayHello (HelloRequest) returns (HelloResponse);
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "proto.service_block_unclosed")
}

func TestValidateReportsMethodWithoutRequestOrResponse(t *testing.T) {
	dir := writeProto(t, "greeting.proto", `syntax = "proto3";
service Greeting {
  rpc SayHello;
}
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "proto.rpc_invalid")
}

func TestValidateReportsInvalidHTTPRule(t *testing.T) {
	dir := writeProto(t, "greeting.proto", `syntax = "proto3";
service Greeting {
  rpc SayHello (HelloRequest) returns (HelloResponse) {
    option (google.api.http) = {
      body: "*"
    };
  }
}
`)
	diagnostics := ValidateDir(dir)
	assertDiagnostic(t, diagnostics, "proto.http_rule_invalid")
}

func TestValidateAllowsMissingProtoDir(t *testing.T) {
	diagnostics := ValidateDir(t.TempDir())
	if diagnostics.Failed() {
		t.Fatalf("ValidateDir() = %#v, want no failure", diagnostics)
	}
}

func writeProto(t *testing.T, name string, content string) string {
	t.Helper()
	dir := t.TempDir()
	protoDir := filepath.Join(dir, "api", "proto")
	if err := os.MkdirAll(protoDir, 0o700); err != nil {
		t.Fatalf("mkdir proto: %v", err)
	}
	if err := os.WriteFile(filepath.Join(protoDir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write proto: %v", err)
	}
	return dir
}

func assertDiagnostic(t *testing.T, diagnostics diagnostic.Diagnostics, code string) {
	t.Helper()
	for _, item := range diagnostics {
		if item.Code == code {
			return
		}
	}
	t.Fatalf("diagnostic %q not found in %#v", code, diagnostics)
}
