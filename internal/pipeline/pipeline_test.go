package pipeline_test

import (
	"errors"
	"testing"

	"github.com/81beastie/ssr/internal/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type spyClipboard struct {
	copied string
	err    error
}

func (s *spyClipboard) Copy(text string) error {
	if s.err != nil {
		return s.err
	}
	s.copied = text
	return nil
}

func TestRun_ShouldCopyRedactedText_WhenSecretsPresent(t *testing.T) {
	clipboard := &spyClipboard{}

	result, err := pipeline.New(clipboard).Run("password=hunter2")

	require.NoError(t, err)
	assert.Equal(t, "password=<ssr password>", clipboard.copied)
	assert.Equal(t, "password=<ssr password>", result)
}

func TestRun_ShouldCopyOriginalText_WhenNoSecrets(t *testing.T) {
	clipboard := &spyClipboard{}

	result, err := pipeline.New(clipboard).Run("plain text")
	require.NoError(t, err)
	assert.Equal(t, "plain text", clipboard.copied)
	assert.Equal(t, "plain text", result)
}

func TestRun_ShouldReturnClipboardError_WhenCopyFails(t *testing.T) {
	clipboard := &spyClipboard{err: errors.New("no clipboard tool")}

	_, err := pipeline.New(clipboard).Run("password=hunter2")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no clipboard tool")
}
