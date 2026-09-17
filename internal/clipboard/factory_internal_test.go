package clipboard

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWriter struct {
	writes   []string
	initCall bool
	initErr  error
	writeErr error
}

func (f *fakeWriter) Init() error {
	f.initCall = true
	return f.initErr
}

func (f *fakeWriter) Write(p []byte) (<-chan struct{}, error) {
	if !f.initCall {
		panic("Write called before Init")
	}
	if f.writeErr != nil {
		return nil, f.writeErr
	}
	f.writes = append(f.writes, string(p))
	done := make(chan struct{})
	close(done)
	return done, nil
}

func TestByName_ShouldReturnBackend_ForEveryKnownName(t *testing.T) {
	tests := []struct {
		name     string
		expected Clipboard
	}{
		{"x11", &X11{}},
		{"wayland", &Wayland{}},
		{"win32", &Win32{}},
		{"mac", &Mac{}},
		{"osc52", &OSC52{}},
	}

	for _, tt := range tests {
		backend, err := byName(tt.name)

		require.NoError(t, err, tt.name)
		assert.IsType(t, tt.expected, backend, tt.name)
	}
}

func TestByName_ShouldReturnError_ForUnknownName(t *testing.T) {
	backend, err := byName("ne-bekend")

	require.Error(t, err)
	assert.Nil(t, backend)
}

func TestOSC52Copy_ShouldEmitBase64Escape_ThroughWriter(t *testing.T) {
	out := &bytes.Buffer{}
	backend := &OSC52{Writer: out}

	require.NoError(t, backend.Copy("hunter2"))

	expected := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte("hunter2")) + "\x07"
	assert.Equal(t, expected, out.String())
}

func TestOSC52Copy_ShouldFallBackToStdout_WhenWriterNil(t *testing.T) {
	read, write, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = original }()

	require.NoError(t, (&OSC52{}).Copy("tekst"))

	require.NoError(t, write.Close())
	out := &bytes.Buffer{}
	_, err = io.Copy(out, read)
	require.NoError(t, err)
	os.Stdout = original

	assert.Contains(t, out.String(), "\x1b]52;c;")
}

func TestWin32Copy_ShouldUseInjectedWriter(t *testing.T) {
	w := &fakeWriter{}
	backend := &Win32{Writer: w}

	require.NoError(t, backend.Copy("tekst"))

	assert.Equal(t, []string{"tekst"}, w.writes)
}

func TestMacCopy_ShouldUseInjectedWriter(t *testing.T) {
	w := &fakeWriter{}
	backend := &Mac{Writer: w}

	require.NoError(t, backend.Copy("tekst"))

	assert.Equal(t, []string{"tekst"}, w.writes)
}

func TestWaylandCopy_ShouldUseInjectedWriter(t *testing.T) {
	w := &fakeWriter{}
	backend := &Wayland{Writer: w}

	require.NoError(t, backend.Copy("tekst"))

	assert.Equal(t, []string{"tekst"}, w.writes)
}

func TestCopyVia_InitError_ShouldReturnIt(t *testing.T) {
	backend := &X11{Writer: &fakeWriter{initErr: errors.New("x11 init failed")}}

	require.ErrorContains(t, backend.Copy("tekst"), "x11 init failed")
}

func TestCopyVia_WriteError_ShouldReturnIt(t *testing.T) {
	backend := &X11{Writer: &fakeWriter{writeErr: errors.New("x11 write failed")}}

	require.ErrorContains(t, backend.Copy("tekst"), "x11 write failed")
}
