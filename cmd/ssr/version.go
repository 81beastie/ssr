package main

import (
	"runtime/debug"
	"strings"
)

const devVersion = "dev"

var version = resolveVersion(currentBuildInfo())

func currentBuildInfo() *debug.BuildInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil
	}
	return info
}

func resolveVersion(info *debug.BuildInfo) string {
	if info == nil {
		return devVersion
	}
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return devVersion
	}
	return strings.TrimPrefix(info.Main.Version, "v")
}
