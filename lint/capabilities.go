package lint

import (
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/openapi"
)

func lintCapabilityGraph(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil {
		return nil
	}
	imports, err := inspect.ImportGraph(dir)
	if err != nil {
		return []Finding{{Rule: "L004", Message: safeLoaderError(err, "source import graph is not readable")}}
	}

	declared := map[string]bool{}
	for _, capability := range m.Capabilities {
		declared[capability] = true
	}

	var findings []Finding
	for _, capability := range m.Capabilities {
		module := inspect.CapabilityModule(capability)
		if module == "" {
			continue
		}
		if capabilityConfigured(dir, m, capability) {
			continue
		}
		if len(importMatches(module, imports)) == 0 {
			findings = append(findings, Finding{
				Rule:    "L004",
				Message: "declared capability is not used by import graph: " + capability,
				Path:    "nucleus.yaml",
			})
		}
	}

	for capability, module := range knownCapabilityModules() {
		if declared[capability] {
			continue
		}
		matches := importMatches(module, imports)
		if len(matches) == 0 {
			continue
		}
		findings = append(findings, Finding{
			Rule:    "L004",
			Message: "import graph uses undeclared capability " + capability + ": " + strings.Join(matches, ", "),
			Path:    "nucleus.yaml",
		})
	}
	return findings
}

func capabilityConfigured(dir string, m manifest.Manifest, capability string) bool {
	switch capability {
	case "http":
		return httpCapabilityConfigured(dir)
	case "log":
		return providerConfigured(m.Nucleus.Providers[capability]) || providerConfigured(m.Nucleus.Log)
	case "trace":
		return providerConfigured(m.Nucleus.Providers[capability]) || providerConfigured(m.Nucleus.Trace)
	case "metric":
		return providerConfigured(m.Nucleus.Providers[capability]) || providerConfigured(m.Nucleus.Metric)
	case "sql":
		return providerConfigured(m.Nucleus.Providers[capability]) || providerConfigured(m.Nucleus.SQL)
	case "mongo":
		return providerConfigured(m.Nucleus.Providers[capability]) || providerConfigured(m.Nucleus.Mongo)
	default:
		return providerConfigured(m.Nucleus.Providers[capability])
	}
}

func httpCapabilityConfigured(dir string) bool {
	endpoints, err := openapi.LoadEndpoints(dir)
	if err != nil {
		return false
	}
	declared := map[routeKey]bool{}
	for _, endpoint := range endpoints {
		declared[routeKey{Method: endpoint.Method, Path: endpoint.Path}] = true
	}
	if len(declared) == 0 {
		return hasFreshGeneratedHTTPRouteBinder(dir)
	}
	registered, findings := registeredRoutes(dir, declared)
	if len(findings) > 0 {
		return false
	}
	for route := range declared {
		if _, ok := registered[route]; !ok {
			return false
		}
	}
	return true
}

func hasFreshGeneratedHTTPRouteBinder(dir string) bool {
	if !generatedHTTPRoutesFresh(dir) {
		return false
	}
	found := false
	for _, root := range routeRegistrationRoots(dir) {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil || found {
				return nil
			}
			if entry.IsDir() {
				if shouldSkipLintSourceDir(entry) {
					return filepath.SkipDir
				}
				return nil
			}
			if !isGoSourceFile(path) {
				return nil
			}
			file, _, err := parseGoFile(dir, path, 0)
			if err != nil {
				return nil
			}
			aliases := generatedHTTPAdapterAliases(file)
			if len(aliases) == 0 {
				return nil
			}
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if generatedHTTPRegisterRoutes(call.Fun, aliases) {
					found = true
					return false
				}
				return true
			})
			return nil
		})
	}
	return found
}

func providerConfigured(values map[string]any) bool {
	if len(values) == 0 {
		return false
	}
	provider, _ := values["provider"].(string)
	return strings.TrimSpace(provider) != ""
}

func knownCapabilityModules() map[string]string {
	values := map[string]string{}
	for _, capability := range []string{"http", "grpc", "worker", "log", "trace", "config", "httpclient", "transport", "discovery", "metric", "auth", "health", "sql", "redis", "mongo", "kv", "mq", "store", "lock", "sentinel", "errortracker", "profiler"} {
		values[capability] = inspect.CapabilityModule(capability)
	}
	return values
}

func importMatches(module string, imports []string) []string {
	var matches []string
	for _, importPath := range imports {
		if importPath == module || strings.HasPrefix(importPath, module+"/") {
			matches = append(matches, importPath)
		}
	}
	sort.Strings(matches)
	return matches
}
