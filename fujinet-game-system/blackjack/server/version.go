package main

import (
	"fmt"
	"runtime/debug"
)

// Bump when the wire protocol or gameplay behavior changes
const SERVER_VERSION = "0.1.0"

// Adds the git revision and build time when go build embedded them
func versionString() string {
	v := "blackjack-server v" + SERVER_VERSION

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
