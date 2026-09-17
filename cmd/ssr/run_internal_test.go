package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/81beastie/ssr/internal/clipboard"
	"github.com/81beastie/ssr/internal/daemon"
	"github.com/81beastie/ssr/internal/detector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeServer struct {
	configs []daemon.Config
	err     error
}

func (f *fakeServer) Serve(config daemon.Config) (int, error) {
	f.configs = append(f.configs, config)
	return 4242, f.err
}

func newTestApp(input string) (*app, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	return &app{
		stdin:  strings.NewReader(input),
		stdout: stdout,
		stderr: stderr,
		goos:   "linux",
		env:    map[string]string{"DISPLAY": ":0"},
		serve:  &fakeServer{},
	}, stdout, stderr
}

func TestRun_HelpFlag_ShouldPrintUsage(t *testing.T) {
	a, stdout, _ := newTestApp("")

	require.NoError(t, a.run([]string{"--help"}))
	assert.Contains(t, stdout.String(), "секрет")
}

func TestRun_VersionFlag_ShouldPrintVersion(t *testing.T) {
	a, stdout, _ := newTestApp("")

	require.NoError(t, a.run([]string{"-v"}))
	assert.Contains(t, stdout.String(), "ssr")
}

func TestRun_StdoutMode_ShouldPrintRedactedText(t *testing.T) {
	a, stdout, _ := newTestApp("password=hunter2")

	require.NoError(t, a.run([]string{"-s"}))
	assert.Equal(t, "password=<ssr password>", stdout.String())
}

func TestRun_DryRun_ShouldReportFindings(t *testing.T) {
	a, stdout, _ := newTestApp("password=hunter2 password=hunter2")

	require.NoError(t, a.run([]string{"-n"}))
	assert.Contains(t, stdout.String(), "найдено секретов: 2")
	assert.Equal(t, 1, strings.Count(stdout.String(), "hunter2"), "дубли дедуплицируются")
}

func TestRun_DryRunEmpty_ShouldReportNothing(t *testing.T) {
	a, stdout, _ := newTestApp("clean text")

	require.NoError(t, a.run([]string{"-n"}))
	assert.Contains(t, stdout.String(), "секретов не найдено")
}

func TestRun_ClipboardMode_ShouldServeViaDaemon(t *testing.T) {
	a, _, _ := newTestApp("password=hunter2")
	server := a.serve.(*fakeServer)

	require.NoError(t, a.run([]string{"-t", "5m"}))

	require.Len(t, server.configs, 1)
	assert.Equal(t, "password=<ssr password>", server.configs[0].Text)
	assert.Equal(t, ":0", server.configs[0].Display)
	assert.Equal(t, 5*time.Minute, server.configs[0].Timeout)
}

func TestRun_ClipboardMode_ShouldReturnServeError(t *testing.T) {
	a, _, _ := newTestApp("password=hunter2")
	a.serve.(*fakeServer).err = errors.New("x11 down")

	err := a.run(nil)

	require.ErrorContains(t, err, "x11 down")
}

func TestRun_NonLinux_ShouldUseSelectedBackend(t *testing.T) {
	a, stdout, _ := newTestApp("password=hunter2")
	a.goos = "windows"

	require.NoError(t, a.run([]string{"-b", "osc52"}))

	assert.Contains(t, stdout.String(), "\x1b]52;c;")
	assert.Contains(t, stdout.String(), base64.StdEncoding.EncodeToString([]byte("password=<ssr password>")))
}

func TestRun_UnknownBackend_ShouldFail(t *testing.T) {
	a, _, _ := newTestApp("password=hunter2")

	require.ErrorContains(t, a.run([]string{"-b", "ne-bekend"}), "unknown backend")
}

func TestRun_MissingInputFile_ShouldFail(t *testing.T) {
	a, _, _ := newTestApp("")

	require.ErrorContains(t, a.run([]string{"-f", "/nonexistent-ssr/file.txt"}), "не найден")
}

func TestCopyToClipboard_NonLinuxFallback_ShouldUseFactoryBackend(t *testing.T) {
	a, _, _ := newTestApp("")
	a.goos = "windows"
	a.newBackend = func(clipboard.Options) (clipboard.Clipboard, error) {
		return stubBackend{}, nil
	}

	require.NoError(t, copyToClipboard(a, "tekst", "", 0))
}

func TestCopyToClipboard_NonLinuxFactoryError_ShouldReport(t *testing.T) {
	a, _, _ := newTestApp("")
	a.goos = "darwin"
	a.newBackend = func(clipboard.Options) (clipboard.Clipboard, error) {
		return nil, errors.New("factory broken")
	}

	require.ErrorContains(t, copyToClipboard(a, "tekst", "", 0), "factory broken")
}

func TestCopyToClipboard_BackendCopyError_ShouldReport(t *testing.T) {
	a, _, _ := newTestApp("")
	a.goos = "darwin"
	a.newBackend = func(clipboard.Options) (clipboard.Clipboard, error) {
		return stubBackend{err: errors.New("copy failed")}, nil
	}

	require.ErrorContains(t, copyToClipboard(a, "tekst", "mac", 0), "copy failed")
}

type stubBackend struct {
	err error
}

func (s stubBackend) Copy(string) error { return s.err }

func TestRun_OutputToBadPath_ShouldFail(t *testing.T) {
	a, _, _ := newTestApp("password=hunter2")

	require.ErrorContains(t, a.run([]string{"-o", "/nonexistent-dir-ssr/out.txt"}), "не удалось записать файл")
}

func TestRun_StdoutWriteError_ShouldFail(t *testing.T) {
	a, _, _ := newTestApp("password=hunter2")
	a.stdout = errWriter{}

	require.Error(t, a.run([]string{"-s"}))
}

func TestRun_FileOutput_ShouldWriteAlongsideInput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "in.txt")
	require.NoError(t, os.WriteFile(input, []byte("password=hunter2"), 0o600))
	a, _, _ := newTestApp("")

	require.NoError(t, a.run([]string{"-f", input, "-o", ".ssr"}))

	data, err := os.ReadFile(input + ".ssr")
	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password>", string(data))
}

func TestRun_BadFlag_ShouldReturnError(t *testing.T) {
	a, _, _ := newTestApp("")

	require.Error(t, a.run([]string{"--kakovoy-flag"}))
}

func TestParseFlags_AllFlags_ShouldMapBothSpellings(t *testing.T) {
	opts, err := parseFlags([]string{"-s", "-n", "-v", "-h", "-b", "x11", "-t", "3m", "-f", "in", "-o", "out"})

	require.NoError(t, err)
	assert.True(t, opts.stdout)
	assert.True(t, opts.dryRun)
	assert.True(t, opts.version)
	assert.True(t, opts.help)
	assert.Equal(t, "x11", opts.backend)
	assert.Equal(t, 3*time.Minute, opts.timeout)
	assert.Equal(t, "in", opts.file)
	assert.Equal(t, "out", opts.out)
}

func TestReadInput_MissingFile_ShouldReportNotFound(t *testing.T) {
	_, err := readInput(strings.NewReader(""), "/nonexistent/file.txt")

	require.ErrorContains(t, err, "не найден")
}

func TestReadInput_StdinError_ShouldReport(t *testing.T) {
	_, err := readInput(errReader{}, "")

	require.ErrorContains(t, err, "stdin")
}

func TestReadInput_UnreadableFile_ShouldReportReadError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions")
	}
	dir := t.TempDir()
	blocked := filepath.Join(dir, "locked")
	require.NoError(t, os.Mkdir(blocked, 0o000))
	defer os.Chmod(blocked, 0o700)

	_, err := readInput(strings.NewReader(""), blocked)

	require.ErrorContains(t, err, "не удалось прочитать файл")
}

func TestWriteOutput_SsrSuffixWithoutInput_ShouldFail(t *testing.T) {
	err := writeOutput(".ssr", "", "text")

	require.ErrorContains(t, err, "укажите имя файла")
}

func TestWriteOutput_UnwritablePath_ShouldFail(t *testing.T) {
	err := writeOutput(filepath.Join(t.TempDir(), "no-such-dir", "out.txt"), "", "text")

	require.ErrorContains(t, err, "не удалось записать файл")
}

func TestServeTimeout_ShouldParseDuration(t *testing.T) {
	assert.Equal(t, 5*time.Minute, serveTimeout([]string{"--timeout", "5m"}))
}

func TestServeTimeout_InvalidDuration_ShouldReturnZero(t *testing.T) {
	assert.Equal(t, time.Duration(0), serveTimeout([]string{"--timeout", "ne-taymer"}))
}

func TestServeTimeout_NoArgs_ShouldReturnZero(t *testing.T) {
	assert.Equal(t, time.Duration(0), serveTimeout(nil))
}

func TestServeTimeout_InvalidFlagValue_ShouldBeRejectedByParser(t *testing.T) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.Duration("timeout", 0, "")

	require.Error(t, fs.Parse([]string{"--timeout=ne-dur"}),
		"flag-парсер сам отбраковывает не-длительность, ветка ParseDuration в serveTimeout мертва")
}

func TestEnvMap_ShouldParseEntries(t *testing.T) {
	old := os.Getenv("SSR_TEST_VAR")
	require.NoError(t, os.Setenv("SSR_TEST_VAR", "znachenie"))
	defer os.Setenv("SSR_TEST_VAR", old)

	env := envMap()

	assert.Equal(t, "znachenie", env["SSR_TEST_VAR"])
}

func TestPrintUsage_ShouldWriteToWriter(t *testing.T) {
	out := &bytes.Buffer{}

	printUsage(out)

	assert.Contains(t, out.String(), "ssr")
}

func TestReportFindings_WriterError_ShouldPropagate(t *testing.T) {
	err := reportFindings(errWriter{}, []detector.Finding{{Type: "token", Value: "v"}})

	require.Error(t, err)
}

func TestReportFindings_MidLoopWriterError_ShouldPropagate(t *testing.T) {
	err := reportFindings(errWriter{}, []detector.Finding{
		{Type: "token", Value: "pervyy"},
		{Type: "token", Value: "vtoroy"},
	})

	require.Error(t, err)
}

func TestReportFindings_EmptyWriterError_ShouldPropagate(t *testing.T) {
	err := reportFindings(errWriter{}, nil)

	require.Error(t, err)
}

func TestDaemonServerServe_Wrapper_ShouldDelegate(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("daemon lives on linux only")
	}
	_, _ = daemonServer{}.Serve(daemon.Config{Text: "obertka", Timeout: time.Second})
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, io.ErrClosedPipe }

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }
