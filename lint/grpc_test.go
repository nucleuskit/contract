package lint

import (
	"testing"
)

func TestLintGRPCProtoFindsServiceWithoutRPCMethods(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - grpc
`)
	writeFile(t, dir, "api/proto/empty.proto", `syntax = "proto3";
package demo.v1;

service EmptyService {
}
`)

	findings := RunStrict(dir)
	if !hasRule(findings, "L013") {
		t.Fatalf("expected L013 finding, got %#v", findings)
	}
}

func TestLintGRPCProtoAcceptsServiceWithRPCMethods(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "nucleus.yaml", `schema_version: "1.0"
service:
  name: demo
  version: "0.1.0"
capabilities:
  - grpc
`)
	writeFile(t, dir, "api/proto/greeter.proto", `syntax = "proto3";
package demo.v1;

service GreeterService {
  rpc SayHello (HelloRequest) returns (HelloReply);
}
`)

	findings := RunStrict(dir)
	if hasRule(findings, "L013") {
		t.Fatalf("did not expect L013 finding, got %#v", findings)
	}
}
