// Package version carries the release identity the build stamps into the binaries.
//
// The values come from -ldflags at link time, computed by scripts/version.sh - the same rule
// scripts/publish-relhub.sh publishes with, so a client comparing itself against the update
// service can never disagree with it:
//
//	go build -ldflags "-X afvi/teach-imgui/internal/version.Name=0.1.12 \
//	                   -X afvi/teach-imgui/internal/version.CodeText=112"
//
// Both are strings because -X can only set string variables; Code() parses CodeText. An
// unstamped build (go run, go test, a developer's go build) keeps Name "dev" and code 0, and
// the update check stays off: a build that cannot name itself must not decide whether it is old.
package version

import (
	"fmt"
	"strconv"
)

// Name is the release name shown to people, e.g. "0.1.12". Set with -ldflags -X.
var Name = "dev"

// CodeText is the increasing integer the update service compares, e.g. "112", as stamped.
// It is text only because -X cannot set an int64; use Code.
var CodeText = ""

// Code is the version code, or 0 when this build was not stamped (or the stamp is not a
// number, which would be a build mistake rather than something to guess at).
func Code() int64 {
	n, err := strconv.ParseInt(CodeText, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// Known reports whether this build was stamped, i.e. whether it may take part in updates.
func Known() bool { return Code() > 0 && Name != "" && Name != "dev" }

// String is "0.1.12 (112)", or "dev" for an unstamped build.
func String() string {
	if !Known() {
		return "dev"
	}
	return fmt.Sprintf("%s (%d)", Name, Code())
}
