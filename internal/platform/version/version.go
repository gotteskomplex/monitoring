// Package version holds build information injected via -ldflags.
package version

// Set at build time: -ldflags "-X github.com/gotteskomplex/monitoring/internal/platform/version.Version=..."
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

// ProtocolVersion is the satellite protocol version implemented by this build.
const ProtocolVersion = 1
