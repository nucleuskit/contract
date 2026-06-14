package lint

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
)

func normalizeServicePath(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" || filepath.IsAbs(value) || strings.HasPrefix(value, "/") || strings.Contains(value, "\\") {
		return "", false
	}
	cleaned := pathpkg.Clean(value)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func serviceFilePath(dir string, value string) (string, string, bool) {
	rel, ok := normalizeServicePath(value)
	if !ok {
		return "", "", false
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", "", false
	}
	target := filepath.Join(root, filepath.FromSlash(rel))
	targetRel, err := filepath.Rel(root, target)
	if err != nil || targetRel == ".." || strings.HasPrefix(targetRel, ".."+string(filepath.Separator)) {
		return "", "", false
	}
	return target, rel, true
}

func relativeFindingPath(dir string, filename string) string {
	if filename == "" {
		return ""
	}
	if !filepath.IsAbs(filename) {
		return filepath.ToSlash(filename)
	}
	root, err := filepath.Abs(dir)
	if err != nil {
		return filepath.Base(filename)
	}
	abs, err := filepath.Abs(filename)
	if err != nil {
		return filepath.Base(filename)
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return filepath.Base(filename)
	}
	return filepath.ToSlash(rel)
}

func parseGoFile(dir string, filename string, mode parser.Mode) (*ast.File, string, error) {
	rel := relativeFindingPath(dir, filename)
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, rel, fmt.Errorf("%s is not readable", rel)
	}
	file, err := parser.ParseFile(token.NewFileSet(), rel, data, mode)
	return file, rel, err
}

func safeLoaderError(err error, fallback string) string {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		return fallback
	}
	return err.Error()
}

func shouldSkipLintSourceDir(entry os.DirEntry) bool {
	name := entry.Name()
	return name == ".git" || name == "vendor" || name == "gen" || strings.HasPrefix(name, ".")
}

func isGoSourceFile(filename string) bool {
	return strings.HasSuffix(filename, ".go") && !strings.HasSuffix(filename, "_test.go")
}

func sortFindings(findings []Finding) []Finding {
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Rule != findings[j].Rule {
			return findings[i].Rule < findings[j].Rule
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Message < findings[j].Message
	})
	return findings
}
