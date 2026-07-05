// Package version exposes build metadata injected via -ldflags -X at build time.
package version

import (
	"fmt"
	"runtime"
)

// These are overridden at build time:
//
//	go build -ldflags "-X github.com/Benny93/kafui/pkg/version.Version=v1.2.3 ..."
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)

// Info is the resolved build metadata plus the runtime environment.
type Info struct {
	Version   string
	Commit    string
	BuildTime string
	GoVersion string
	Platform  string
	Features  []string
}

// Get returns the current build info. features lists enabled feature flags
// (e.g. "dynamic-config", "mock") supplied by the caller.
func Get(features ...string) Info {
	return Info{
		Version:   Version,
		Commit:    ShortCommit(),
		BuildTime: BuildTime,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		Features:  features,
	}
}

// ShortCommit returns the commit truncated to 7 characters.
func ShortCommit() string {
	if len(Commit) > 7 {
		return Commit[:7]
	}
	return Commit
}

// String renders a one-line summary.
func (i Info) String() string {
	return fmt.Sprintf("kafui %s (commit %s, built %s, %s, %s)",
		i.Version, i.Commit, i.BuildTime, i.GoVersion, i.Platform)
}
