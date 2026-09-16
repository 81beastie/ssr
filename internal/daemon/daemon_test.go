package daemon_test

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/81beastie/ssr/internal/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDisplay = ":98"

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == "--serve" {
		timeout := time.Duration(0)
		for i, arg := range os.Args {
			if arg == "--timeout" && i+1 < len(os.Args) {
				parsed, err := time.ParseDuration(os.Args[i+1])
				if err == nil {
					timeout = parsed
				}
			}
		}
		if err := daemon.RunServe(timeout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func requireXvfb(t *testing.T) {
	t.Helper()

	for _, tool := range []string{"Xvfb", "xclip"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skip(tool + " not installed, skipping integration test")
		}
	}

	if err := exec.Command("pkill", "-f", "Xvfb "+testDisplay).Run(); err == nil {
		time.Sleep(500 * time.Millisecond)
	}

	cmd := exec.Command("Xvfb", testDisplay, "-screen", "0", "800x600x24", "-nolisten", "tcp")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		out, _ := exec.Command("xclip", "-selection", "clipboard", "-o", "-display", testDisplay).CombinedOutput()
		if !strings.Contains(string(out), "Can't open display") {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("Xvfb did not become ready in time")
}

func xclipCopy(t *testing.T, text string) {
	t.Helper()

	cmd := exec.Command("xclip", "-selection", "clipboard", "-display", testDisplay)
	cmd.Stdin = strings.NewReader(text)
	require.NoError(t, cmd.Run())
}

func xclipPaste(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("xclip", "-selection", "clipboard", "-o", "-display", testDisplay).Output()
	require.NoError(t, err)
	return string(out)
}

func processAlive(pid int) bool {
	return exec.Command("kill", "-0", strconv.Itoa(pid)).Run() == nil
}

func waitDeath(t *testing.T, pid int, msg string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal(msg)
}

func TestServe_ShouldHoldClipboard_UntilOwnershipStolen(t *testing.T) {
	requireXvfb(t)

	text := "config with <ssr token> inside"
	pid, err := daemon.Serve(daemon.Config{Display: testDisplay, Text: text})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = exec.Command("kill", strconv.Itoa(pid)).Run()
	})

	assert.Equal(t, text, xclipPaste(t), "daemon must serve clipboard content")

	xclipCopy(t, "new owner takes over")

	waitDeath(t, pid, "daemon must die when ownership is stolen")
	assert.Equal(t, "new owner takes over", xclipPaste(t))
}

func TestServe_ShouldDieByTimeout_WhenTimeoutConfigured(t *testing.T) {
	requireXvfb(t)

	pid, err := daemon.Serve(daemon.Config{Display: testDisplay, Text: "temporary", Timeout: 2 * time.Second})
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = exec.Command("kill", strconv.Itoa(pid)).Run()
	})

	assert.Equal(t, "temporary", xclipPaste(t))

	waitDeath(t, pid, "daemon must die when timeout expires")
}
