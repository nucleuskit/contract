// Package validation validates a Nucleus service manifest and contract sources.
package validation

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/openapi"
	"github.com/nucleuskit/contract/proto"
)

const (
	manifestPath = "nucleus.yaml"
)

// ValidateService checks a service directory's manifest, contracts, and local cross-file references.
func ValidateService(dir string) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	m, manifestLoaded := loadManifestDiagnostics(dir, &diagnostics)
	contractDiagnostics := diagnostic.Diagnostics{}
	contractDiagnostics = append(contractDiagnostics, openapi.ValidateDir(dir)...)
	contractDiagnostics = append(contractDiagnostics, errors.ValidateDir(dir)...)
	contractDiagnostics = append(contractDiagnostics, proto.ValidateDir(dir)...)
	diagnostics = append(diagnostics, contractDiagnostics...)
	if manifestLoaded {
		diagnostics = append(diagnostics, validateDependencyContracts(dir, m.Dependencies)...)
		diagnostics = append(diagnostics, validateCapabilityContracts(dir, m.Capabilities, contractDiagnostics)...)
		diagnostics = append(diagnostics, validateGeneratedTargets(m)...)
	}
	diagnostics.Sort()
	return diagnostics
}

// loadManifestDiagnostics loads a service's manifest and returns diagnostics.
func loadManifestDiagnostics(dir string, diagnostics *diagnostic.Diagnostics) (manifest.Manifest, bool) {
	m, err := manifest.Load(dir)
	if err != nil {
		code := "manifest.read_failed"
		if !os.IsNotExist(err) && strings.Contains(err.Error(), "parse nucleus.yaml") {
			code = "manifest.parse_failed"
		}
		*diagnostics = append(*diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     code,
			Path:     manifestPath,
			Message:  err.Error(),
		})
		return manifest.Manifest{}, false
	}
	*diagnostics = append(*diagnostics, manifest.ValidateDiagnostics(m)...)
	return m, true
}

// edit surface
func validateDependencyContracts(dir string, dependencies []manifest.Dependency) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, dependency := range dependencies {
		contractRef := strings.TrimSpace(dependency.Contract)
		if contractRef == "" || remoteContractRef(contractRef) {
			continue
		}
		refPath := strings.SplitN(contractRef, "#", 2)[0]
		if refPath == "" {
			continue
		}
		if invalidLocalRefPath(refPath) {
			diagnostics = append(diagnostics, serviceDiagnostic("dependency.contract_invalid", "dependency contract references must be relative paths inside the service directory"))
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(refPath))); err != nil {
			if os.IsNotExist(err) {
				diagnostics = append(diagnostics, serviceDiagnostic("dependency.contract_missing", "dependency contract file does not exist"))
				continue
			}
			diagnostics = append(diagnostics, serviceDiagnostic("dependency.contract_read_failed", err.Error()))
		}
	}
	return diagnostics
}

// capability
func validateCapabilityContracts(dir string, capabilities []string, existing diagnostic.Diagnostics) diagnostic.Diagnostics {
	capabilitySet := map[string]struct{}{}
	for _, capability := range capabilities {
		capabilitySet[strings.TrimSpace(capability)] = struct{}{}
	}
	endpoints, _ := openapi.LoadEndpoints(dir)
	services, _ := proto.LoadServices(dir)
	var diagnostics diagnostic.Diagnostics
	if _, ok := capabilitySet["http"]; ok && len(endpoints) == 0 && !protoHasHTTPRules(services) && !hasFatalCodePrefix(existing, "openapi.", "proto.") {
		diagnostics = append(diagnostics, serviceDiagnostic("capability.http_contract_missing", "http capability requires api/openapi.yaml endpoints or proto HTTP rules"))
	}
	if _, ok := capabilitySet["grpc"]; ok && len(services) == 0 && !hasFatalCodePrefix(existing, "proto.") {
		diagnostics = append(diagnostics, serviceDiagnostic("capability.grpc_contract_missing", "grpc capability requires api/proto services"))
	}
	return diagnostics
}

// diagnostic
func hasFatalCodePrefix(diagnostics diagnostic.Diagnostics, prefixes ...string) bool {
	for _, item := range diagnostics {
		if item.Severity != diagnostic.SeverityError {
			continue
		}
		for _, prefix := range prefixes {
			if strings.HasPrefix(item.Code, prefix) {
				return true
			}
		}
	}
	return false
}

// proto
func protoHasHTTPRules(services []proto.Service) bool {
	for _, service := range services {
		for _, method := range service.Methods {
			if len(method.HTTPRules) > 0 {
				return true
			}
		}
	}
	return false
}

// generated
func validateGeneratedTargets(m manifest.Manifest) diagnostic.Diagnostics {
	forbidden := map[string]struct{}{}
	for _, value := range m.AI.Forbidden {
		forbidden[value] = struct{}{}
	}
	var diagnostics diagnostic.Diagnostics
	for _, value := range m.AI.Generated {
		if _, ok := forbidden[value]; ok {
			diagnostics = append(diagnostics, serviceDiagnostic("manifest.generated_forbidden", "ai.generated entries must not also be forbidden"))
		}
	}
	return diagnostics
}

// contract
func remoteContractRef(value string) bool {
	return strings.Contains(value, "://")
}

// contract
func invalidLocalRefPath(value string) bool {
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return true
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	return clean == ".." || strings.HasPrefix(clean, "../")
}

// diagnostic
func serviceDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     manifestPath,
		Message:  message,
	}
}
