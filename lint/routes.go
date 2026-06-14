package lint

import (
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/manifest"
	"github.com/nucleuskit/contract/openapi"
)

type routeKey struct {
	Method string
	Path   string
}

func lintRoutes(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil || !hasCapability(m, "http") {
		return nil
	}
	endpoints, err := openapi.LoadEndpoints(dir)
	if err != nil {
		return []Finding{{Rule: "L001", Message: safeLoaderError(err, "api/openapi.yaml is not readable"), Path: "api/openapi.yaml"}}
	}
	if len(endpoints) == 0 {
		return nil
	}
	declared := map[routeKey]bool{}
	for _, endpoint := range endpoints {
		declared[routeKey{Method: endpoint.Method, Path: endpoint.Path}] = true
	}

	registered, findings := registeredRoutes(dir, declared)
	for route, path := range registered {
		if !declared[route] {
			findings = append(findings, Finding{
				Rule:    "L001",
				Message: "registered route is missing from OpenAPI: " + route.Method + " " + route.Path,
				Path:    path,
			})
		}
	}
	for route := range declared {
		if _, ok := registered[route]; !ok {
			findings = append(findings, Finding{
				Rule:    "L001",
				Message: "OpenAPI route is not registered: " + route.Method + " " + route.Path,
				Path:    "api/openapi.yaml",
			})
		}
	}
	return findings
}

func hasCapability(m manifest.Manifest, capability string) bool {
	for _, item := range m.Capabilities {
		if item == capability {
			return true
		}
	}
	return false
}

func registeredRoutes(dir string, declared map[routeKey]bool) (map[routeKey]string, []Finding) {
	routes := map[routeKey]string{}
	var findings []Finding
	for _, root := range routeRegistrationRoots(dir) {
		if _, err := os.Stat(root); err != nil {
			continue
		}
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
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
			file, relPath, err := parseGoFile(dir, path, 0)
			if err != nil {
				findings = append(findings, Finding{Rule: "L001", Message: err.Error(), Path: relPath})
				return nil
			}
			generatedHTTPAliases := generatedHTTPAdapterAliases(file)
			ast.Inspect(file, func(node ast.Node) bool {
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				if isRouteRegisterWellKnownCall(call.Fun) {
					routes[routeKey{Method: "GET", Path: "/.well-known/nucleus.json"}] = relPath
					return true
				}
				if len(call.Args) >= 1 && isRegisterRoutesSelector(call.Fun) {
					if generatedHTTPRegisterRoutes(call.Fun, generatedHTTPAliases) && generatedHTTPRoutesFresh(dir) {
						for route := range declared {
							routes[route] = relPath
						}
						return true
					}
					if isRouteRegisterRoutesCall(call.Fun) {
						for _, route := range routeKeysFromComposite(call.Args[0]) {
							routes[route] = relPath
						}
					}
					return true
				}
				if len(call.Args) < 2 || !isRouteHandleCall(call.Fun) {
					return true
				}
				method, ok := routeMethod(call.Args[0])
				if !ok {
					return true
				}
				routePath, ok := stringLiteral(call.Args[1])
				if !ok {
					return true
				}
				routes[routeKey{Method: method, Path: routePath}] = relPath
				return true
			})
			return nil
		})
	}
	return routes, findings
}

func routeRegistrationRoots(dir string) []string {
	return []string{
		filepath.Join(dir, "cmd"),
		filepath.Join(dir, "internal", "app"),
		filepath.Join(dir, "internal", "adapter"),
		filepath.Join(dir, "internal", "server"),
	}
}

func generatedHTTPAdapterAliases(file *ast.File) map[string]bool {
	aliases := map[string]bool{}
	for _, spec := range file.Imports {
		importPath := strings.Trim(spec.Path.Value, `"`)
		if !strings.HasSuffix(importPath, "/internal/adapter/http/gen") {
			continue
		}
		name := filepath.Base(importPath)
		if spec.Name != nil && spec.Name.Name != "_" && spec.Name.Name != "." {
			name = spec.Name.Name
		}
		aliases[name] = true
	}
	return aliases
}

func generatedHTTPRegisterRoutes(expr ast.Expr, aliases map[string]bool) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "RegisterRoutes" {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && aliases[ident.Name]
}

func generatedHTTPRoutesFresh(dir string) bool {
	sourceHash, err := inspect.ContractSourceHash(dir)
	if err != nil || sourceHash == "" {
		return false
	}
	targetHash, err := inspect.ReadGeneratedHash(dir, "internal/adapter/http/gen")
	return err == nil && targetHash == sourceHash
}

func isRouteHandleCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "Handle" && isRouteReceiver(selector.X)
}

func isRouteRegisterWellKnownCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "RegisterWellKnown" && isRouteReceiver(selector.X)
}

func isRouteRegisterRoutesCall(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "RegisterRoutes" && isRouteReceiver(selector.X)
}

func isRegisterRoutesSelector(expr ast.Expr) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	return ok && selector.Sel.Name == "RegisterRoutes"
}

func isRouteReceiver(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	name := strings.ToLower(ident.Name)
	return name == "r" ||
		name == "mux" ||
		strings.Contains(name, "router") ||
		strings.Contains(name, "server")
}

func routeKeysFromComposite(expr ast.Expr) []routeKey {
	literal, ok := expr.(*ast.CompositeLit)
	if !ok {
		return nil
	}
	var routes []routeKey
	for _, element := range literal.Elts {
		routeLiteral, ok := element.(*ast.CompositeLit)
		if !ok {
			continue
		}
		var method string
		var routePath string
		for _, item := range routeLiteral.Elts {
			keyValue, ok := item.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			key, ok := keyValue.Key.(*ast.Ident)
			if !ok {
				continue
			}
			switch key.Name {
			case "Method":
				if value, ok := routeMethod(keyValue.Value); ok {
					method = value
				}
			case "Path":
				if value, ok := stringLiteral(keyValue.Value); ok {
					routePath = value
				}
			}
		}
		if method != "" && routePath != "" {
			routes = append(routes, routeKey{Method: method, Path: routePath})
		}
	}
	return routes
}

func routeMethod(expr ast.Expr) (string, bool) {
	if value, ok := stringLiteral(expr); ok {
		return strings.ToUpper(value), true
	}
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	ident, ok := selector.X.(*ast.Ident)
	if !ok || ident.Name != "http" {
		return "", false
	}
	switch selector.Sel.Name {
	case "MethodGet":
		return "GET", true
	case "MethodPost":
		return "POST", true
	case "MethodPut":
		return "PUT", true
	case "MethodPatch":
		return "PATCH", true
	case "MethodDelete":
		return "DELETE", true
	case "MethodHead":
		return "HEAD", true
	case "MethodOptions":
		return "OPTIONS", true
	default:
		return "", false
	}
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	if err != nil {
		return "", false
	}
	return value, true
}
