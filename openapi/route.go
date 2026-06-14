package openapi

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
	"go.yaml.in/yaml/v3"
)

// Route describes an HTTP operation with request metadata used by inspection.
type Route struct {
	Method              string      `json:"method"`
	Path                string      `json:"path"`
	OperationID         string      `json:"operation_id,omitempty"`
	Parameters          []Parameter `json:"parameters,omitempty"`
	RequestBodyRequired bool        `json:"request_body_required,omitempty"`
	Priority            int         `json:"priority,omitempty"`
}

// Parameter describes an OpenAPI operation parameter.
type Parameter struct {
	Name       string `json:"name"`
	In         string `json:"in"`
	Required   bool   `json:"required,omitempty"`
	SchemaType string `json:"schema_type,omitempty"`
}

type routeDocument struct {
	Paths map[string]routePathItem `yaml:"paths"`
}

type routePathItem struct {
	Parameters []routeParameter `yaml:"parameters"`
	Get        *routeOperation  `yaml:"get"`
	Post       *routeOperation  `yaml:"post"`
	Put        *routeOperation  `yaml:"put"`
	Patch      *routeOperation  `yaml:"patch"`
	Delete     *routeOperation  `yaml:"delete"`
	Head       *routeOperation  `yaml:"head"`
	Options    *routeOperation  `yaml:"options"`
}

type routeOperation struct {
	OperationID string           `yaml:"operationId"`
	Parameters  []routeParameter `yaml:"parameters"`
	RequestBody routeRequestBody `yaml:"requestBody"`
	Responses   map[string]any   `yaml:"responses"`
	Priority    int              `yaml:"x-nucleus-priority"`
}

type routeParameter struct {
	Name     string      `yaml:"name"`
	In       string      `yaml:"in"`
	Required bool        `yaml:"required"`
	Schema   routeSchema `yaml:"schema"`
}

type routeSchema struct {
	Type string `yaml:"type"`
}

type routeRequestBody struct {
	Required bool `yaml:"required"`
}

// LoadRouteRegistry loads HTTP routes from api/openapi.yaml.
func LoadRouteRegistry(dir string) ([]Route, error) {
	path := filepath.Join(dir, apiDirName, openAPIFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc routeDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	var routes []Route
	for path, item := range doc.Paths {
		pathParameters := convertParameters(item.Parameters)
		for method, operation := range item.operations() {
			parameters := append([]Parameter{}, pathParameters...)
			parameters = append(parameters, convertParameters(operation.Parameters)...)
			routes = append(routes, Route{
				Method:              strings.ToUpper(method),
				Path:                path,
				OperationID:         operation.OperationID,
				Parameters:          parameters,
				RequestBodyRequired: operation.RequestBody.Required,
				Priority:            operation.Priority,
			})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path == routes[j].Path {
			return routes[i].Method < routes[j].Method
		}
		return routes[i].Path < routes[j].Path
	})
	return routes, nil
}

// operations returns the operations for the path item.
func (item routePathItem) operations() map[string]routeOperation {
	candidates := map[string]*routeOperation{
		methodGet:     item.Get,
		methodPost:    item.Post,
		methodPut:     item.Put,
		methodPatch:   item.Patch,
		methodDelete:  item.Delete,
		methodHead:    item.Head,
		methodOptions: item.Options,
	}
	operations := make(map[string]routeOperation, len(candidates))
	for method, operation := range candidates {
		if operation != nil {
			operations[method] = *operation
		}
	}
	return operations
}

// convertParameters converts routeParameters to Parameters.
func convertParameters(parameters []routeParameter) []Parameter {
	converted := make([]Parameter, 0, len(parameters))
	for _, parameter := range parameters {
		converted = append(converted, Parameter{
			Name:       parameter.Name,
			In:         parameter.In,
			Required:   parameter.Required,
			SchemaType: parameter.Schema.Type,
		})
	}
	return converted
}

var pathParameterPattern = regexp.MustCompile(`\{([^}/]+)\}`)

// ValidateDir checks api/openapi.yaml when an OpenAPI contract is present.
func ValidateDir(dir string) diagnostic.Diagnostics {
	path := filepath.Join(dir, apiDirName, openAPIFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return diagnostic.Diagnostics{errorDiagnostic("openapi.read_failed", err.Error())}
	}

	var doc routeDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return diagnostic.Diagnostics{errorDiagnostic("openapi.parse_failed", "parse api/openapi.yaml: "+err.Error())}
	}
	if len(doc.Paths) == 0 {
		return diagnostic.Diagnostics{errorDiagnostic("openapi.paths_required", "paths must contain at least one operation")}
	}

	var diagnostics diagnostic.Diagnostics
	operationIDs := map[string]struct{}{}
	for pathValue, item := range doc.Paths {
		pathParameters := convertParameters(item.Parameters)
		operations := item.operations()
		if len(operations) == 0 {
			diagnostics = append(diagnostics, errorDiagnostic("openapi.operation_required", "path entries must define at least one operation"))
			continue
		}
		for _, operation := range operations {
			operationID := strings.TrimSpace(operation.OperationID)
			if operationID == "" {
				diagnostics = append(diagnostics, errorDiagnostic("openapi.operation_id_required", "operationId is required"))
			}
			if _, ok := operationIDs[operationID]; operationID != "" && ok {
				diagnostics = append(diagnostics, errorDiagnostic("openapi.operation_id_duplicate", "operationId values must be unique"))
			}
			if operationID != "" {
				operationIDs[operationID] = struct{}{}
			}
			parameters := append([]Parameter{}, pathParameters...)
			parameters = append(parameters, convertParameters(operation.Parameters)...)
			for _, parameterName := range pathParameterNames(pathValue) {
				if !hasRequiredPathParameter(parameterName, parameters) {
					diagnostics = append(diagnostics, errorDiagnostic("openapi.path_parameter_missing", "path parameters must have matching required in: path parameters"))
				}
			}
			if len(operation.Responses) == 0 {
				diagnostics = append(diagnostics, errorDiagnostic("openapi.responses_required", "operations must define at least one response"))
			}
		}
	}
	return diagnostics
}

func pathParameterNames(pathValue string) []string {
	matches := pathParameterPattern.FindAllStringSubmatch(pathValue, -1)
	names := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			names = append(names, match[1])
		}
	}
	return names
}

func hasRequiredPathParameter(name string, parameters []Parameter) bool {
	for _, parameter := range parameters {
		if strings.TrimSpace(parameter.Name) == name && strings.TrimSpace(parameter.In) == "path" && parameter.Required {
			return true
		}
	}
	return false
}

func errorDiagnostic(code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     "api/openapi.yaml",
		Message:  message,
	}
}
