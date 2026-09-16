package main_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func TestSSR_FileToStdout_ShouldPrintRedacted(t *testing.T) {
	path := writeFile(t, "input.txt", "password=hunter2 TOKEN_A=alpha")

	out, err := runSSR(t, "", "-f", path, "-s")

	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password> TOKEN_A=<ssr token>", strings.TrimSpace(out))
}

func TestSSR_FileToClipboard_ShouldCopyRedacted(t *testing.T) {
	requireXvfb(t)

	path := writeFile(t, "input.txt", "password=hunter2")

	_, err := runSSR(t, "", "-f", path)
	require.NoError(t, err)

	out, err := exec.Command("xclip", "-selection", "clipboard", "-o", "-display", testDisplay).Output()
	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password>", strings.TrimSpace(string(out)))
}

func TestSSR_FileToSsrFile_ShouldCreateFileWithExtension(t *testing.T) {
	path := writeFile(t, "input.txt", "password=hunter2")

	_, err := runSSR(t, "", "-f", path, "-o", ".ssr")
	require.NoError(t, err)

	result := readTestFile(t, path+".ssr")
	assert.Equal(t, "password=<ssr password>", strings.TrimSpace(result))
}

func TestSSR_FileToUserFile_ShouldWriteToSpecifiedPath(t *testing.T) {
	path := writeFile(t, "input.txt", "password=hunter2")
	target := filepath.Join(t.TempDir(), "clean.txt")

	_, err := runSSR(t, "", "-f", path, "-o", target)
	require.NoError(t, err)

	result := readTestFile(t, target)
	assert.Equal(t, "password=<ssr password>", strings.TrimSpace(result))
}

func TestSSR_MissingFile_ShouldFailWithClearError(t *testing.T) {
	out, err := runSSR(t, "", "-f", "/nonexistent/file.txt")

	require.Error(t, err)
	assert.Contains(t, out, "не найден")
}
