package main

import (
	"fmt"
	"runtime/debug"
)

// SERVER_VERSION is the Texas Hold'em server release version.
// Bump when the wire protocol or gameplay behavior changes.
const SERVER_VERSION = "1.1.0"

// versionString returns the version plus git revision/build time when the
// binary was built inside a git checkout (go build embeds VCS info; go run
// may not, in which case only the version is shown)
func versionString() string {
	v := "texasholdem-server v" + SERVER_VERSION

	if info, ok := debug.ReadBuildInfo(); ok {
		rev, modified, btime := "", "", ""
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				rev = s.Value
			case "vcs.modified":
				if s.Value == "true" {
					modified = "+dirty"
				}
			case "vcs.time":
				btime = s.Value
			}
		}
		if rev != "" {
			if len(rev) > 8 {
				rev = rev[:8]
			}
			v += fmt.Sprintf(" (commit %s%s", rev, modified)
			if btime != "" {
				v += ", " + btime
			}
			v += ")"
		}
	}
	return v
}
