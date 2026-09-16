package clipboard

import "fmt"

type Clipboard interface {
	Copy(text string) error
}

type Options struct {
	GOOS           string
	WaylandDisplay string
	Display        string
	Override       string
}

func New(options Options) (Clipboard, error) {
	if options.Override != "" {
		return byName(options.Override)
	}

	switch options.GOOS {
	case "windows":
		return &Win32{}, nil
	case "darwin":
		return &Mac{}, nil
	}

	if options.WaylandDisplay != "" {
		return &Wayland{}, nil
	}
	if options.Display != "" {
		return &X11{}, nil
	}
	return &OSC52{}, nil
}

func byName(name string) (Clipboard, error) {
	switch name {
	case "x11":
		return &X11{}, nil
	case "wayland":
		return &Wayland{}, nil
	case "win32":
		return &Win32{}, nil
	case "mac":
		return &Mac{}, nil
	case "osc52":
		return &OSC52{}, nil
	}
	return nil, fmt.Errorf("unknown backend: %q", name)
}
