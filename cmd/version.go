package cmd

import "runtime/debug"

// devVersion is the placeholder main.version holds in a plain `go build`.
const devVersion = "dev"

// ResolveVersion picks the version to print. A release build passes the
// version GoReleaser injected with -ldflags; otherwise the module version
// recorded in the build info is used, so `go install …@v0.4.0` reports
// v0.4.0. Local builds report "dev".
func ResolveVersion(injected string, info *debug.BuildInfo) string {
	if injected != "" && injected != devVersion {
		return injected
	}
	if info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return devVersion
}
