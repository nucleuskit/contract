package openapi

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// RequestShape describes the contract-derived request inputs for one operation.
type RequestShape struct {
	Method      string             `json:"method"`
	Path        string             `json:"path"`
	OperationID string             `json:"operation_id,omitempty"`
	Parameters  []RequestParameter `json:"parameters,omitempty"`
	Body        *RequestBody       `json:"body,omitempty"`
}

// RequestParameter describes one OpenAPI operation parameter with full schema metadata.
type RequestParameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required,omitempty"`
	Schema   Schema `json:"schema,omitempty"`
}

// RequestBody describes the selected JSON-compatible request body contract.
type RequestBody struct {
	Required    bool   `json:"required,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Schema      Schema `json:"schema,omitempty"`
}

// LoadRequestShapes loads request shapes from api/openapi.yaml.
func LoadRequestShapes(dir string) (map[string]RequestShape, error) {
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
	resolved, err := ResolveSchemas(doc.Components.Schemas)
	if err != nil {
		return nil, err
	}

	shapes := map[string]RequestShape{}
	for pathValue, item := range doc.Paths {
		pathParameters, err := convertRequestParameters(item.Parameters, resolved)
		if err != nil {
			return nil, err
		}
		for method, operation := range item.operations() {
			parameters := append([]RequestParameter{}, pathParameters...)
			operationParameters, err := convertRequestParameters(operation.Parameters, resolved)
			if err != nil {
				return nil, err
			}
			parameters = append(parameters, operationParameters...)
			body, err := convertRequestBody(operation.RequestBody, resolved)
			if err != nil {
				return nil, err
			}
			shape := RequestShape{
				Method:      strings.ToUpper(method),
				Path:        pathValue,
				OperationID: operation.OperationID,
				Parameters:  parameters,
				Body:        body,
			}
			shapes[RequestShapeKey(shape.Method, shape.Path, shape.OperationID)] = shape
		}
	}
	return shapes, nil
}

// RequestShapeKey returns the stable key used by LoadRequestShapes.
func RequestShapeKey(method string, path string, operationID string) string {
	if strings.TrimSpace(operationID) != "" {
		return "operation:" + operationID
	}
	return strings.ToUpper(strings.TrimSpace(method)) + " " + strings.TrimSpace(path)
}

func convertRequestParameters(parameters []routeParameter, resolved map[string]Schema) ([]RequestParameter, error) {
	converted := make([]RequestParameter, 0, len(parameters))
	for _, parameter := range parameters {
		schema, err := resolveRequestSchema(parameter.Schema, resolved)
		if err != nil {
			return nil, err
		}
		converted = append(converted, RequestParameter{
			Name:     parameter.Name,
			In:       parameter.In,
			Required: parameter.Required,
			Schema:   schema,
		})
	}
	return converted, nil
}

func convertRequestBody(body routeRequestBody, resolved map[string]Schema) (*RequestBody, error) {
	if len(body.Content) == 0 {
		if !body.Required {
			return nil, nil
		}
		return &RequestBody{Required: true}, nil
	}
	contentType := preferredContentType(body.Content)
	media := body.Content[contentType]
	schema, err := resolveRequestSchema(media.Schema, resolved)
	if err != nil {
		return nil, err
	}
	return &RequestBody{
		Required:    body.Required,
		ContentType: contentType,
		Schema:      schema,
	}, nil
}

func preferredContentType(content map[string]routeMediaType) string {
	if _, ok := content["application/json"]; ok {
		return "application/json"
	}
	keys := make([]string, 0, len(content))
	for key := range content {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys[0]
}

func resolveRequestSchema(schema Schema, resolved map[string]Schema) (Schema, error) {
	if len(resolved) == 0 {
		return schema, nil
	}
	return resolveSchema(schema, resolved, map[string]bool{})
}
