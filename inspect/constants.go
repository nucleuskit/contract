package inspect

const (
	descriptionSchemaVersion = "1.0"
	flowGraphSchemaVersion   = "1.0"
)

const (
	goModFileName = "go.mod"
)

const (
	contractSourceAPI       = "api"
	contractPathOpenAPI     = "api/openapi.yaml"
	contractPathErrors      = "api/errors.yaml"
	contractProtoDir        = "api/proto"
	contractProtoFileSuffix = ".proto"
)

const (
	defaultPolicyOutboundKey = "outbound"
	providerConfigKey        = "provider"
)

const (
	commandValidate   = "nucleus validate --dir ."
	commandLintStrict = "nucleus lint --dir . --strict"
	commandVerifyJSON = "nucleus verify --dir . --json"
)
const (
	capabilitySQL   = "sql"
	capabilityMongo = "mongo"
)

const (
	moduleRoot       = "github.com/nucleuskit"
	moduleBridgeRoot = moduleRoot + "/bridge"
	moduleCapRoot    = moduleRoot + "/cap"
)
const (
	configDirName        = "configs" // Config directory
	configLocalNamePart  = "local"
	configSecretNamePart = "secret"
	configYAMLExtension  = ".yaml"
	configYMLExtension   = ".yml"
	configEnvPrefix      = "${"
	configEnvSuffix      = "}"
	configEnvFallbackSep = ":-"
	configKeySeparator   = "."
)
const (
	generatedHTTPAdapterTarget = "internal/adapter/http/gen"
	generatedGRPCAdapterTarget = "internal/adapter/grpc/gen"
	contractGeneratedTarget    = "contract/gen"
	runtimeHTTPServerSource    = "runtime/http/server.go"
	responseEnvelopeName       = "runtime/http.ResponseEnvelope"
)

const (
	flowNodeKindRoute      = "route"
	flowNodeKindHandler    = "handler"
	flowNodeKindOutbound   = "outbound"
	flowNodeKindDomain     = "domain"
	flowNodeKindResponse   = "response"
	flowNodeKindCapability = "capability"
)

const (
	flowEdgeKindDispatch        = "dispatch"
	flowEdgeKindOutboundUnknown = "outbound_unknown"
	flowEdgeKindCall            = "call"
	flowEdgeKindOutboundCall    = "outbound_call"
	flowEdgeKindEncodesResponse = "encodes_response"
	flowEdgeKindUsesCapability  = "uses_capability"
)

const (
	flowFactKindOpenAPIParameter   = "openapi_parameter"
	flowFactKindOpenAPIRequestBody = "openapi_request_body"
	flowFactKindErrorCode          = "error_code"
)

const (
	flowIDPrefixRoute      = "route:"
	flowIDPrefixOperation  = "operation:"
	flowIDPrefixDomain     = "domain:"
	flowIDPartResponse     = ":response:"
	flowIDPartCapability   = ":capability:"
	flowUnknownOutboundID  = ":outbound:unknown"
	flowHTTPClientOutbound = ":outbound:httpclient.Do"
)

const (
	flowParamPrefixPath   = "path."
	flowParamPrefixQuery  = "query."
	flowParamPrefixHeader = "header."
	flowRequestBodyName   = "body.required"
)

const (
	flowSourceOpenAPI           = contractPathOpenAPI
	flowSourceErrors            = contractPathErrors
	flowSourceStaticUnavailable = "static analysis unavailable"
	flowSourceConfidence        = "source"
	flowUnknownName             = "unknown"
	flowHTTPClientDoName        = "httpclient.Do"
	capabilityLog               = "log"
	generatedRouteBinderName    = "generated route binder dispatch"
	wellKnownNucleusPath        = "/.well-known/nucleus.json"
)

const (
	generatedFreshnessReasonMissingMarker = "missing freshness marker"
	generatedFreshnessReasonHashDiffers   = "source hash differs from generated marker"
	generatedFreshnessReasonInvalidTarget = "generated target must be a relative path inside the service directory"
)
const (
	goSourceExtension     = ".go"
	goTestSourceExtension = "_test.go"
	sourceLineSeparator   = ":"
	routeKeySeparator     = " "
	handlerNameSuffix     = " handler"
	domainSourcePathPart  = "internal/domain/"
)

const (
	selectorRegisterRoutes    = "RegisterRoutes"    // Register routes
	selectorHandle            = "Handle"            // Handle requests
	selectorRegisterWellKnown = "RegisterWellKnown" // Register Well-Known
	selectorHTTPClientDo      = "Do"                // Execute HTTP request
	selectorLogDebug          = "Debug"             // Debug log
	selectorLogInfo           = "Info"              // Info log
	selectorLogWarn           = "Warn"              // Warning log
	selectorLogError          = "Error"             // Error log

	identifierHTTP        = "http" // http package name
	identifierBlankImport = "_"    //
	identifierDotImport   = "."
)

const (
	selectorHTTPMethodGet    = "MethodGet"  // GET
	selectorHTTPMethodPost   = "MethodPost" //  POST
	selectorHTTPMethodPut    = "MethodPut"  // PUT
	selectorHTTPMethodPatch  = "MethodPatch"
	selectorHTTPMethodDelete = "MethodDelete"
)

const (
	httpMethodGet    = "GET"
	httpMethodPost   = "POST"
	httpMethodPut    = "PUT"
	httpMethodPatch  = "PATCH"
	httpMethodDelete = "DELETE"
)

const (
	skipDirGit         = ".git"
	skipDirGitNexus    = ".gitnexus"
	skipDirVendor      = "vendor"
	skipDirNodeModules = "node_modules"
	skipPathGit        = "/.git/"
	skipPathIdea       = "/.idea/"
	skipPathCursor     = "/.cursor/"
)
