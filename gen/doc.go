// Package gen generates reproducible artifacts from Nucleus contract sources.
//
// The package reads OpenAPI, protobuf, and error catalog contracts and writes
// derived metadata, adapter stubs, client stubs, and freshness markers. Runtime
// wiring and CLI evidence rendering belong outside this package.
package gen
