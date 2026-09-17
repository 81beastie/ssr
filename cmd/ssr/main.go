package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/81beastie/ssr/internal/clipboard"
	"github.com/81beastie/ssr/internal/daemon"
	"github.com/81beastie/ssr/internal/detector"
	"github.com/81beastie/ssr/internal/replacer"
)

const usage = `ssr — утилита для удаления секретов из текста (stdin → буфер обмена)

Читает текст из stdin, находит секреты (токены, пароли, ключи),
заменяет их на плейсхолдеры вида <ssr token>, <ssr password>
и кладёт результат в буфер обмена.

Использование:
  cat file.txt | ssr                 # stdin → буфер (по умолчанию)
  cat file.txt | ssr -s              # stdin → stdout
  ssr -f file.txt                    # файл → буфер
  ssr -f file.txt -s                 # файл → stdout
  ssr -f file.txt -o .ssr            # файл → file.txt.ssr (рядом)
  ssr -f file.txt -o clean.txt       # файл → указанный файл
  echo $TOKEN | ssr -t 5m            # буфер умрёт через 5 минут

Флаги:
  -f, --file string      читать текст из файла вместо stdin
  -o, --out string       записать результат в файл ('.ssr' — рядом с исходным)
  -s, --stdout           вывести очищенный текст в stdout, не трогая буфер
  -n, --dry-run          отчёт о найденных секретах, ничего не менять
  -b, --backend string   принудительный бэкенд буфера: x11, wayland, win32, mac, osc52
  -t, --timeout duration максимальное время жизни демона-держателя буфера (напр. 5m)
  -v, --version          показать версию и выйти
  -h, --help             эта справка

Плейсхолдеры:
  <ssr token>         один секрет этого типа в тексте
  <ssr token:2>       несколько разных секретов нумеруются
  одинаковые значения получают одинаковый плейсхолдер

Особенности:
  на Linux (X11/Wayland) ssr оставляет фоновый демон, держащий буфер;
  демон умирает сам, когда вы скопируете что-то другое (или по --timeout)
  на Windows и macOS буфер системный — процесс завершается сразу

Пример:
  $ echo "password=hunter2" | ssr
  $ # Ctrl+V → password=<ssr password>
`

type options struct {
	stdout  bool
	dryRun  bool
	version bool
	help    bool
	backend string
	timeout time.Duration
	file    string
	out     string
}

type clipboardServer interface {
	Serve(config daemon.Config) (int, error)
}

type daemonServer struct{}

func (daemonServer) Serve(config daemon.Config) (int, error) {
	return daemon.Serve(config)
}

type app struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	goos       string
	env        map[string]string
	serve      clipboardServer
	newBackend func(clipboard.Options) (clipboard.Clipboard, error)
}

func main() {
	a := &app{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		goos:   runtime.GOOS,
		env:    envMap(),
		serve:  daemonServer{},
	}

	code := 0
	if len(os.Args) > 1 && os.Args[1] == "--serve" {
		if err := daemon.RunServe(serveTimeout(os.Args[2:])); err != nil {
			fmt.Fprintln(os.Stderr, err)
			code = 1
		}
		os.Exit(code)
	}

	if err := a.run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ssr:", err)
		os.Exit(1)
	}
}

func envMap() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		if key, value, ok := strings.Cut(entry, "="); ok {
			env[key] = value
		}
	}
	return env
}

func serveTimeout(args []string) time.Duration {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.Duration("timeout", 0, "maximum lifetime of the clipboard holder")
	_ = fs.Parse(args)
	d, err := time.ParseDuration(fs.Lookup("timeout").Value.String())
	if err != nil {
		return 0
	}
	return d
}

func (a *app) run(args []string) error {
	opts, err := parseFlags(args)
	if err != nil {
		return err
	}

	if opts.help {
		printUsage(a.stdout)
		return nil
	}

	if opts.version {
		fmt.Fprintf(a.stdout, "ssr %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return nil
	}

	text, err := readInput(a.stdin, opts.file)
	if err != nil {
		return err
	}

	findings := detector.New().Detect(text)

	if opts.dryRun {
		return reportFindings(a.stdout, findings)
	}

	result := replacer.New().Replace(text, findings)

	if opts.out != "" {
		return writeOutput(opts.out, opts.file, result)
	}

	if opts.stdout {
		_, err := io.WriteString(a.stdout, result)
		return err
	}

	return copyToClipboard(a, result, opts.backend, opts.timeout)
}

func parseFlags(args []string) (options, error) {
	var opts options
	fs := flag.NewFlagSet("ssr", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&opts.stdout, "s", false, "")
	fs.BoolVar(&opts.stdout, "stdout", false, "")
	fs.BoolVar(&opts.dryRun, "n", false, "")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "")
	fs.BoolVar(&opts.version, "v", false, "")
	fs.BoolVar(&opts.version, "version", false, "")
	fs.BoolVar(&opts.help, "h", false, "")
	fs.BoolVar(&opts.help, "help", false, "")
	fs.StringVar(&opts.backend, "b", "", "")
	fs.StringVar(&opts.backend, "backend", "", "")
	fs.DurationVar(&opts.timeout, "t", 0, "")
	fs.DurationVar(&opts.timeout, "timeout", 0, "")
	fs.StringVar(&opts.file, "f", "", "")
	fs.StringVar(&opts.file, "file", "", "")
	fs.StringVar(&opts.out, "o", "", "")
	fs.StringVar(&opts.out, "out", "", "")
	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	return opts, nil
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, usage)
}

func readInput(stdin io.Reader, filePath string) (string, error) {
	if filePath == "" {
		input, err := io.ReadAll(stdin)
		if err != nil {
			return "", fmt.Errorf("не удалось прочитать stdin: %w", err)
		}
		return string(input), nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("файл не найден: %s", filePath)
		}
		return "", fmt.Errorf("не удалось прочитать файл %s: %w", filePath, err)
	}
	return string(data), nil
}

func writeOutput(outputPath, inputPath, text string) error {
	if outputPath == ".ssr" {
		if inputPath == "" {
			return fmt.Errorf("нельзя писать рядом с stdin — укажите имя файла: -o <путь>")
		}
		outputPath = inputPath + ".ssr"
	}

	if err := os.WriteFile(outputPath, []byte(text), 0o600); err != nil {
		return fmt.Errorf("не удалось записать файл %s: %w", outputPath, err)
	}
	return nil
}

func reportFindings(w io.Writer, findings []detector.Finding) error {
	if len(findings) == 0 {
		_, err := fmt.Fprintln(w, "ssr: секретов не найдено")
		return err
	}
	if _, err := fmt.Fprintf(w, "ssr: найдено секретов: %d\n", len(findings)); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, f := range findings {
		if seen[f.Value] {
			continue
		}
		seen[f.Value] = true
		if _, err := fmt.Fprintf(w, "  [%s] %s\n", f.Type, f.Value); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) backend(options clipboard.Options) (clipboard.Clipboard, error) {
	if a.newBackend != nil {
		return a.newBackend(options)
	}
	return clipboard.New(options)
}

func copyToClipboard(a *app, text, backendName string, timeout time.Duration) error {
	if backendName == "osc52" {
		return (&clipboard.OSC52{Writer: a.stdout}).Copy(text)
	}

	if backendName != "" {
		backend, err := a.backend(clipboard.Options{Override: backendName})
		if err != nil {
			return err
		}
		return backend.Copy(text)
	}

	if a.goos == "linux" {
		_, err := a.serve.Serve(daemon.Config{
			Display: a.env["DISPLAY"],
			Text:    text,
			Timeout: timeout,
		})
		return err
	}

	backend, err := a.backend(clipboard.Options{GOOS: a.goos})
	if err != nil {
		return err
	}
	return backend.Copy(text)
}
