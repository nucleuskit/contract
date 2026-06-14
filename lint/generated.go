package lint

import (
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
)

func lintGeneratedFreshness(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil {
		return nil
	}
	sourceHash, err := inspect.ContractSourceHash(dir)
	if err != nil {
		return []Finding{{Rule: "L010", Message: "contract sources are not readable", Path: "api"}}
	}
	var findings []Finding
	for _, target := range m.AI.Generated {
		_, targetRel, ok := serviceFilePath(dir, target)
		if !ok {
			findings = append(findings, Finding{
				Rule:    "L010",
				Message: "generated target must be a relative path inside the service directory",
				Path:    "nucleus.yaml",
			})
			continue
		}
		targetHash, err := inspect.ReadGeneratedHash(dir, targetRel)
		if err != nil {
			findings = append(findings, Finding{
				Rule:    "L010",
				Message: "generated target is missing freshness marker: " + targetRel,
				Path:    targetRel,
			})
			continue
		}
		if sourceHash == "" || targetHash != sourceHash {
			findings = append(findings, Finding{
				Rule:    "L010",
				Message: "generated target is stale: " + targetRel,
				Path:    targetRel,
			})
		}
	}
	return findings
}
