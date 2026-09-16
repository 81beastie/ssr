package clipboard

import (
	"context"
	"encoding/base64"
	"fmt"
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

type X11 struct{}

func (x *X11) Copy(text string) error {
	if err := gclipInit(); err != nil {
		return err
	}
	_, err := gclip.Write(context.Background(), gclip.FmtText, []byte(text))
	return err
}

type Wayland struct{}

func (w *Wayland) Copy(text string) error {
	if err := gclipInit(); err != nil {
		return err
	}
	_, err := gclip.Write(context.Background(), gclip.FmtText, []byte(text))
	return err
}

type Win32 struct{}

func (w *Win32) Copy(text string) error {
	if err := gclipInit(); err != nil {
		return err
	}
	_, err := gclip.Write(context.Background(), gclip.FmtText, []byte(text))
	return err
}

type Mac struct{}

func (m *Mac) Copy(text string) error {
	if err := gclipInit(); err != nil {
		return err
	}
	_, err := gclip.Write(context.Background(), gclip.FmtText, []byte(text))
	return err
}

type OSC52 struct {
	Out io.Writer
}

func (o *OSC52) Copy(text string) error {
	if o.Out == nil {
		o.Out = os.Stdout
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	_, err := fmt.Fprintf(o.Out, "\x1b]52;c;%s\x07", encoded)
	return err
}
