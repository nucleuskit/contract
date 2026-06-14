package lint

import (
	"os"
	"path/filepath"
	"strings"
)

func lintTopLevelDirs(dir string) []Finding {
	allowed, ok := topLevelDirsFromAgents(dir)
	if !ok {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var findings []Finding
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() || strings.HasPrefix(name, ".") || allowed[name] {
			continue
		}
		findings = append(findings, Finding{Rule: "L011", Message: "top-level directory is not registered in AGENTS.md: " + name, Path: name})
	}
	return findings
}

func topLevelDirsFromAgents(dir string) (map[string]bool, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil {
		return nil, false
	}
	allowed := map[string]bool{}
	inSection := false
	for _, rawLine := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "## ") {
			heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
			inSection = heading == "Top-Level Directories" || heading == "顶层目录登记"
			continue
		}
		if !inSection {
			continue
		}
		for _, value := range backtickValues(line) {
			name := normalizeTopLevelDir(value)
			if name != "" {
				allowed[name] = true
			}
		}
	}
	if len(allowed) == 0 {
		return nil, false
	}
	return allowed, true
}

func backtickValues(line string) []string {
	var values []string
	for {
		start := strings.Index(line, "`")
		if start < 0 {
			return values
		}
		line = line[start+1:]
		end := strings.Index(line, "`")
		if end < 0 {
			return values
		}
		values = append(values, line[:end])
		line = line[end+1:]
	}
}

func normalizeTopLevelDir(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "./")
	value = strings.TrimSuffix(value, "/")
	if value == "" || strings.Contains(value, "*") || strings.Contains(value, string(filepath.Separator)) || strings.Contains(value, "/") {
		return ""
	}
	return value
}
