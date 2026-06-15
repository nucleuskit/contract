package inspect

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/nucleuskit/contract/manifest"
)

// GeneratedFreshnessForDir loads a service manifest and reports generated target freshness.
func GeneratedFreshnessForDir(dir string) ([]GeneratedFreshness, error) {
	m, err := manifest.Load(dir)
	if err != nil {
		return nil, err
	}
	return freshness(dir, m), nil
}

func freshness(dir string, m manifest.Manifest) []GeneratedFreshness {
	if len(m.AI.Generated) == 0 {
		return nil
	}
	sourceHash, err := ContractSourceHash(dir)
	if err != nil {
		sourceHash = ""
	}
	items := make([]GeneratedFreshness, 0, len(m.AI.Generated))
	for _, target := range m.AI.Generated {
		rel, ok := normalizeGeneratedTarget(target)
		item := GeneratedFreshness{
			Source:     contractSourceAPI,
			Target:     target,
			SourceHash: sourceHash,
		}
		if !ok {
			item.Reason = generatedFreshnessReasonInvalidTarget
			items = append(items, item)
			continue
		}
		item.Target = rel
		targetHash, err := ReadGeneratedHash(dir, rel)
		if err != nil {
			item.Reason = generatedFreshnessReasonMissingMarker
			items = append(items, item)
			continue
		}
		item.TargetHash = targetHash
		item.Fresh = sourceHash != "" && targetHash == sourceHash
		if !item.Fresh {
			item.Reason = generatedFreshnessReasonHashDiffers
		}
		items = append(items, item)
	}
	return items
}

func normalizeGeneratedTarget(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return "", false
	}
	cleaned := path.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}
