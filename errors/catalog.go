package errors

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"go.yaml.in/yaml/v3"
)

// Code represents an error code.
type Code struct {
	Code       int    `yaml:"code" json:"code"`               // Error code
	Message    string `yaml:"message" json:"message"`         // Error message
	HTTPStatus int    `yaml:"http_status" json:"http_status"` // HTTP status codes
}

type catalog struct {
	Errors []Code `yaml:"errors"`
}

// Load loads api/errors.yaml when an error catalog is present.
func Load(dir string) ([]Code, error) {
	path := filepath.Join(dir, "api", "errors.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var info catalog
	if err := yaml.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	sort.Slice(info.Errors, func(i, j int) bool {
		return info.Errors[i].Code < info.Errors[j].Code
	})
	return info.Errors, nil
}

// ValidateDir checks api/errors.yaml when an error catalog is present.
func ValidateDir(dir string) diagnostic.Diagnostics {
	path := filepath.Join(dir, "api", "errors.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return diagnostic.Diagnostics{errorDiagnostic("errors.read_failed", err.Error())}
	}

	var catalog catalog
	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return diagnostic.Diagnostics{errorDiagnostic("errors.parse_failed", "parse api/errors.yaml: "+err.Error())}
	}

	var diagnostics diagnostic.Diagnostics
	seenCodes := map[int]struct{}{}
	seenMessages := map[string]struct{}{}
	for _, item := range catalog.Errors {
		if item.Code <= 0 {
			diagnostics = append(diagnostics, errorDiagnostic("errors.code_required", "error code must be a positive integer"))
		}
		if _, ok := seenCodes[item.Code]; item.Code > 0 && ok {
			diagnostics = append(diagnostics, errorDiagnostic("errors.code_duplicate", "error codes must be unique"))
		}
		if item.Code > 0 {
			seenCodes[item.Code] = struct{}{}
		}
		message := strings.TrimSpace(item.Message)
		if message == "" {
			diagnostics = append(diagnostics, errorDiagnostic("errors.message_required", "error message is required"))
		}
		if _, ok := seenMessages[message]; message != "" && ok {
			diagnostics = append(diagnostics, errorDiagnostic("errors.message_duplicate", "error messages must be unique"))
		}
		if message != "" {
			seenMessages[message] = struct{}{}
		}
		if item.HTTPStatus < 100 || item.HTTPStatus > 599 {
			diagnostics = append(diagnostics, errorDiagnostic("errors.http_status_invalid", "http_status must be between 100 and 599"))
		}
	}
	return diagnostics
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     "api/errors.yaml",
		Message:  message,
	}
}
