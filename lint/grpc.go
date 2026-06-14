package lint

import (
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/proto"
)

func lintGRPCProto(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil || !hasCapability(m, "grpc") {
		return nil
	}
	services, err := proto.LoadServices(dir)
	if err != nil {
		return []Finding{{Rule: "L013", Message: safeLoaderError(err, "api/proto is not readable"), Path: "api/proto"}}
	}
	var findings []Finding
	for _, service := range services {
		if len(service.Methods) == 0 {
			findings = append(findings, Finding{
				Rule:    "L013",
				Message: "gRPC service has no rpc methods: " + service.Name,
				Path:    service.Source,
			})
		}
	}
	return findings
}
