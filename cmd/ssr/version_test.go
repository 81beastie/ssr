package main_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSR_Version_ShouldMatchBuildInfo(t *testing.T) {
	out, err := runSSR(t, "", "-v")
	require.NoError(t, err)

	assert.NotContains(t, out, "0.1.0", "версия не должна быть устаревшей константой")

	printed := printedVersion(t, out)
	assert.Equal(t, expectedVersion(t), printed)
}

func printedVersion(t *testing.T, output string) string {
	t.Helper()

	fields := strings.Fields(output)
	require.GreaterOrEqual(t, len(fields), 2, "вывод -v должен быть вида 'ssr <версия>'")
	return fields[1]
}

func expectedVersion(t *testing.T) string {
	t.Helper()

	info, err := exec.Command("go", "version", "-m", "./"+binaryName).Output()
	require.NoError(t, err)

	for _, line := range strings.Split(string(info), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "mod" {
			return normalizeVersion(fields[2])
		}
	}
	t.Fatal("в build info нет строки mod")
	return ""
}

func normalizeVersion(raw string) string {
	if raw == "" || raw == "(devel)" {
		return "dev"
	}
	return strings.TrimPrefix(raw, "v")
}
