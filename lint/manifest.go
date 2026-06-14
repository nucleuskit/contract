package lint

import (
	"errors"
	"os"

	"github.com/nucleuskit/contract/manifest"
)

func lintManifest(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil {
		return []Finding{{Rule: "L006", Message: manifestReadError(err), Path: "nucleus.yaml"}}
	}
	var findings []Finding
	for _, err := range manifest.Validate(m) {
		findings = append(findings, Finding{Rule: "L006", Message: err.Error(), Path: "nucleus.yaml"})
	}
	return findings
}

func manifestReadError(err error) string {
	if errors.Is(err, os.ErrNotExist) {
		return "nucleus.yaml is missing"
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return "nucleus.yaml is not readable"
	}
	return err.Error()
}
