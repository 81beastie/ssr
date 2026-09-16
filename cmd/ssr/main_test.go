package main_test

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testDisplay = ":97"
	binaryName  = "ssr"
)

var binaryPath string

func TestMain(m *testing.M) {
	build := exec.Command("go", "build", "-o", binaryName, ".")
	if out, err := build.CombinedOutput(); err != nil {
		panic("cannot build ssr binary: " + string(out))
	}

	code := m.Run()
	_ = exec.Command("rm", "-f", binaryName).Run()
	_ = exec.Command("pkill", "-f", "Xvfb "+testDisplay).Run()
	os.Exit(code)
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

func runSSR(t *testing.T, input string, args ...string) (string, error) {
	t.Helper()

	cmd := exec.Command("./"+binaryName, args...)
	cmd.Stdin = strings.NewReader(input)
	cmd.Env = append(cmd.Environ(), "DISPLAY="+testDisplay)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestSSR_Stdout_ShouldPrintRedactedTextWithoutClipboard(t *testing.T) {
	out, err := runSSR(t, "password=hunter2 and TOKEN_A=alpha\n", "--stdout")

	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password> and TOKEN_A=<ssr token>\n", out)
}

func TestSSR_DryRun_ShouldReportFindingsWithoutCopying(t *testing.T) {
	out, err := runSSR(t, "password=hunter2\n", "--dry-run")

	require.NoError(t, err)
	assert.Contains(t, out, "password")
	assert.Contains(t, out, "hunter2")
}

func TestSSR_Version_ShouldPrintVersion(t *testing.T) {
	out, err := runSSR(t, "", "--version")

	require.NoError(t, err)
	assert.Contains(t, out, "ssr")
}

func TestSSR_Default_ShouldCopyToClipboardAndExit(t *testing.T) {
	requireXvfb(t)

	_, err := runSSR(t, "password=hunter2\n")
	require.NoError(t, err)

	out, err := exec.Command("xclip", "-selection", "clipboard", "-o", "-display", testDisplay).Output()
	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password>", strings.TrimSpace(string(out)))
}

func TestSSR_BackendOverride_ShouldUseOSC52(t *testing.T) {
	out, err := runSSR(t, "plain text\n", "--backend", "osc52")

	require.NoError(t, err)
	assert.Contains(t, out, "]52;c;")
}

func TestSSR_StdinEmpty_ShouldNotFail(t *testing.T) {
	out, err := runSSR(t, "", "--stdout")

	require.NoError(t, err)
	assert.Equal(t, "", strings.TrimSpace(out))
}
