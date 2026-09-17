package daemon

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	gclip "golang.design/x/clipboard"
)

const (
	readySignal   = "READY"
	defaultMaxAge = 24 * time.Hour
)

type Config struct {
	Display string
	Text    string
	Timeout time.Duration
	Exe     string
}

const readyTimeout = 10 * time.Second

type launchResult struct {
	pid        int
	stdin      *os.File
	stdout     *os.File
	waitCh     chan struct{}
	killParent func()
}

type launcher interface {
	launch(config Config) (*launchResult, error)
}

type launchFunc func(Config) (*launchResult, error)

func (f launchFunc) launch(config Config) (*launchResult, error) { return f(config) }

func Serve(config Config) (int, error) {
	return serveWith(launchFunc(launchReal), config)
}

func serveWith(l launcher, config Config) (int, error) {
	result, err := l.launch(config)
	if err != nil {
		return 0, fmt.Errorf("daemon: launch failed: %w", err)
	}

	if err := sendTextAndWaitReady(result.stdin, result.stdout, config.Text, readyTimeout); err != nil {
		result.killParent()
		return 0, fmt.Errorf("daemon: %w", err)
	}

	return result.pid, nil
}

func launchReal(config Config) (*launchResult, error) {
	exe := config.Exe
	if exe == "" {
		resolved, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("cannot resolve executable: %w", err)
		}
		exe = resolved
	}

	args := []string{"--serve"}
	if config.Timeout > 0 {
		args = append(args, "--timeout", config.Timeout.String())
	}

	stdinRead, stdinWrite, stdinErr := os.Pipe()
	stdoutRead, stdoutWrite, stdoutErr := os.Pipe()
	if stdinErr != nil || stdoutErr != nil {
		return nil, fmt.Errorf("cannot create pipes")
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = withDisplay(os.Environ(), config.Display)
	cmd.Stderr = nil
	cmd.Stdin = stdinRead
	cmd.Stdout = stdoutWrite

	if err := cmd.Start(); err != nil {
		_ = stdinRead.Close()
		_ = stdinWrite.Close()
		_ = stdoutRead.Close()
		_ = stdoutWrite.Close()
		return nil, fmt.Errorf("cannot start serve process: %w", err)
	}

	_ = stdinRead.Close()
	_ = stdoutWrite.Close()

	waitCh := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(waitCh)
	}()

	return &launchResult{
		pid:        cmd.Process.Pid,
		stdin:      stdinWrite,
		stdout:     stdoutRead,
		waitCh:     waitCh,
		killParent: func() { _ = cmd.Process.Kill() },
	}, nil
}

func sendTextAndWaitReady(stdin io.WriteCloser, stdout io.Reader, text string, timeout time.Duration) error {
	if _, err := io.WriteString(stdin, text); err != nil {
		return fmt.Errorf("cannot send text: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("cannot close stdin: %w", err)
	}

	type lineResult struct {
		line string
		err  error
	}
	lines := make(chan lineResult, 1)
	go func() {
		reader := bufio.NewReader(stdout)
		for {
			line, err := reader.ReadString('\n')
			if line != "" || err != nil {
				lines <- lineResult{line: line, err: err}
			}
			if err != nil {
				return
			}
		}
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case result := <-lines:
			if strings.TrimSpace(result.line) == readySignal {
				return nil
			}
			if result.err != nil {
				return fmt.Errorf("serve process exited before ready: %w", result.err)
			}
		case <-timer.C:
			return fmt.Errorf("serve process did not confirm readiness in %s", timeout)
		}
	}
}

func RunServe(timeout time.Duration) error {
	return runServe(os.Stdin, timeout)
}

func runServe(stdin io.Reader, timeout time.Duration) error {
	text, err := io.ReadAll(stdin)
	if err != nil {
		return fmt.Errorf("daemon: cannot read text from stdin: %w", err)
	}

	if err := gclip.Init(); err != nil {
		return fmt.Errorf("daemon: clipboard init failed: %w", err)
	}

	lost, err := gclip.Write(context.Background(), gclip.FmtText, text)
	if err != nil {
		return fmt.Errorf("daemon: clipboard write failed: %w", err)
	}

	fmt.Println(readySignal)

	maxAge := defaultMaxAge
	if timeout > 0 {
		maxAge = timeout
	}

	select {
	case <-lost:
	case <-time.After(maxAge):
	}

	return nil
}

func withDisplay(env []string, display string) []string {
	if display == "" {
		return env
	}
	result := make([]string, 0, len(env))
	replaced := false
	for _, e := range env {
		if strings.HasPrefix(e, "DISPLAY=") {
			if !replaced {
				result = append(result, "DISPLAY="+display)
				replaced = true
			}
			continue
		}
		result = append(result, e)
	}
	if !replaced {
		result = append(result, "DISPLAY="+display)
	}
	return result
}
