package main

import (
	"runtime/debug"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveVersion_ShouldTrimVPrefix_WhenTaggedBuild(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.2.1"}}

	assert.Equal(t, "0.2.1", resolveVersion(info))
}

func TestResolveVersion_ShouldKeepPseudoVersion_WhenUntaggedBuild(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "v0.0.0-20260916033506-1ae8cc2dd766"}}

	assert.Equal(t, "0.0.0-20260916033506-1ae8cc2dd766", resolveVersion(info))
}

func TestResolveVersion_ShouldReturnDev_WhenDevelBuild(t *testing.T) {
	info := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}

	assert.Equal(t, "dev", resolveVersion(info))
}

func TestResolveVersion_ShouldReturnDev_WhenVersionEmpty(t *testing.T) {
	assert.Equal(t, "dev", resolveVersion(&debug.BuildInfo{}))
}

func TestResolveVersion_ShouldReturnDev_WhenBuildInfoUnavailable(t *testing.T) {
	assert.Equal(t, "dev", resolveVersion(nil))
}

func TestCurrentBuildInfo_ShouldReportAvailability(t *testing.T) {
	info, ok := currentBuildInfo()

	assert.True(t, ok, "тестовый бинарь всегда собирается с build info")
	assert.NotNil(t, info)
}
