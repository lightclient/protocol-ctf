// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	VersionMajor = 1          // Major version component of the current release
	VersionMinor = 11         // Minor version component of the current release
	VersionPatch = 0          // Patch version component of the current release
	VersionMeta  = "unstable" // Version metadata to append to the version string

	ourPath = "github.com/ethereum/go-ethereum" // Path to our module
)

// Version holds the textual version string.
var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

// ArchiveVersion holds the textual version string used for Geth archives. e.g.
// "1.8.11-dea1ce05" for stable releases, or "1.8.13-unstable-21c059b6" for unstable
// releases.
func ArchiveVersion(gitCommit string) string {
	vsn := Version
	if VersionMeta != "stable" {
		vsn += "-" + VersionMeta
	}
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	return vsn
}

func VersionWithCommit(gitCommit, gitDate string) string {
	vsn := VersionWithMeta
	if len(gitCommit) >= 8 {
		vsn += "-" + gitCommit[:8]
	}
	if (VersionMeta != "stable") && (gitDate != "") {
		vsn += "-" + gitDate
	}
	return vsn
}

// RuntimeInfo returns build and platform information about the current binary.
//
// If the package that is currently executing is a prefixed by our go-ethereum
// module path, it will print out commit and date VCS information. Otherwise,
// it will assume it's imported by a third-party and will return the imported
// version and whether it was replaced by another module.
func RuntimeInfo() string {
	var (
		version       = VersionWithMeta
		buildInfo, ok = debug.ReadBuildInfo()
	)

	switch {
	case !ok:
		// BuildInfo should generally always be set.
	case strings.HasPrefix(buildInfo.Path, ourPath):
		// If the main package is from our repo, we can actually
		// retrieve the VCS information directly from the buildInfo.
		revision, dirty, date := vcsInfo(buildInfo)
		version = fmt.Sprintf("geth %s", VersionWithCommit(revision, date))
		if dirty != "" {
			version = fmt.Sprintf("%s %s", version, dirty)
		}
	default:
		// Not our main package, probably imported by a different
		// project. VCS data less relevant here.
		mod := findModule(buildInfo, ourPath)
		version = mod.Version
		if mod.Replace != nil {
			version = fmt.Sprintf("%s (replaced by %s@%s)", version, mod.Replace.Path, mod.Replace.Version)
		}
	}
	return fmt.Sprintf("%s %s %s %s", version, runtime.Version(), runtime.GOARCH, runtime.GOOS)
}

// findModule returns the module at path.
func findModule(info *debug.BuildInfo, path string) *debug.Module {
	if info.Path == ourPath {
		return &info.Main
	}
	for _, mod := range info.Deps {
		if mod.Path == path {
			return mod
		}
	}
	return nil
}

// vcsInfo returns VCS information regarding the commit, dirty status, and date
// modified.
func vcsInfo(info *debug.BuildInfo) (string, string, string) {
	var (
		revision = "unknown"
		dirty    = ""
		date     = "unknown"
	)
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, v := range info.Settings {
			switch v.Key {
			case "vcs.revision":
				revision = v.Value
			case "vcs.modified":
				if v.Value == "true" {
					dirty = " (dirty)"
				}
			case "vcs.time":
				date = v.Value
			}
		}
	}
	return revision, dirty, date
}
