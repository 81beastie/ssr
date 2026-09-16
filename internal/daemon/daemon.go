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
}

func Serve(config Config) (int, error) {
	exe, err := os.Executable()
	if err != nil {
		return 0, fmt.Errorf("daemon: cannot resolve executable: %w", err)
	}

	args := []string{"--serve"}
	if config.Timeout > 0 {
		args = append(args, "--timeout", config.Timeout.String())
	}

	stdinRead, stdinWrite, stdinErr := os.Pipe()
	stdoutRead, stdoutWrite, stdoutErr := os.Pipe()
	if stdinErr != nil || stdoutErr != nil {
		return 0, fmt.Errorf("daemon: cannot create pipes")
	}

	cmd := exec.Command(exe, args...)
	cmd.Env = withDisplay(os.Environ(), config.Display)
	cmd.Stderr = nil
	cmd.Stdin = stdinRead
	cmd.Stdout = stdoutWrite

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("daemon: cannot start serve process: %w", err)
	}

	_ = stdinRead.Close()
	_ = stdoutWrite.Close()

	if err := sendTextAndWaitReady(stdinWrite, stdoutRead, config.Text); err != nil {
		return 0, fmt.Errorf("daemon: %w", err)
	}

	go func() {
		_ = cmd.Wait()
	}()

	return cmd.Process.Pid, nil
}

func sendTextAndWaitReady(stdin io.WriteCloser, stdout io.Reader, text string) error {
	if _, err := io.WriteString(stdin, text); err != nil {
		return fmt.Errorf("cannot send text: %w", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("cannot close stdin: %w", err)
	}

	reader := bufio.NewReader(stdout)
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if strings.TrimSpace(line) == readySignal {
			return nil
		}
		if err != nil {
			return fmt.Errorf("serve process exited before ready: %w", err)
		}
	}
	return fmt.Errorf("serve process did not confirm readiness in 10s")
}

func RunServe(timeout time.Duration) error {
	text, err := io.ReadAll(os.Stdin)
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
