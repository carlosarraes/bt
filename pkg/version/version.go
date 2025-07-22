package version

import "runtime/debug"

// Version information for the bt CLI
var (
	// Version is the current version of bt
	Version = "0.0.9"

	// Commit is the git commit hash
	Commit = "unknown"

	// Date is the build date
	Date = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = "unknown"
)

// BuildInfo contains build information
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Date      string `json:"date"`
	GoVersion string `json:"go_version"`
}

// GetBuildInfo returns the current build information
func GetBuildInfo() BuildInfo {
	goVersion := GoVersion
	if goVersion == "unknown" {
		if info, ok := debug.ReadBuildInfo(); ok {
			goVersion = info.GoVersion
		}
	}

	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: goVersion,
	}
}

// String returns a formatted version string
func (b BuildInfo) String() string {
	if b.Version == "dev" {
		return "bt dev (development build)"
	}
	return "bt " + b.Version
}

// GetVersion returns just the version string
func GetVersion() string {
	return Version
}
