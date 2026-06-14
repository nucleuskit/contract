package manifest

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"go.yaml.in/yaml/v3"
)

// Manifest 清单

// Manifest represents the service manifest.
type Manifest struct {
	SchemaVersion string       `yaml:"schema_version" json:"schema_version"` // Schema version
	Service       Service      `yaml:"service" json:"service"`               // Service
	AI            AI           `yaml:"ai" json:"ai"`                         // AI skills
	Nucleus       Nucleus      `yaml:"nucleus" json:"nucleus"`               // Nucleus core
	Capabilities  []string     `yaml:"capabilities" json:"capabilities"`     // Capabilities
	Dependencies  []Dependency `yaml:"dependencies" json:"dependencies"`     // Dependencies
	Features      []string     `yaml:"features" json:"features"`             // Features
}

// Service represents service metadata.
type Service struct {
	Name        string            `yaml:"name" json:"name"`                         // Name
	Version     string            `yaml:"version" json:"version"`                   // Version
	Env         string            `yaml:"env" json:"env,omitempty"`                 // Environment
	Owner       string            `yaml:"owner" json:"owner,omitempty"`             // Owner
	Tier        string            `yaml:"tier" json:"tier,omitempty"`               // Tier
	Namespace   string            `yaml:"namespace" json:"namespace,omitempty"`     // Namespace
	Metadata    map[string]string `yaml:"metadata" json:"metadata,omitempty"`       // Metadata
	Description string            `yaml:"description" json:"description,omitempty"` // Description
}

// Nucleus represents Nucleus core configuration.
type Nucleus struct {
	PlatformURL string                    `yaml:"platform_url" json:"platform_url,omitempty"` // Platform URL
	Registry    map[string]any            `yaml:"registry" json:"registry,omitempty"`         // Registry
	Config      map[string]any            `yaml:"config" json:"config,omitempty"`             // Configuration
	Providers   map[string]map[string]any `yaml:"providers" json:"providers,omitempty"`       // Providers
	SQL         map[string]any            `yaml:"sql" json:"sql,omitempty"`                   // SQL
	Mongo       map[string]any            `yaml:"mongo" json:"mongo,omitempty"`               // MongoDB
	Gateway     map[string]any            `yaml:"gateway" json:"gateway,omitempty"`           // Gateway
	Trace       map[string]any            `yaml:"trace" json:"trace,omitempty"`               // Trace
	Log         map[string]any            `yaml:"log" json:"log,omitempty"`                   // Log
	Metric      map[string]any            `yaml:"metric" json:"metric,omitempty"`             // Metrics
}

// Dependency represents a service dependency.
type Dependency struct {
	Name     string `yaml:"name" json:"name"`         // Name
	Contract string `yaml:"contract" json:"contract"` // Contract
	Required bool   `yaml:"required" json:"required"` // Required
}

// AI represents the AI skills configuration.
type AI struct {
	Intent         string   `yaml:"intent" json:"intent,omitempty"`                   // Intent
	AllowedChanges []string `yaml:"allowed_changes" json:"allowed_changes,omitempty"` // Allowed Changes
	ReadOnly       []string `yaml:"readonly" json:"readonly,omitempty"`               // Read Only
	Forbidden      []string `yaml:"forbidden" json:"forbidden,omitempty"`             // Forbidden
	Generated      []string `yaml:"generated" json:"generated,omitempty"`             // Generated
}

// Load loads the manifest file.
func Load(dir string) (Manifest, error) {
	path := filepath.Join(dir, "nucleus.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("parse nucleus.yaml: %w", err)
	}
	return manifest, nil
}

// Validate checks manifest fields that are required before service generation or inspection.
func Validate(manifest Manifest) []error {
	diagnostics := ValidateDiagnostics(manifest)
	errs := make([]error, 0, len(diagnostics))
	for _, item := range diagnostics {
		if item.Severity == diagnostic.SeverityError {
			errs = append(errs, fmt.Errorf("%s", item.Message))
		}
	}
	return errs
}

// ValidateDiagnostics checks manifest fields and returns structured diagnostics.
func ValidateDiagnostics(manifest Manifest) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	if strings.TrimSpace(manifest.SchemaVersion) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("manifest.schema_version_required", "schema_version is required"))
	}
	if strings.TrimSpace(manifest.Service.Name) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("manifest.service_name_required", "service.name is required"))
	}
	if strings.TrimSpace(manifest.Service.Version) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("manifest.service_version_required", "service.version is required"))
	}
	if strings.TrimSpace(manifest.AI.Intent) == "" {
		diagnostics = append(diagnostics, warningDiagnostic("manifest.ai_intent_missing", "ai.intent is recommended for AI-safe changes"))
	}
	//  capabilities check
	diagnostics = append(diagnostics, validateUniqueCapabilities(manifest.Capabilities)...)
	//  dependencies check
	diagnostics = append(diagnostics, validateDependencies(manifest.Dependencies)...)
	// edit surface check
	diagnostics = append(diagnostics, validateEditSurface("ai.allowed_changes", manifest.AI.AllowedChanges)...)
	diagnostics = append(diagnostics, validateEditSurface("ai.readonly", manifest.AI.ReadOnly)...)
	diagnostics = append(diagnostics, validateEditSurface("ai.forbidden", manifest.AI.Forbidden)...)
	diagnostics = append(diagnostics, validateEditSurface("ai.generated", manifest.AI.Generated)...)
	return diagnostics
}

// validateUniqueCapabilities checks capability fields and returns structured diagnostics.
func validateUniqueCapabilities(capabilities []string) diagnostic.Diagnostics {
	seen := map[string]struct{}{}
	var diagnostics diagnostic.Diagnostics
	for _, capability := range capabilities {
		capability = strings.TrimSpace(capability)
		if capability == "" {
			diagnostics = append(diagnostics, errorDiagnostic("manifest.capability_empty", "capabilities entries must not be empty"))
			continue
		}
		if _, ok := seen[capability]; ok {
			diagnostics = append(diagnostics, errorDiagnostic("manifest.capability_duplicate", "capabilities entries must be unique"))
			continue
		}
		seen[capability] = struct{}{}
	}
	return diagnostics
}

// validateDependencies checks dependency fields and returns structured diagnostics.
func validateDependencies(dependencies []Dependency) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, dependency := range dependencies {
		if strings.TrimSpace(dependency.Name) == "" {
			diagnostics = append(diagnostics, errorDiagnostic("manifest.dependency_name_required", "dependencies entries require name"))
		}
		if strings.TrimSpace(dependency.Contract) == "" {
			diagnostics = append(diagnostics, errorDiagnostic("manifest.dependency_contract_required", "dependencies entries require contract"))
		}
	}
	return diagnostics
}

// validateEditSurface checks edit surface fields and returns structured diagnostics.
func validateEditSurface(field string, values []string) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	for _, value := range values {
		if invalidManifestPath(value) {
			diagnostics = append(diagnostics, diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "manifest.edit_surface_path_invalid",
				Path:     "nucleus.yaml",
				Message:  field + " entries must be relative paths inside the service directory",
			})
		}
	}
	return diagnostics
}

// invalidManifestPath reports whether the provided path is invalid for the manifest.
func invalidManifestPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return true
	}
	for _, part := range strings.Split(path.Clean(strings.ReplaceAll(value, "\\", "/")), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

// errorDiagnostic creates an error diagnostic.
func errorDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     "nucleus.yaml",
		Message:  message,
	}
}

// warningDiagnostic creates a warning diagnostic.
func warningDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityWarning,
		Code:     code,
		Path:     "nucleus.yaml",
		Message:  message,
	}
}
