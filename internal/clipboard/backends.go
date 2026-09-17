package clipboard

import (
	"context"
	"encoding/base64"
	"io"
	"os"
	"sync"

	gclip "golang.design/x/clipboard"
)

var (
	gclipInitOnce sync.Once
	gclipInitErr  error
)

func gclipInit() error {
	gclipInitOnce.Do(func() {
		gclipInitErr = gclip.Init()
	})
	return gclipInitErr
}

type textWriter interface {
	Init() error
	Write(p []byte) (<-chan struct{}, error)
}

type gclipWriter struct{}

func (gclipWriter) Init() error { return gclipInit() }

func (gclipWriter) Write(p []byte) (<-chan struct{}, error) {
	return gclip.Write(context.Background(), gclip.FmtText, p)
}

func copyVia(w textWriter, text string) error {
	if w == nil {
		w = gclipWriter{}
	}
	if err := w.Init(); err != nil {
		return err
	}
	// Возвращаемый канал — сигнал владения буфером (его использует демон),
	// в одноразовом Copy мы за ним не наблюдаем.
	if _, err := w.Write([]byte(text)); err != nil {
		return err
	}
	return nil
}

type X11 struct {
	Writer textWriter
}

func (x *X11) Copy(text string) error {
	return copyVia(x.Writer, text)
}

type Wayland struct {
	Writer textWriter
}

func (w *Wayland) Copy(text string) error {
	return copyVia(w.Writer, text)
}

type Win32 struct {
	Writer textWriter
}

func (c *Win32) Copy(text string) error {
	return copyVia(c.Writer, text)
}

type Mac struct {
	Writer textWriter
}

func (m *Mac) Copy(text string) error {
	return copyVia(m.Writer, text)
}

type OSC52 struct {
	Writer io.Writer
}

func (o *OSC52) Copy(text string) error {
	w := o.Writer
	if w == nil {
		w = os.Stdout
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	_, err := io.WriteString(w, "\x1b]52;c;"+encoded+"\x07")
	return err
}
