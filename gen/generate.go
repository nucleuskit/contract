package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	contracterrors "github.com/nucleuskit/contract/errors"
	"github.com/nucleuskit/contract/inspect"
	"github.com/nucleuskit/contract/openapi"
	contractproto "github.com/nucleuskit/contract/proto"
)

// Result describes files written by a generation run.
type Result struct {
	Files []string
	Hash  string
}

// Options selects which contract artifacts are generated.
type Options struct {
	HTTP    bool
	GRPC    bool
	Errors  bool
	Clients bool
}

// Generate writes the default Nucleus contract artifacts for dir.
func Generate(dir string) (Result, error) {
	return GenerateWithOptions(dir, Options{})
}

// GenerateWithOptions writes selected Nucleus contract artifacts for dir.
//
// When no option is selected, HTTP, gRPC, and error metadata are generated.
func GenerateWithOptions(dir string, options Options) (Result, error) {
	selected := options.HTTP || options.GRPC || options.Errors || options.Clients
	if !selected {
		options.HTTP = true
		options.GRPC = true
		options.Errors = true
	}

	contractTargetDir := filepath.Join(dir, "contract", "gen")
	httpAdapterTargetDir := filepath.Join(dir, "internal", "adapter", "http", "gen")
	if err := os.MkdirAll(contractTargetDir, 0o755); err != nil {
		return Result{}, err
	}

	hash, err := inspect.ContractSourceHash(dir)
	if err != nil {
		return Result{}, err
	}

	var files []string
	if options.Errors {
		errorCodes, err := contracterrors.Load(dir)
		if err != nil {
			return Result{}, err
		}
		if err := validateUniqueErrorNames(errorCodes); err != nil {
			return Result{}, err
		}
		if path, err := writeGo(filepath.Join(contractTargetDir, "errors.go"), renderErrors(errorCodes)); err != nil {
			return Result{}, err
		} else {
			files = append(files, path)
		}
	}
	if options.HTTP {
		endpoints, err := openapi.LoadEndpoints(dir)
		if err != nil {
			return Result{}, err
		}
		routes, err := openapi.LoadRouteRegistry(dir)
		if err != nil {
			return Result{}, err
		}
		if err := validateUniqueRouteMethodNames(routes); err != nil {
			return Result{}, err
		}
		if path, err := writeGo(filepath.Join(contractTargetDir, "endpoints.go"), renderEndpoints(endpoints)); err != nil {
			return Result{}, err
		} else {
			files = append(files, path)
		}
		if path, err := writeGo(filepath.Join(httpAdapterTargetDir, "handler.gen.go"), renderHTTPHandler(routes)); err != nil {
			return Result{}, err
		} else {
			files = append(files, path)
		}
		if path, err := writeGo(filepath.Join(httpAdapterTargetDir, "types.gen.go"), renderHTTPRouteTypes(routes)); err != nil {
			return Result{}, err
		} else {
			files = append(files, path)
		}
		if path, err := writeGo(filepath.Join(httpAdapterTargetDir, "routes.gen.go"), renderHTTPRouteBinder(routes)); err != nil {
			return Result{}, err
		} else {
			files = append(files, path)
		}
		if err := writeFreshnessMarker(filepath.Join(httpAdapterTargetDir, inspect.FreshnessMarker), hash); err != nil {
			return Result{}, err
		}
		files = append(files, filepath.Join(httpAdapterTargetDir, inspect.FreshnessMarker))
	}
	if options.Clients {
		clientTargetDir := filepath.Join(dir, "sdk", "go")
		routes, err := openapi.LoadRouteRegistry(dir)
		if err != nil {
			return Result{}, err
		}
		if err := validateUniqueClientMethodNames(routes); err != nil {
			return Result{}, err
		}
		if path, err := writeGo(filepath.Join(clientTargetDir, "client.gen.go"), renderHTTPClient(routes)); err != nil {
			return Result{}, err
		} else {
			files = append(files, path)
		}
		if err := writeFreshnessMarker(filepath.Join(clientTargetDir, inspect.FreshnessMarker), hash); err != nil {
			return Result{}, err
		}
		files = append(files, filepath.Join(clientTargetDir, inspect.FreshnessMarker))
	}
	if options.GRPC {
		grpcServices, err := contractproto.LoadServices(dir)
		if err != nil {
			return Result{}, err
		}
		if len(grpcServices) > 0 || selected {
			if path, err := writeGo(filepath.Join(contractTargetDir, "grpc.go"), renderGRPCServices(grpcServices)); err != nil {
				return Result{}, err
			} else {
				files = append(files, path)
			}
		}
	}
	sources, err := loadContractSources(dir)
	if err != nil {
		return Result{}, err
	}
	if path, err := writeGo(filepath.Join(contractTargetDir, "contract_source.go"), renderContractSources(hash, sources)); err != nil {
		return Result{}, err
	} else {
		files = append(files, path)
	}
	marker := filepath.Join(contractTargetDir, inspect.FreshnessMarker)
	if err := writeFreshnessMarker(marker, hash); err != nil {
		return Result{}, err
	}
	files = append(files, marker)

	return Result{Files: files, Hash: hash}, nil
}

func renderErrors(codes []contracterrors.Code) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("// Code is a generated Nucleus error code.\n")
	buffer.WriteString("type Code int\n\n")
	buffer.WriteString("const (\n")
	for _, code := range codes {
		fmt.Fprintf(&buffer, "\t%s Code = %d\n", errorConstName(code), code.Code)
	}
	buffer.WriteString(")\n\n")
	buffer.WriteString("// ErrorMessages maps generated error codes to stable messages.\n")
	buffer.WriteString("var ErrorMessages = map[Code]string{\n")
	for _, code := range codes {
		fmt.Fprintf(&buffer, "\t%s: %q,\n", errorConstName(code), code.Message)
	}
	buffer.WriteString("}\n\n")
	buffer.WriteString("// HTTPStatuses maps generated error codes to HTTP statuses.\n")
	buffer.WriteString("var HTTPStatuses = map[Code]int{\n")
	for _, code := range codes {
		fmt.Fprintf(&buffer, "\t%s: %d,\n", errorConstName(code), code.HTTPStatus)
	}
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func renderEndpoints(endpoints []openapi.Endpoint) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("// Endpoint is generated from OpenAPI.\n")
	buffer.WriteString("type Endpoint struct {\n")
	buffer.WriteString("\tMethod string\n")
	buffer.WriteString("\tPath string\n")
	buffer.WriteString("\tOperationID string\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// Endpoints lists generated HTTP endpoint metadata.\n")
	buffer.WriteString("var Endpoints = []Endpoint{\n")
	for _, endpoint := range endpoints {
		fmt.Fprintf(&buffer, "\t{Method: %q, Path: %q, OperationID: %q},\n", endpoint.Method, endpoint.Path, endpoint.OperationID)
	}
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func renderHTTPHandler(routes []openapi.Route) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	if len(routes) > 0 {
		buffer.WriteString("import \"net/http\"\n\n")
	}
	buffer.WriteString("// Handler is implemented by the handwritten HTTP adapter.\n")
	buffer.WriteString("type Handler interface {\n")
	for _, route := range routes {
		methodName := routeMethodName(route)
		fmt.Fprintf(&buffer, "\t// %s handles the %s operation.\n", methodName, operationLabel(route))
		fmt.Fprintf(&buffer, "\t%s(request *http.Request) (any, error)\n", methodName)
	}
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func renderHTTPRouteTypes(routes []openapi.Route) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("// RouteParameter is generated OpenAPI route parameter metadata.\n")
	buffer.WriteString("type RouteParameter struct {\n")
	buffer.WriteString("\tName string\n")
	buffer.WriteString("\tIn string\n")
	buffer.WriteString("\tRequired bool\n")
	buffer.WriteString("\tSchemaType string\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// ServerRoute is generated HTTP route metadata for adapter registration.\n")
	buffer.WriteString("type ServerRoute struct {\n")
	buffer.WriteString("\tMethod string\n")
	buffer.WriteString("\tPath string\n")
	buffer.WriteString("\tOperationID string\n")
	buffer.WriteString("\tHandlerName string\n")
	buffer.WriteString("\tParameters []RouteParameter\n")
	buffer.WriteString("\tRequestBodyRequired bool\n")
	buffer.WriteString("\tPriority int\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// ServerRoutes lists generated HTTP server route metadata.\n")
	buffer.WriteString("var ServerRoutes = []ServerRoute{\n")
	for _, route := range routes {
		fmt.Fprintf(
			&buffer,
			"\t{Method: %q, Path: %q, OperationID: %q, HandlerName: %q, Parameters: []RouteParameter{",
			route.Method,
			route.Path,
			route.OperationID,
			routeMethodName(route),
		)
		for _, parameter := range route.Parameters {
			fmt.Fprintf(&buffer, "{Name: %q, In: %q, Required: %t, SchemaType: %q},", parameter.Name, parameter.In, parameter.Required, parameter.SchemaType)
		}
		fmt.Fprintf(&buffer, "}, RequestBodyRequired: %t, Priority: %d},\n", route.RequestBodyRequired, route.Priority)
	}
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func renderHTTPRouteBinder(routes []openapi.Route) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	if len(routes) > 0 {
		buffer.WriteString("import (\n")
		buffer.WriteString("\t\"net/http\"\n\n")
		buffer.WriteString("\truntimehttp \"github.com/nucleuskit/http\"\n")
		buffer.WriteString(")\n\n")
	} else {
		buffer.WriteString("import runtimehttp \"github.com/nucleuskit/http\"\n\n")
	}
	buffer.WriteString("// RegisterRoutes binds generated OpenAPI routes to a handwritten handler.\n")
	buffer.WriteString("func RegisterRoutes(server *runtimehttp.Server, handler Handler) {\n")
	buffer.WriteString("\tif server == nil || handler == nil {\n")
	buffer.WriteString("\t\treturn\n")
	buffer.WriteString("\t}\n")
	buffer.WriteString("\tserver.RegisterRoutes([]runtimehttp.Route{\n")
	for _, route := range routes {
		methodName := routeMethodName(route)
		fmt.Fprintf(&buffer, "\t\t{Method: %q, Path: %q, OperationID: %q, Handler: func(request *http.Request) (any, error) {\n", route.Method, route.Path, route.OperationID)
		fmt.Fprintf(&buffer, "\t\t\treturn handler.%s(request)\n", methodName)
		buffer.WriteString("\t\t}},\n")
	}
	buffer.WriteString("\t})\n")
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func renderHTTPClient(routes []openapi.Route) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("import (\n")
	buffer.WriteString("\t\"context\"\n")
	buffer.WriteString("\t\"io\"\n")
	buffer.WriteString("\t\"net/http\"\n")
	buffer.WriteString("\t\"strings\"\n")
	buffer.WriteString(")\n\n")
	buffer.WriteString("// Client is a generated minimal HTTP client stub.\n")
	buffer.WriteString("type Client struct {\n")
	buffer.WriteString("\tBaseURL string\n")
	buffer.WriteString("\tHTTPClient *http.Client\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// NewClient creates a generated HTTP client.\n")
	buffer.WriteString("func NewClient(baseURL string, client *http.Client) *Client {\n")
	buffer.WriteString("\tif client == nil {\n")
	buffer.WriteString("\t\tclient = http.DefaultClient\n")
	buffer.WriteString("\t}\n")
	buffer.WriteString("\treturn &Client{BaseURL: strings.TrimRight(baseURL, \"/\"), HTTPClient: client}\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// ClientOperation is generated HTTP client operation metadata.\n")
	buffer.WriteString("type ClientOperation struct {\n")
	buffer.WriteString("\tMethod string\n")
	buffer.WriteString("\tPath string\n")
	buffer.WriteString("\tOperationID string\n")
	buffer.WriteString("\tMethodName string\n")
	buffer.WriteString("\tRequestBodyRequired bool\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// ClientOperations lists generated HTTP client operation metadata.\n")
	buffer.WriteString("var ClientOperations = []ClientOperation{\n")
	for _, route := range routes {
		fmt.Fprintf(
			&buffer,
			"\t{Method: %q, Path: %q, OperationID: %q, MethodName: %q, RequestBodyRequired: %t},\n",
			route.Method,
			route.Path,
			route.OperationID,
			routeMethodName(route),
			route.RequestBodyRequired,
		)
	}
	buffer.WriteString("}\n\n")
	for _, route := range routes {
		methodName := routeMethodName(route)
		fmt.Fprintf(&buffer, "// %s calls the %s operation.\n", methodName, operationLabel(route))
		fmt.Fprintf(&buffer, "func (c *Client) %s(ctx context.Context, body io.Reader) (*http.Response, error) {\n", methodName)
		fmt.Fprintf(&buffer, "\treturn c.do(ctx, %q, %q, body)\n", route.Method, route.Path)
		buffer.WriteString("}\n\n")
	}
	buffer.WriteString("func (c *Client) do(ctx context.Context, method string, path string, body io.Reader) (*http.Response, error) {\n")
	buffer.WriteString("\tclient := c.HTTPClient\n")
	buffer.WriteString("\tif client == nil {\n")
	buffer.WriteString("\t\tclient = http.DefaultClient\n")
	buffer.WriteString("\t}\n")
	buffer.WriteString("\treq, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(c.BaseURL, \"/\")+path, body)\n")
	buffer.WriteString("\tif err != nil {\n")
	buffer.WriteString("\t\treturn nil, err\n")
	buffer.WriteString("\t}\n")
	buffer.WriteString("\treturn client.Do(req)\n")
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

type contractSource struct {
	Path    string
	Content string
}

func renderContractSources(hash string, sources []contractSource) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("// ContractSourceHash is generated from contract source files.\n")
	fmt.Fprintf(&buffer, "const ContractSourceHash = %q\n\n", hash)
	buffer.WriteString("// EmbeddedContractSource is a generated contract source snapshot.\n")
	buffer.WriteString("type EmbeddedContractSource struct {\n")
	buffer.WriteString("\tPath string\n")
	buffer.WriteString("\tContent string\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// EmbeddedContractSources contains generated contract source snapshots.\n")
	buffer.WriteString("var EmbeddedContractSources = []EmbeddedContractSource{\n")
	for _, source := range sources {
		fmt.Fprintf(&buffer, "\t{Path: %q, Content: %q},\n", source.Path, source.Content)
	}
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func renderGRPCServices(services []contractproto.Service) []byte {
	var buffer bytes.Buffer
	buffer.WriteString("package gen\n\n")
	buffer.WriteString("// GRPCService is generated from protobuf service definitions.\n")
	buffer.WriteString("type GRPCService struct {\n")
	buffer.WriteString("\tPackage string\n")
	buffer.WriteString("\tName string\n")
	buffer.WriteString("\tSource string\n")
	buffer.WriteString("\tMethods []GRPCMethod\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// GRPCMethod is generated from protobuf rpc definitions.\n")
	buffer.WriteString("type GRPCMethod struct {\n")
	buffer.WriteString("\tName string\n")
	buffer.WriteString("\tRequest string\n")
	buffer.WriteString("\tResponse string\n")
	buffer.WriteString("\tClientStreaming bool\n")
	buffer.WriteString("\tServerStreaming bool\n")
	buffer.WriteString("\tHTTPRules []GRPCHTTPRule\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// GRPCHTTPRule is generated from google.api.http annotations.\n")
	buffer.WriteString("type GRPCHTTPRule struct {\n")
	buffer.WriteString("\tMethod string\n")
	buffer.WriteString("\tPath string\n")
	buffer.WriteString("\tBody string\n")
	buffer.WriteString("\tResponseBody string\n")
	buffer.WriteString("}\n\n")
	buffer.WriteString("// GRPCServices lists generated gRPC service metadata.\n")
	buffer.WriteString("var GRPCServices = []GRPCService{\n")
	for _, service := range services {
		fmt.Fprintf(&buffer, "\t{Package: %q, Name: %q, Source: %q, Methods: []GRPCMethod{\n", service.Package, service.Name, service.Source)
		for _, method := range service.Methods {
			fmt.Fprintf(
				&buffer,
				"\t\t{Name: %q, Request: %q, Response: %q, ClientStreaming: %t, ServerStreaming: %t, HTTPRules: []GRPCHTTPRule{",
				method.Name,
				method.Request,
				method.Response,
				method.ClientStreaming,
				method.ServerStreaming,
			)
			for _, rule := range method.HTTPRules {
				fmt.Fprintf(&buffer, "{Method: %q, Path: %q, Body: %q, ResponseBody: %q},", rule.Method, rule.Path, rule.Body, rule.ResponseBody)
			}
			buffer.WriteString("}},\n")
		}
		buffer.WriteString("\t}},\n")
	}
	buffer.WriteString("}\n")
	return buffer.Bytes()
}

func writeGo(path string, data []byte) (string, error) {
	formatted, err := format.Source(data)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func writeFreshnessMarker(path string, hash string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(hash+"\n"), 0o644)
}

func loadContractSources(dir string) ([]contractSource, error) {
	paths := []string{
		filepath.Join(dir, "api", "openapi.yaml"),
		filepath.Join(dir, "api", "errors.yaml"),
	}
	protoDir := filepath.Join(dir, "api", "proto")
	if entries, err := os.ReadDir(protoDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".proto") {
				paths = append(paths, filepath.Join(protoDir, entry.Name()))
			}
		}
	}
	sort.Strings(paths)

	sources := make([]contractSource, 0, len(paths))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return nil, err
		}
		sources = append(sources, contractSource{
			Path:    filepath.ToSlash(relative),
			Content: string(data),
		})
	}
	return sources, nil
}

func endpointMethodName(endpoint openapi.Endpoint) string {
	name := endpoint.OperationID
	if name == "" {
		name = endpoint.Method + " " + endpoint.Path
	}
	return methodName(name)
}

func routeMethodName(route openapi.Route) string {
	name := route.OperationID
	if name == "" {
		name = route.Method + " " + route.Path
	}
	return methodName(name)
}

func operationLabel(route openapi.Route) string {
	if strings.TrimSpace(route.OperationID) != "" {
		return route.OperationID
	}
	return strings.TrimSpace(route.Method + " " + route.Path)
}

func validateUniqueRouteMethodNames(routes []openapi.Route) error {
	seen := map[string]openapi.Route{}
	for _, route := range routes {
		name := routeMethodName(route)
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("duplicate generated handler name %q for %s and %s", name, operationLabel(previous), operationLabel(route))
		}
		seen[name] = route
	}
	return nil
}

func validateUniqueClientMethodNames(routes []openapi.Route) error {
	seen := map[string]openapi.Route{}
	for _, route := range routes {
		name := routeMethodName(route)
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("duplicate generated client method name %q for %s and %s", name, operationLabel(previous), operationLabel(route))
		}
		seen[name] = route
	}
	return nil
}

func validateUniqueErrorNames(codes []contracterrors.Code) error {
	seen := map[string]contracterrors.Code{}
	for _, code := range codes {
		name := errorConstName(code)
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("duplicate generated error code name %q for codes %d and %d", name, previous.Code, code.Code)
		}
		seen[name] = code
	}
	return nil
}

func methodName(name string) string {
	parts := splitMethodIdentifier(name)
	if len(parts) == 0 {
		return "Handle"
	}
	joined := strings.Join(parts, "")
	if joined[0] >= '0' && joined[0] <= '9' {
		return "Operation" + joined
	}
	return joined
}

func splitMethodIdentifier(value string) []string {
	raw := nonIdentifier.Split(value, -1)
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		if item == "" {
			continue
		}
		parts = append(parts, strings.ToUpper(item[:1])+item[1:])
	}
	return parts
}

func errorConstName(code contracterrors.Code) string {
	if code.Code == 0 {
		return "CodeOK"
	}
	parts := splitIdentifier(code.Message)
	if len(parts) == 0 {
		return "Code" + strconv.Itoa(code.Code)
	}
	return "Code" + strings.Join(parts, "")
}

var nonIdentifier = regexp.MustCompile(`[^A-Za-z0-9]+`)

func splitIdentifier(value string) []string {
	raw := nonIdentifier.Split(value, -1)
	parts := make([]string, 0, len(raw))
	for _, item := range raw {
		if item == "" {
			continue
		}
		parts = append(parts, strings.ToUpper(item[:1])+strings.ToLower(item[1:]))
	}
	return parts
}
