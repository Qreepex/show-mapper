// Package version holds build-time information stamped via -ldflags.
//
// Release builds inject values like:
//
//	go build -ldflags "\
//	  -X github.com/yourorg/showbridge/internal/version.Version=0.1.0 \
//	  -X github.com/yourorg/showbridge/internal/version.Commit=$(git rev-parse HEAD) \
//	  -X github.com/yourorg/showbridge/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package version

var (
	// Version is the semantic version without the leading "v" (e.g. "0.1.0"), or "dev".
	Version = "dev"
	// Commit is the full git commit SHA, or "none".
	Commit = "none"
	// Date is the UTC build date (RFC3339), or "unknown".
	Date = "unknown"
)
