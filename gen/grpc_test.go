package gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateWritesGRPCServiceMetadataWhenProtoExists(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "api/errors.yaml", `errors:
  - code: 0
    message: ok
    http_status: 200
`)
	writeFile(t, dir, "api/proto/greeter.proto", `syntax = "proto3";
package nucleus.examples.hello.v1;

service GreeterService {
  rpc SayHello (HelloRequest) returns (HelloReply);
}
`)

	result, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 9 {
		t.Fatalf("expected errors, HTTP metadata/stubs, grpc metadata, source metadata, and marker; got %d files: %#v", len(result.Files), result.Files)
	}

	data, err := os.ReadFile(filepath.Join(dir, "contract", "gen", "grpc.go"))
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	for _, want := range []string{"GRPCServices", "GreeterService", "SayHello", "nucleus.examples.hello.v1"} {
		if !strings.Contains(output, want) {
			t.Fatalf("generated grpc.go missing %q:\n%s", want, output)
		}
	}
}
