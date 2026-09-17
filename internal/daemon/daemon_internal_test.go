package daemon

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testError = errTest("тестовая ошибка")

type errTest string

func (e errTest) Error() string { return string(e) }

type launcherFunc func(Config) (*launchResult, error)

func (f launcherFunc) launch(config Config) (*launchResult, error) { return f(config) }

type blockedReader struct{}

func (blockedReader) Read([]byte) (int, error) { select {} }

type failingCloser struct {
	io.WriteCloser
}

func (failingCloser) Close() error { return testError }

func mustPipe(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	read, write, err := os.Pipe()
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = read.Close()
		_ = write.Close()
	})
	return read, write
}

func TestWithDisplay_KeepEnv_WhenDisplayEmpty(t *testing.T) {
	env := []string{"PATH=/usr/bin", "DISPLAY=:0"}

	assert.Equal(t, env, withDisplay(env, ""))
}

func TestWithDisplay_ReplaceInPlace_WhenDisplayPresent(t *testing.T) {
	env := []string{"PATH=/usr/bin", "DISPLAY=:0", "HOME=/root"}

	got := withDisplay(env, ":77")

	assert.Equal(t, []string{"PATH=/usr/bin", "DISPLAY=:77", "HOME=/root"}, got)
}

func TestWithDisplay_Append_WhenDisplayMissing(t *testing.T) {
	got := withDisplay([]string{"PATH=/usr/bin"}, ":77")

	assert.Equal(t, []string{"PATH=/usr/bin", "DISPLAY=:77"}, got)
}

func TestWithDisplay_KeepSingleEntry_WhenDisplayDuplicated(t *testing.T) {
	got := withDisplay([]string{"DISPLAY=:1", "DISPLAY=:2", "PATH=/usr/bin"}, ":77")

	assert.Equal(t, []string{"DISPLAY=:77", "PATH=/usr/bin"}, got)
}

func TestSendTextAndWaitReady_ReturnNil_WhenReadyReceived(t *testing.T) {
	stdinRead, stdinWrite := mustPipe(t)
	go io.Copy(io.Discard, stdinRead)
	stdout := strings.NewReader(readySignal + "\n")

	require.NoError(t, sendTextAndWaitReady(stdinWrite, stdout, "sekret", time.Second))
}

func TestSendTextAndWaitReady_ReportError_WhenWriteFails(t *testing.T) {
	stdinRead, stdinWrite := mustPipe(t)
	require.NoError(t, stdinRead.Close())

	err := sendTextAndWaitReady(stdinWrite, strings.NewReader(""), "sekret", time.Second)

	require.ErrorContains(t, err, "cannot send text")
}

func TestSendTextAndWaitReady_ReportError_WhenCloseFails(t *testing.T) {
	stdinRead, stdinWrite := mustPipe(t)
	go io.Copy(io.Discard, stdinRead)

	err := sendTextAndWaitReady(failingCloser{stdinWrite}, strings.NewReader(""), "sekret", time.Second)

	require.ErrorContains(t, err, "cannot close stdin")
}

func TestSendTextAndWaitReady_ReportError_WhenExitBeforeReady(t *testing.T) {
	stdinRead, stdinWrite := mustPipe(t)
	go io.Copy(io.Discard, stdinRead)

	err := sendTextAndWaitReady(stdinWrite, strings.NewReader("proshii vyvod\n"), "sekret", time.Second)

	require.ErrorContains(t, err, "exited before ready")
}

func TestSendTextAndWaitReady_ReportError_WhenDeadlineExceeded(t *testing.T) {
	stdinRead, stdinWrite := mustPipe(t)
	go io.Copy(io.Discard, stdinRead)

	err := sendTextAndWaitReady(stdinWrite, blockedReader{}, "sekret", 50*time.Millisecond)

	require.ErrorContains(t, err, "did not confirm readiness")
}

func TestServe_ReportError_WhenLaunchFails(t *testing.T) {
	_, err := serveWith(launcherFunc(func(Config) (*launchResult, error) {
		return nil, testError
	}), Config{Text: "sekret"})

	require.ErrorContains(t, err, "daemon: launch failed")
}

func TestServe_ReportError_WhenProcessDiesBeforeReady(t *testing.T) {
	stdinRead, stdinWrite := mustPipe(t)
	stdoutRead, stdoutWrite := mustPipe(t)
	require.NoError(t, stdoutRead.Close())
	require.NoError(t, stdoutWrite.Close())

	result := &launchResult{
		pid:        4242,
		stdin:      stdinWrite,
		stdout:     stdoutRead,
		waitCh:     make(chan struct{}),
		killParent: func() {},
	}
	go stdinRead.Close()

	_, err := serveWith(launcherFunc(func(Config) (*launchResult, error) {
		return result, nil
	}), Config{Text: "sekret"})

	require.ErrorContains(t, err, "exited before ready")
}

func TestLaunchReal_ReportError_WhenExecutableMissing(t *testing.T) {
	_, err := launchReal(Config{Exe: "/nonexistent/ssr-test-exe"})

	require.ErrorContains(t, err, "cannot start serve process")
}

func TestRunServe_ReportError_WhenStdinUnreadable(t *testing.T) {
	read, write, err := os.Pipe()
	require.NoError(t, err)
	require.NoError(t, write.Close())

	err = runServe(closedFile{read}, time.Second)

	require.ErrorContains(t, err, "cannot read text from stdin")
}

type closedFile struct{ *os.File }

func (c closedFile) Read([]byte) (int, error) {
	_ = c.File.Close()
	return 0, os.ErrClosed
}

func TestRunServe_ReturnByTimeout_WhenClipboardReady(t *testing.T) {
	requireInternalXvfb(t, ":97")
	t.Setenv("DISPLAY", ":97")

	start := time.Now()
	err := runServe(strings.NewReader("tekst dlya bufera"), 300*time.Millisecond)

	require.NoError(t, err)
	assert.Less(t, time.Since(start), 3*time.Second, "демон обязан умереть по таймауту")
}

func requireInternalXvfb(t *testing.T, display string) {
	t.Helper()

	if _, err := os.Stat("/tmp/.X11-unix/X" + display[1:]); err == nil {
		if exec.Command("pgrep", "-f", "Xvfb "+display).Run() == nil {
			return
		}
		_ = os.Remove("/tmp/.X11-unix/X" + display[1:])
	}
	if _, err := exec.LookPath("Xvfb"); err != nil {
		t.Skip("Xvfb not installed")
	}

	cmd := exec.Command("Xvfb", display, "-screen", "0", "800x600x24", "-nolisten", "tcp")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat("/tmp/.X11-unix/X" + display[1:]); err == nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("Xvfb did not become ready in time")
}
