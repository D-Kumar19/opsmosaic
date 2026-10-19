package brief

import "fmt"

// These values are set by release builds using -ldflags.
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func versionString() string {
	return fmt.Sprintf("opsmosaic %s (commit %s; built %s)", cleanText(Version), cleanText(Commit), cleanText(BuildTime))
}
