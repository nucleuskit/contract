package proto

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/nucleuskit/contract/diagnostic"
)

const (
	protoDirName        = "api/proto"
	protoFileExtension  = ".proto"
	protoSourceBasePath = "api/proto"
)

// Service 服务
type Service struct {
	Package string   `json:"package,omitempty"` // 包名
	Name    string   `json:"name"`              // 服务名
	Source  string   `json:"source"`            // 源文件
	Methods []Method `json:"methods"`           // 方法
}

// Method 方法
type Method struct {
	Name            string     `json:"name"`                       // 方法名
	Request         string     `json:"request"`                    // 请求
	Response        string     `json:"response"`                   // 响应
	ClientStreaming bool       `json:"client_streaming,omitempty"` // 客户端流
	ServerStreaming bool       `json:"server_streaming,omitempty"` // 服务端流
	HTTPRules       []HTTPRule `json:"http_rules,omitempty"`       // HTTP 规则
}

// HTTPRule HTTP 规则
type HTTPRule struct {
	Method       string `json:"method"`                  // 请求方法
	Path         string `json:"path"`                    // 路径
	Body         string `json:"body,omitempty"`          // 请求体
	ResponseBody string `json:"response_body,omitempty"` // 响应体
}

// LoadServices 加载服务
func LoadServices(dir string) ([]Service, error) {
	protoDir := filepath.Join(dir, filepath.FromSlash(protoDirName))
	entries, err := os.ReadDir(protoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var services []Service
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), protoFileExtension) {
			continue
		}
		path := filepath.Join(protoDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		source := filepath.ToSlash(filepath.Join(protoSourceBasePath, entry.Name()))
		services = append(services, parseServices(source, string(data))...)
	}
	sort.Slice(services, func(i, j int) bool {
		if services[i].Source == services[j].Source {
			return services[i].Name < services[j].Name
		}
		return services[i].Source < services[j].Source
	})
	return services, nil
}

// ValidateDir checks api/proto/*.proto files with the package's lightweight parser.
func ValidateDir(dir string) diagnostic.Diagnostics {
	protoDir := filepath.Join(dir, filepath.FromSlash(protoDirName))
	entries, err := os.ReadDir(protoDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return diagnostic.Diagnostics{errorDiagnostic(protoDirName, "proto.read_failed", err.Error())}
	}

	var diagnostics diagnostic.Diagnostics
	seenServices := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), protoFileExtension) {
			continue
		}
		fullPath := filepath.Join(protoDir, entry.Name())
		data, err := os.ReadFile(fullPath)
		source := filepath.ToSlash(filepath.Join(protoSourceBasePath, entry.Name()))
		if err != nil {
			diagnostics = append(diagnostics, errorDiagnostic(source, "proto.read_failed", err.Error()))
			continue
		}
		diagnostics = append(diagnostics, validateProtoSource(source, string(data), seenServices)...)
	}
	return diagnostics
}

var (
	packagePattern          = regexp.MustCompile(`(?m)^\s*package\s+([A-Za-z0-9_.]+)\s*;`)                                                                                                              // 包名
	serviceHeaderPattern    = regexp.MustCompile(`service\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{`)                                                                                                             // 服务头
	methodPattern           = regexp.MustCompile(`rpc\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*(stream\s+)?([.A-Za-z_][.A-Za-z0-9_]*)\s*\)\s+returns\s*\(\s*(stream\s+)?([.A-Za-z_][.A-Za-z0-9_]*)\s*\)\s*;`)  // 方法
	methodBlockHeader       = regexp.MustCompile(`rpc\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(\s*(stream\s+)?([.A-Za-z_][.A-Za-z0-9_]*)\s*\)\s+returns\s*\(\s*(stream\s+)?([.A-Za-z_][.A-Za-z0-9_]*)\s*\)\s*\{`) // 方法块头
	httpRulePattern         = regexp.MustCompile(`(?s)option\s+\(google\.api\.http\)\s*=\s*\{(.*?)\}\s*;`)                                                                                              // HTTP 规则
	httpVerbPattern         = regexp.MustCompile(`(?m)\b(get|put|post|delete|patch)\s*:\s*"([^"]+)"`)                                                                                                   // HTTP 谓
	httpBodyPattern         = regexp.MustCompile(`(?m)\bbody\s*:\s*"([^"]*)"`)                                                                                                                          // 请求体
	httpResponseBodyPattern = regexp.MustCompile(`(?m)\bresponse_body\s*:\s*"([^"]*)"`)                                                                                                                 // 响应体
	additionalBindingStart  = regexp.MustCompile(`\badditional_bindings\s*\{`)                                                                                                                          // 添加绑定
	lineComment             = regexp.MustCompile(`(?m)//.*$`)                                                                                                                                           // 单行注释
)

// parseServices 解析服务
func parseServices(source string, data string) []Service {
	data = lineComment.ReplaceAllString(data, "")
	pkg := ""
	if match := packagePattern.FindStringSubmatch(data); len(match) == 2 {
		pkg = match[1]
	}

	var services []Service
	for _, serviceMatch := range parseServiceBlocks(data) {
		service := Service{
			Package: pkg,
			Name:    serviceMatch.Name,
			Source:  source,
		}
		service.Methods = parseMethods(serviceMatch.Body)
		services = append(services, service)
	}
	return services
}

func validateProtoSource(source string, data string, seenServices map[string]struct{}) diagnostic.Diagnostics {
	stripped := lineComment.ReplaceAllString(data, "")
	pkg := ""
	if match := packagePattern.FindStringSubmatch(stripped); len(match) == 2 {
		pkg = match[1]
	}

	var diagnostics diagnostic.Diagnostics
	for _, match := range serviceHeaderPattern.FindAllStringSubmatchIndex(stripped, -1) {
		bodyStart := match[1]
		bodyEnd := matchingBraceEnd(stripped, bodyStart-1)
		serviceName := stripped[match[2]:match[3]]
		if bodyEnd <= bodyStart {
			diagnostics = append(diagnostics, errorDiagnostic(source, "proto.service_block_unclosed", "service blocks must have a matching closing brace"))
			continue
		}
		serviceKey := pkg + "." + serviceName
		if _, ok := seenServices[serviceKey]; ok {
			diagnostics = append(diagnostics, errorDiagnostic(source, "proto.service_duplicate", "service names must be unique within a package"))
		}
		seenServices[serviceKey] = struct{}{}
		body := stripped[bodyStart:bodyEnd]
		diagnostics = append(diagnostics, validateRPCDeclarations(source, body)...)
		if strings.Contains(body, "google.api.http") && len(parseHTTPRules(body)) == 0 {
			diagnostics = append(diagnostics, errorDiagnostic(source, "proto.http_rule_invalid", "google.api.http options must include a supported HTTP verb"))
		}
	}
	return diagnostics
}

func validateRPCDeclarations(source string, body string) diagnostic.Diagnostics {
	var diagnostics diagnostic.Diagnostics
	blockRanges := serviceMethodBlockRanges(body)
	rpcPattern := regexp.MustCompile(`\brpc\s+([A-Za-z_][A-Za-z0-9_]*)`)
	for _, match := range rpcPattern.FindAllStringIndex(body, -1) {
		if validRPCAt(body, match[0]) || withinAny(match[0], blockRanges) {
			continue
		}
		diagnostics = append(diagnostics, errorDiagnostic(source, "proto.rpc_invalid", "rpc declarations must include request and response types"))
	}
	return diagnostics
}

func serviceMethodBlockRanges(body string) [][2]int {
	var blockRanges [][2]int
	for _, match := range methodBlockHeader.FindAllStringSubmatchIndex(body, -1) {
		bodyStart := match[1]
		bodyEnd := matchingBraceEnd(body, bodyStart-1)
		if bodyEnd > bodyStart {
			blockRanges = append(blockRanges, [2]int{match[0], bodyEnd + 1})
		}
	}
	return blockRanges
}

func validRPCAt(body string, position int) bool {
	for _, match := range methodPattern.FindAllStringIndex(body, -1) {
		if match[0] == position {
			return true
		}
	}
	for _, match := range methodBlockHeader.FindAllStringIndex(body, -1) {
		if match[0] == position {
			return true
		}
	}
	return false
}

// methodDecl 方法声明
type serviceBlock struct {
	Name string // 服务名
	Body string // 服务块内容
}

// parseServiceBlocks 解析服务块
func parseServiceBlocks(data string) []serviceBlock {
	matches := serviceHeaderPattern.FindAllStringSubmatchIndex(data, -1)
	blocks := make([]serviceBlock, 0, len(matches))
	for _, match := range matches {
		bodyStart := match[1]
		bodyEnd := matchingBraceEnd(data, bodyStart-1)
		if bodyEnd <= bodyStart {
			continue
		}
		blocks = append(blocks, serviceBlock{
			Name: data[match[2]:match[3]],
			Body: data[bodyStart:bodyEnd],
		})
	}
	return blocks
}

// matchingBraceEnd 匹配括号
func matchingBraceEnd(data string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(data) || data[openIndex] != '{' {
		return -1
	}
	depth := 0
	for i := openIndex; i < len(data); i++ {
		switch data[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// Method 方法
type methodDecl struct {
	Position int    // 位置
	Method   Method // 方法
}

// methodDecl 方法声明
func parseMethods(body string) []Method {
	var decls []methodDecl
	var blockRanges [][2]int
	for _, match := range methodBlockHeader.FindAllStringSubmatchIndex(body, -1) {
		bodyStart := match[1]
		bodyEnd := matchingBraceEnd(body, bodyStart-1)
		if bodyEnd <= bodyStart {
			continue
		}
		decls = append(decls, methodDecl{
			Position: match[0],
			Method: Method{
				Name:            body[match[2]:match[3]],
				Request:         body[match[6]:match[7]],
				Response:        body[match[10]:match[11]],
				ClientStreaming: strings.TrimSpace(matchString(body, match, 4, 5)) == "stream",
				ServerStreaming: strings.TrimSpace(matchString(body, match, 8, 9)) == "stream",
				HTTPRules:       parseHTTPRules(body[bodyStart:bodyEnd]),
			},
		})
		blockRanges = append(blockRanges, [2]int{match[0], bodyEnd + 1})
	}
	for _, match := range methodPattern.FindAllStringSubmatchIndex(body, -1) {
		if withinAny(match[0], blockRanges) {
			continue
		}
		decls = append(decls, methodDecl{
			Position: match[0],
			Method: Method{
				Name:            body[match[2]:match[3]],
				Request:         body[match[6]:match[7]],
				Response:        body[match[10]:match[11]],
				ClientStreaming: strings.TrimSpace(matchString(body, match, 4, 5)) == "stream",
				ServerStreaming: strings.TrimSpace(matchString(body, match, 8, 9)) == "stream",
			},
		})
	}
	sort.Slice(decls, func(i, j int) bool {
		return decls[i].Position < decls[j].Position
	})
	methods := make([]Method, len(decls))
	for i, decl := range decls {
		methods[i] = decl.Method
	}
	return methods
}
func matchString(data string, indexes []int, startIndex int, endIndex int) string {
	if startIndex >= len(indexes) || endIndex >= len(indexes) {
		return ""
	}
	start := indexes[startIndex]
	end := indexes[endIndex]
	if start < 0 || end < 0 || start > end {
		return ""
	}
	return data[start:end]
}
func withinAny(position int, ranges [][2]int) bool {
	for _, item := range ranges {
		if position >= item[0] && position < item[1] {
			return true
		}
	}
	return false
}
func parseHTTPRules(body string) []HTTPRule {
	rules := []HTTPRule{}
	for _, ruleMatch := range httpRulePattern.FindAllStringSubmatch(body, -1) {
		rules = append(rules, parseHTTPRuleBodies(ruleMatch[1])...)
	}
	if len(rules) == 0 && strings.Contains(body, "google.api.http") {
		rules = append(rules, parseHTTPRuleBodies(body)...)
	}
	return rules
}
func parseHTTPRuleBodies(ruleBody string) []HTTPRule {
	bindings := extractAdditionalBindings(ruleBody)
	primaryBody := blankRanges(ruleBody, bindings)
	rules := []HTTPRule{}
	if rule, ok := parseHTTPRuleBody(primaryBody); ok {
		rules = append(rules, rule)
	}
	for _, binding := range bindings {
		rules = append(rules, parseHTTPRuleBodies(binding.Body)...)
	}
	return rules
}

type bodyRange struct {
	Start int
	End   int
	Body  string
}

func extractAdditionalBindings(data string) []bodyRange {
	matches := additionalBindingStart.FindAllStringIndex(data, -1)
	ranges := []bodyRange{}
	lastEnd := -1
	for _, match := range matches {
		if match[0] < lastEnd {
			continue
		}
		openIdx := match[1] - 1
		closeIdx := matchingBraceEnd(data, openIdx)
		if closeIdx <= openIdx {
			continue
		}
		ranges = append(ranges, bodyRange{
			Start: match[0],
			End:   closeIdx + 1,
			Body:  data[openIdx+1 : closeIdx],
		})
		lastEnd = closeIdx + 1
	}
	return ranges
}
func blankRanges(data string, ranges []bodyRange) string {
	if len(ranges) == 0 {
		return data
	}
	buffer := []byte(data)
	for _, item := range ranges {
		for i := item.Start; i < item.End && i < len(buffer); i++ {
			buffer[i] = ' '
		}
	}
	return string(buffer)
}
func parseHTTPRuleBody(ruleBody string) (HTTPRule, bool) {
	verbMatch := httpVerbPattern.FindStringSubmatch(ruleBody)
	if len(verbMatch) != 3 {
		return HTTPRule{}, false
	}
	rule := HTTPRule{
		Method: strings.ToUpper(verbMatch[1]),
		Path:   verbMatch[2],
	}
	if bodyMatch := httpBodyPattern.FindStringSubmatch(ruleBody); len(bodyMatch) == 2 {
		rule.Body = bodyMatch[1]
	}
	if responseBodyMatch := httpResponseBodyPattern.FindStringSubmatch(ruleBody); len(responseBodyMatch) == 2 {
		rule.ResponseBody = responseBodyMatch[1]
	}
	return rule, true
}

func errorDiagnostic(path string, code string, message string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity: diagnostic.SeverityError,
		Code:     code,
		Path:     path,
		Message:  message,
	}
}
