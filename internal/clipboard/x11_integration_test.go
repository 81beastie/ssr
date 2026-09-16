package clipboard_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/81beastie/ssr/internal/clipboard"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gclip "golang.design/x/clipboard"
)

const x11TestDisplay = ":99"

var (
	xvfbOnce sync.Once
	xvfbErr  error
	xvfbKill func()
)

func requireXvfb(t *testing.T) {
	t.Helper()

	if _, err := exec.LookPath("Xvfb"); err != nil {
		t.Skip("Xvfb not installed, skipping integration test")
	}

	xvfbOnce.Do(func() {
		cmd := exec.Command("Xvfb", x11TestDisplay, "-screen", "0", "800x600x24", "-nolisten", "tcp")
		if err := cmd.Start(); err != nil {
			xvfbErr = err
			return
		}
		xvfbKill = func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}

		xvfbErr = errors.New("Xvfb did not become ready in time")
		deadline := time.Now().Add(10 * time.Second)
		for time.Now().Before(deadline) {
			if xvfbReady() {
				xvfbErr = nil
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	})

	if xvfbErr != nil {
		t.Skip("cannot start Xvfb:", xvfbErr)
	}
	t.Setenv("DISPLAY", x11TestDisplay)
}

func xvfbReady() bool {
	out, err := exec.Command("xclip", "-selection", "clipboard", "-o", "-display", x11TestDisplay).CombinedOutput()
	if err == nil {
		return true
	}
	return !strings.Contains(string(out), "Can't open display")
}

func TestIntegration_X11Copy_ShouldMakeTextAvailableViaXclip(t *testing.T) {
	requireXvfb(t)

	backend, err := clipboard.New(clipboard.Options{
		GOOS:    "linux",
		Display: x11TestDisplay,
	})
	require.NoError(t, err)

	err = backend.Copy("hello from ssr backend")
	require.NoError(t, err)

	out, err := exec.Command("xclip", "-selection", "clipboard", "-o", "-display", x11TestDisplay).Output()
	require.NoError(t, err)
	assert.Equal(t, "hello from ssr backend", string(out))
}

func TestIntegration_X11Copy_ShouldMakeTextAvailableViaLibraryRead(t *testing.T) {
	requireXvfb(t)

	backend, err := clipboard.New(clipboard.Options{
		GOOS:    "linux",
		Display: x11TestDisplay,
	})
	require.NoError(t, err)

	err = backend.Copy("verify via library")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := gclip.Read(ctx, gclip.FmtText)
	require.NoError(t, err)
	assert.Equal(t, "verify via library", string(data))
}
