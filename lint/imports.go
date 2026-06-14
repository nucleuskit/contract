package lint

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"

	"github.com/nucleuskit/contract/manifest"
)

func lintDomainImports(dir string) []Finding {
	return lintImports(dir, filepath.Join(dir, "internal", "domain"), func(path string) (string, bool) {
		switch {
		case path == "github.com/gin-gonic/gin":
			return "domain must not import gin: " + path, true
		case strings.HasPrefix(path, "google.golang.org/grpc"):
			return "domain must not import grpc: " + path, true
		case strings.Contains(path, "/bridge/") || strings.HasSuffix(path, "/bridge"):
			return "domain must not import bridge: " + path, true
		case strings.Contains(path, "/runtime/") || strings.HasSuffix(path, "/runtime"):
			return "domain must not import runtime: " + path, true
		default:
			return "", false
		}
	}, "L003")
}

func lintCriticalLegacyImports(dir string) []Finding {
	m, err := manifest.Load(dir)
	if err != nil || m.Service.Tier != "critical" {
		return nil
	}
	return lintImports(dir, dir, func(path string) (string, bool) {
		if strings.Contains(path, "/bridge/legacy/") {
			return "critical service must not import legacy bridge: " + path, true
		}
		return "", false
	}, "L007")
}

func lintCoreImports(dir string) []Finding {
	return lintImports(dir, filepath.Join(dir, "core"), func(path string) (string, bool) {
		if isNonStandardImport(path) {
			return "core only allows standard library imports: " + path, true
		}
		return "", false
	}, "L008")
}

func lintRuntimeBridgeImports(dir string) []Finding {
	return lintImports(dir, filepath.Join(dir, "runtime"), func(path string) (string, bool) {
		if strings.Contains(path, "/bridge/") || strings.HasSuffix(path, "/bridge") {
			return "runtime must not import bridge: " + path, true
		}
		return "", false
	}, "L009")
}

func lintImports(dir string, root string, check func(string) (string, bool), rule string) []Finding {
	var findings []Finding
	if _, err := os.Stat(root); err != nil {
		return findings
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
		file, relPath, err := parseGoFile(dir, path, parser.ImportsOnly)
		if err != nil {
			findings = append(findings, Finding{Rule: rule, Message: err.Error(), Path: relPath})
			return nil
		}
		for _, spec := range file.Imports {
			importPath := strings.Trim(spec.Path.Value, `"`)
			if message, ok := check(importPath); ok {
				findings = append(findings, Finding{Rule: rule, Message: message, Path: relPath})
			}
		}
		return nil
	})
	return findings
}

func isNonStandardImport(path string) bool {
	return strings.Contains(path, ".")
}

// UsesImport reports whether a parsed Go file imports the provided package path.
func UsesImport(file *ast.File, importPath string) bool {
	for _, spec := range file.Imports {
		if strings.Trim(spec.Path.Value, `"`) == importPath {
			return true
		}
	}
	return false
}
