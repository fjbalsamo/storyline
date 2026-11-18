// Package templates holds the canonical /story command definitions, written
// once in a tool-agnostic format. Adapters render them for each AI tool.
package templates

import "embed"

//go:embed commands
var FS embed.FS
