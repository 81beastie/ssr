package clipboard_test

import (
	"testing"

	"github.com/81beastie/ssr/internal/clipboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_ShouldReturnX11Backend_WhenDisplaySetOnLinux(t *testing.T) {
	backend, err := clipboard.New(clipboard.Options{
		GOOS:    "linux",
		Display: ":0",
	})

	require.NoError(t, err)
	assert.IsType(t, &clipboard.X11{}, backend)
}

func TestNew_ShouldReturnWaylandBackend_WhenWaylandDisplaySetOnLinux(t *testing.T) {
	backend, err := clipboard.New(clipboard.Options{
		GOOS:           "linux",
		WaylandDisplay: "wayland-0",
		Display:        ":0",
	})

	require.NoError(t, err)
	assert.IsType(t, &clipboard.Wayland{}, backend)
}

func TestNew_ShouldReturnWin32Backend_WhenOSIsWindows(t *testing.T) {
	backend, err := clipboard.New(clipboard.Options{GOOS: "windows"})

	require.NoError(t, err)
	assert.IsType(t, &clipboard.Win32{}, backend)
}

func TestNew_ShouldReturnMacBackend_WhenOSIsDarwin(t *testing.T) {
	backend, err := clipboard.New(clipboard.Options{GOOS: "darwin"})

	require.NoError(t, err)
	assert.IsType(t, &clipboard.Mac{}, backend)
}

func TestNew_ShouldReturnOSC52Backend_WhenNoDisplayOnLinux(t *testing.T) {
	backend, err := clipboard.New(clipboard.Options{GOOS: "linux"})

	require.NoError(t, err)
	assert.IsType(t, &clipboard.OSC52{}, backend)
}

func TestNew_ShouldPreferOverride_WhenBackendNameProvided(t *testing.T) {
	backend, err := clipboard.New(clipboard.Options{
		GOOS:     "linux",
		Display:  ":0",
		Override: "osc52",
	})

	require.NoError(t, err)
	assert.IsType(t, &clipboard.OSC52{}, backend)
}

func TestNew_ShouldReturnError_WhenOverrideUnknown(t *testing.T) {
	_, err := clipboard.New(clipboard.Options{Override: "nonexistent"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown backend")
}
