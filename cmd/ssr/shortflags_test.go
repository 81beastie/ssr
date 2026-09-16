package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSR_ShortFlags_ShouldWork(t *testing.T) {
	out, err := runSSR(t, "password=hunter2 and TOKEN_A=alpha\n", "-s")

	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password> and TOKEN_A=<ssr token>\n", out)
}

func TestSSR_ShortFlagDryRun_ShouldReportFindings(t *testing.T) {
	out, err := runSSR(t, "password=hunter2\n", "-n")

	require.NoError(t, err)
	assert.Contains(t, out, "hunter2")
}

func TestSSR_ShortFlagVersion_ShouldPrintVersion(t *testing.T) {
	out, err := runSSR(t, "", "-v")

	require.NoError(t, err)
	assert.Contains(t, out, "ssr")
}

func TestSSR_ShortFlagHelp_ShouldPrintUsage(t *testing.T) {
	out, err := runSSR(t, "", "-h")

	require.NoError(t, err)
	assert.Contains(t, out, "секрет")
}

func TestSSR_ShortFlagBackend_ShouldUseOSC52(t *testing.T) {
	out, err := runSSR(t, "plain text\n", "-b", "osc52")

	require.NoError(t, err)
	assert.Contains(t, out, "]52;c;")
}

func TestSSR_ShortFlagTimeout_ShouldBeAccepted(t *testing.T) {
	out, err := runSSR(t, "password=hunter2\n", "-s", "-t", "5m")

	require.NoError(t, err)
	assert.Contains(t, out, "<ssr password>")
}
