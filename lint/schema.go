package lint

import (
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

func lintSchemaVersions(dir string) []Finding {
	root := filepath.Join(dir, "contract", "schema")
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	var findings []Finding
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		relPath := relativeFindingPath(dir, path)
		data, err := os.ReadFile(path)
		if err != nil {
			findings = append(findings, Finding{Rule: "L012", Message: relPath + " is not readable", Path: relPath})
			return nil
		}
		var doc map[string]any
		if err := yaml.Unmarshal(data, &doc); err != nil {
			findings = append(findings, Finding{Rule: "L012", Message: relPath + " is not valid YAML or JSON", Path: relPath})
			return nil
		}
		version, ok := doc["x-nucleus-schema-version"].(string)
		if !ok || strings.TrimSpace(version) == "" {
			findings = append(findings, Finding{
				Rule:    "L012",
				Message: "schema file must declare x-nucleus-schema-version",
				Path:    relPath,
			})
		}
		return nil
	})
	return findings
}
