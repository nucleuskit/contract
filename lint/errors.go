package lint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"

	"github.com/nucleuskit/contract/errors"
)

func lintErrorCodes(dir string) []Finding {
	registered := map[int]bool{}
	codes, err := errors.Load(dir)
	if err != nil {
		return []Finding{{Rule: "L002", Message: safeLoaderError(err, "api/errors.yaml is not readable"), Path: "api/errors.yaml"}}
	}
	for _, code := range codes {
		registered[code.Code] = true
	}

	var findings []Finding
	for _, root := range []string{filepath.Join(dir, "internal", "adapter"), filepath.Join(dir, "cmd")} {
		findings = append(findings, lintCoreErrorCodeRefs(dir, root, registered)...)
	}
	return findings
}

func lintCoreErrorCodeRefs(dir string, root string, registered map[int]bool) []Finding {
	if _, err := os.Stat(root); err != nil {
		return nil
	}
	var findings []Finding
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
		file, relPath, err := parseGoFile(dir, path, parser.ParseComments)
		if err != nil {
			findings = append(findings, Finding{Rule: "L002", Message: err.Error(), Path: relPath})
			return nil
		}
		aliases := coreErrorsAliases(file)
		if len(aliases) == 0 {
			return nil
		}
		ast.Inspect(file, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			ident, ok := selector.X.(*ast.Ident)
			if !ok || !aliases[ident.Name] {
				return true
			}
			code, ok := coreErrorCodeValue(selector.Sel.Name)
			if !ok || registered[code] {
				return true
			}
			findings = append(findings, Finding{
				Rule:    "L002",
				Message: fmt.Sprintf("error code %s=%d is not registered in api/errors.yaml", selector.Sel.Name, code),
				Path:    relPath,
			})
			return true
		})
		return nil
	})
	return findings
}

func coreErrorsAliases(file *ast.File) map[string]bool {
	aliases := map[string]bool{}
	for _, spec := range file.Imports {
		if strings.Trim(spec.Path.Value, `"`) != "github.com/nucleuskit/core/errors" {
			continue
		}
		switch {
		case spec.Name == nil:
			aliases["errors"] = true
		case spec.Name.Name == ".":
			aliases[""] = true
		case spec.Name.Name != "_":
			aliases[spec.Name.Name] = true
		}
	}
	return aliases
}

func coreErrorCodeValue(name string) (int, bool) {
	switch name {
	case "CodeOK":
		return 0, true
	case "CodeInternal":
		return 1, true
	case "CodeInvalidArgument":
		return 2, true
	case "CodeNotFound":
		return 3, true
	case "CodeUnauthenticated":
		return 4, true
	case "CodePermissionDenied":
		return 5, true
	case "CodeDeadlineExceeded":
		return 6, true
	case "CodeUnavailable":
		return 7, true
	case "CodeFailedPrecondition":
		return 8, true
	default:
		return 0, false
	}
}
