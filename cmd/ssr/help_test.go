package main_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSR_Help_ShouldPrintRussianUsage(t *testing.T) {
	out, err := runSSR(t, "", "--help")

	require.NoError(t, err)
	assert.Contains(t, out, "ssr")
	assert.Contains(t, out, "stdin")
	assert.Contains(t, out, "секрет")
	assert.Contains(t, out, "--stdout")
	assert.Contains(t, out, "--dry-run")
	assert.Contains(t, out, "--backend")
	assert.Contains(t, out, "--timeout")
	assert.Contains(t, out, "--version")
	assert.Contains(t, out, "--help")
}
