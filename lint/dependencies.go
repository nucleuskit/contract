package lint

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/nucleuskit/contract/manifest"
	"go.yaml.in/yaml/v3"
)

func lintDependencies(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil {
		return nil
	}
	var findings []Finding
	for _, dependency := range m.Dependencies {
		if dependency.Contract == "" {
			findings = append(findings, Finding{Rule: "L005", Message: "dependency contract is required: " + dependency.Name, Path: "nucleus.yaml"})
			continue
		}
		path, fragment := dependencyContractRef(dependency.Contract)
		if remoteContractRef(path) {
			continue
		}
		contractPath, contractRel, ok := serviceFilePath(dir, path)
		if !ok || urlScheme(path) != "" {
			findings = append(findings, Finding{
				Rule:    "L005",
				Message: "dependency contract must be a relative path inside the service directory: " + dependency.Name,
				Path:    "nucleus.yaml",
			})
			continue
		}
		if err := validateDependencyContract(contractPath, contractRel, fragment); err != nil {
			findings = append(findings, Finding{Rule: "L005", Message: err.Error(), Path: contractRel})
		}
	}
	return findings
}

func dependencyContractRef(contract string) (string, string) {
	path, fragment, _ := strings.Cut(contract, "#")
	return path, fragment
}

func validateDependencyContract(path string, displayPath string, fragment string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%s does not exist", displayPath)
		}
		return fmt.Errorf("%s is not readable", displayPath)
	}
	switch {
	case strings.HasSuffix(path, ".proto"):
		if len(bytes.TrimSpace(data)) == 0 {
			return fmt.Errorf("%s is empty", displayPath)
		}
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"), strings.HasSuffix(path, ".json"):
		var doc map[string]any
		if err := yaml.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("%s is not valid YAML or JSON", displayPath)
		}
		if _, ok := doc["openapi"]; !ok {
			return fmt.Errorf("%s is not an OpenAPI contract", displayPath)
		}
		if fragment != "" && !jsonPointerExists(doc, fragment) {
			return fmt.Errorf("%s does not contain fragment #%s", displayPath, fragment)
		}
	default:
		return fmt.Errorf("%s has unsupported contract extension", displayPath)
	}
	return nil
}

func remoteContractRef(contract string) bool {
	parsed, err := url.Parse(contract)
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func urlScheme(value string) string {
	parsed, err := url.Parse(value)
	if err != nil {
		return ""
	}
	return parsed.Scheme
}

func jsonPointerExists(doc any, fragment string) bool {
	if fragment == "" {
		return true
	}
	decoded, err := url.PathUnescape(fragment)
	if err != nil {
		return false
	}
	if !strings.HasPrefix(decoded, "/") {
		return false
	}
	current := doc
	for _, rawToken := range strings.Split(strings.TrimPrefix(decoded, "/"), "/") {
		token := strings.ReplaceAll(strings.ReplaceAll(rawToken, "~1", "/"), "~0", "~")
		switch value := current.(type) {
		case map[string]any:
			next, ok := value[token]
			if !ok {
				return false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(value) {
				return false
			}
			current = value[index]
		default:
			return false
		}
	}
	return true
}
