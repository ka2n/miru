package api

import (
	"runtime/debug"

	"github.com/samber/lo"
)

// Version and VersionCommit hold the version information
var (
	Version       = "dev"
	VersionCommit = ""
)

func init() {
	if i, ok := debug.ReadBuildInfo(); ok {
		Version = i.Main.Version
		if vcsv, ok := lo.Find(i.Settings, func(s debug.BuildSetting) bool {
			return s.Key == "vcs.revision"
		}); ok {
			VersionCommit = vcsv.Value
		}
	}
}
