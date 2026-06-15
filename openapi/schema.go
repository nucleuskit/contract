package openapi

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Schema is the subset of OpenAPI schema metadata Nucleus keeps machine-readable.
type Schema struct {
	Ref         string            `json:"ref,omitempty" yaml:"$ref"`
	Type        string            `json:"type,omitempty" yaml:"type"`
	Format      string            `json:"format,omitempty" yaml:"format"`
	Description string            `json:"description,omitempty" yaml:"description"`
	Required    []string          `json:"required,omitempty" yaml:"required"`
	Enum        []any             `json:"enum,omitempty" yaml:"enum"`
	Default     any               `json:"default,omitempty" yaml:"default"`
	Example     any               `json:"example,omitempty" yaml:"example"`
	Properties  map[string]Schema `json:"properties,omitempty" yaml:"properties"`
	Items       *Schema           `json:"items,omitempty" yaml:"items"`
}

// SchemaExample is a representative value extracted from an OpenAPI schema.
type SchemaExample struct {
	Value  any    `json:"value"`
	Source string `json:"source"`
}

type schemaDocument struct {
	Components schemaComponents `yaml:"components"`
}

type schemaComponents struct {
	Schemas map[string]Schema `yaml:"schemas"`
}

// LoadResolvedSchemas loads and resolves component schemas from api/openapi.yaml.
func LoadResolvedSchemas(dir string) (map[string]Schema, error) {
	path := filepath.Join(dir, "api", "openapi.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var doc schemaDocument
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return ResolveSchemas(doc.Components.Schemas)
}

// ResolveSchemas resolves local component schema references.
func ResolveSchemas(schemas map[string]Schema) (map[string]Schema, error) {
	resolved := make(map[string]Schema, len(schemas))
	for name, schema := range schemas {
		next, err := resolveSchema(schema, schemas, map[string]bool{name: true})
		if err != nil {
			return nil, fmt.Errorf("resolve schema %s: %w", name, err)
		}
		resolved[name] = next
	}
	return resolved, nil
}

func resolveSchema(schema Schema, root map[string]Schema, stack map[string]bool) (Schema, error) {
	if schema.Ref != "" {
		name, err := localComponentSchemaName(schema.Ref)
		if err != nil {
			return Schema{}, err
		}
		target, ok := root[name]
		if !ok {
			return Schema{}, fmt.Errorf("missing local schema ref %q", schema.Ref)
		}
		if stack[name] {
			return Schema{}, fmt.Errorf("circular local schema ref %q", schema.Ref)
		}
		nextStack := cloneStack(stack)
		nextStack[name] = true
		resolved, err := resolveSchema(target, root, nextStack)
		if err != nil {
			return Schema{}, err
		}
		return mergeRefSchema(resolved, schema), nil
	}
	for name, property := range schema.Properties {
		next, err := resolveSchema(property, root, cloneStack(stack))
		if err != nil {
			return Schema{}, fmt.Errorf("property %s: %w", name, err)
		}
		if schema.Properties == nil {
			schema.Properties = map[string]Schema{}
		}
		schema.Properties[name] = next
	}
	if schema.Items != nil {
		next, err := resolveSchema(*schema.Items, root, cloneStack(stack))
		if err != nil {
			return Schema{}, fmt.Errorf("items: %w", err)
		}
		schema.Items = &next
	}
	return schema, nil
}

// ExampleForSchema returns a representative example for scenario and report consumers.
func ExampleForSchema(schema Schema) (SchemaExample, bool) {
	if schema.Example != nil {
		return SchemaExample{Value: normalizeSchemaValue(schema.Example), Source: "example"}, true
	}
	if schema.Default != nil {
		return SchemaExample{Value: normalizeSchemaValue(schema.Default), Source: "default"}, true
	}
	if len(schema.Enum) > 0 {
		return SchemaExample{Value: normalizeSchemaValue(schema.Enum[0]), Source: "enum"}, true
	}

	if schema.Type == "object" || len(schema.Properties) > 0 {
		value := make(map[string]any, len(schema.Properties))
		names := make([]string, 0, len(schema.Properties))
		for name := range schema.Properties {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			if example, ok := ExampleForSchema(schema.Properties[name]); ok {
				value[name] = example.Value
			}
		}
		return SchemaExample{Value: value, Source: "object"}, true
	}
	if schema.Type == "array" || schema.Items != nil {
		if schema.Items == nil {
			return SchemaExample{Value: []any{}, Source: "array"}, true
		}
		item, ok := ExampleForSchema(*schema.Items)
		if !ok {
			return SchemaExample{Value: []any{}, Source: "array"}, true
		}
		return SchemaExample{Value: []any{item.Value}, Source: "array"}, true
	}

	switch schema.Type {
	case "string":
		return SchemaExample{Value: "", Source: "type"}, true
	case "integer":
		return SchemaExample{Value: 0, Source: "type"}, true
	case "number":
		return SchemaExample{Value: 0.0, Source: "type"}, true
	case "boolean":
		return SchemaExample{Value: false, Source: "type"}, true
	default:
		return SchemaExample{}, false
	}
}

func mergeRefSchema(target Schema, local Schema) Schema {
	target.Ref = ""
	if local.Type != "" {
		target.Type = local.Type
	}
	if local.Format != "" {
		target.Format = local.Format
	}
	if local.Description != "" {
		target.Description = local.Description
	}
	if len(local.Required) > 0 {
		target.Required = local.Required
	}
	if len(local.Enum) > 0 {
		target.Enum = local.Enum
	}
	if local.Default != nil {
		target.Default = local.Default
	}
	if local.Example != nil {
		target.Example = local.Example
	}
	if local.Properties != nil {
		target.Properties = local.Properties
	}
	if local.Items != nil {
		target.Items = local.Items
	}
	return target
}

func normalizeSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			normalized[key] = normalizeSchemaValue(item)
		}
		return normalized
	case map[any]any:
		normalized := make(map[string]any, len(typed))
		for key, item := range typed {
			normalized[fmt.Sprint(key)] = normalizeSchemaValue(item)
		}
		return normalized
	case []any:
		normalized := make([]any, len(typed))
		for index, item := range typed {
			normalized[index] = normalizeSchemaValue(item)
		}
		return normalized
	default:
		return value
	}
}

func localComponentSchemaName(ref string) (string, error) {
	const prefix = "#/components/schemas/"
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("unsupported ref %q", ref)
	}
	name := strings.TrimPrefix(ref, prefix)
	if name == "" {
		return "", fmt.Errorf("empty local schema ref %q", ref)
	}
	return strings.ReplaceAll(strings.ReplaceAll(name, "~1", "/"), "~0", "~"), nil
}

func cloneStack(stack map[string]bool) map[string]bool {
	clone := make(map[string]bool, len(stack))
	for key, value := range stack {
		clone[key] = value
	}
	return clone
}
