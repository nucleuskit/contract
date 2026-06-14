package lint

// Finding describes one lint rule violation.
type Finding struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

// Run executes the default Nucleus lint rule set for a service directory.
func Run(dir string) []Finding {
	return run(dir, false)
}

// RunStrict executes the default and strict Nucleus lint rule sets for a service directory.
func RunStrict(dir string) []Finding {
	return run(dir, true)
}

func run(dir string, strict bool) []Finding {
	var findings []Finding
	findings = append(findings, lintManifest(dir)...)
	findings = append(findings, lintCoreImports(dir)...)
	findings = append(findings, lintRuntimeBridgeImports(dir)...)
	findings = append(findings, lintTopLevelDirs(dir)...)
	if !strict {
		return sortFindings(findings)
	}
	findings = append(findings, lintRoutes(dir)...)
	findings = append(findings, lintErrorCodes(dir)...)
	findings = append(findings, lintDomainImports(dir)...)
	findings = append(findings, lintCapabilityGraph(dir)...)
	findings = append(findings, lintDependencies(dir)...)
	findings = append(findings, lintGRPCProto(dir)...)
	findings = append(findings, lintCriticalLegacyImports(dir)...)
	findings = append(findings, lintGeneratedFreshness(dir)...)
	findings = append(findings, lintSchemaVersions(dir)...)
	return sortFindings(findings)
}
