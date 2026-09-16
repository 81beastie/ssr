package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	"github.com/81beastie/ssr/internal/clipboard"
	"github.com/81beastie/ssr/internal/daemon"
	"github.com/81beastie/ssr/internal/detector"
	"github.com/81beastie/ssr/internal/replacer"
)

const version = "0.1.0"

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

func printUsage() {
	fmt.Print(usage)
}

func registerBool(short, long string, value bool, description string) *bool {
	shortFlag := flag.Bool(short, value, description)
	flag.BoolVar(shortFlag, long, value, description)
	return shortFlag
}

func registerString(short, long, value, description string) *string {
	shortFlag := flag.String(short, value, description)
	flag.StringVar(shortFlag, long, value, description)
	return shortFlag
}

func registerDuration(short, long string, value time.Duration, description string) *time.Duration {
	shortFlag := flag.Duration(short, value, description)
	flag.DurationVar(shortFlag, long, value, description)
	return shortFlag
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--serve" {
		timeout := time.Duration(0)
		serveFlags := flag.NewFlagSet("serve", flag.ExitOnError)
		serveFlags.Duration("timeout", 0, "maximum lifetime of the clipboard holder")
		_ = serveFlags.Parse(os.Args[2:])
		if lookup := serveFlags.Lookup("timeout"); lookup != nil {
			if d, err := time.ParseDuration(lookup.Value.String()); err == nil {
				timeout = d
			}
		}
		if err := daemon.RunServe(timeout); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	stdoutFlag := registerBool("s", "stdout", false, "print redacted text to stdout instead of copying")
	dryRunFlag := registerBool("n", "dry-run", false, "report findings without redacting or copying")
	versionFlag := registerBool("v", "version", false, "print version and exit")
	helpFlag := registerBool("h", "help", false, "print usage and exit")
	backendFlag := registerString("b", "backend", "", "force clipboard backend: x11, wayland, win32, mac, osc52")
	timeoutFlag := registerDuration("t", "timeout", 0, "maximum clipboard holder lifetime (e.g. 5m)")
	inputFlag := registerString("f", "file", "", "read text from file instead of stdin")
	outputFlag := registerString("o", "out", "", "write result to file ('.ssr' means alongside input file)")
	flag.Parse()

	if *helpFlag {
		printUsage()
		return
	}

	if *versionFlag {
		fmt.Printf("ssr %s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
		return
	}

	text, err := readInput(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ssr:", err)
		os.Exit(1)
	}

	findings := detector.New().Detect(text)

	if *dryRunFlag {
		reportFindings(findings)
		return
	}

	result := replacer.New().Replace(text, findings)

	if *outputFlag != "" {
		if err := writeOutput(*outputFlag, *inputFlag, result); err != nil {
			fmt.Fprintln(os.Stderr, "ssr:", err)
			os.Exit(1)
		}
		return
	}

	if *stdoutFlag {
		fmt.Print(result)
		return
	}

	if err := copyToClipboard(result, *backendFlag, *timeoutFlag); err != nil {
		fmt.Fprintln(os.Stderr, "ssr:", err)
		os.Exit(1)
	}
}

func readInput(filePath string) (string, error) {
	if filePath == "" {
		input, err := io.ReadAll(os.Stdin)
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

func reportFindings(findings []detector.Finding) {
	if len(findings) == 0 {
		fmt.Println("ssr: no secrets found")
		return
	}
	fmt.Printf("ssr: found %d secret(s):\n", len(findings))
	seen := map[string]bool{}
	for _, f := range findings {
		if seen[f.Value] {
			continue
		}
		seen[f.Value] = true
		fmt.Printf("  [%s] %s\n", f.Type, f.Value)
	}
}

func copyToClipboard(text, backendName string, timeout time.Duration) error {
	if backendName == "osc52" {
		return (&clipboard.OSC52{}).Copy(text)
	}

	if backendName != "" {
		backend, err := clipboard.New(clipboard.Options{Override: backendName})
		if err != nil {
			return err
		}
		return backend.Copy(text)
	}

	if runtime.GOOS == "linux" {
		_, err := daemon.Serve(daemon.Config{
			Display: os.Getenv("DISPLAY"),
			Text:    text,
			Timeout: timeout,
		})
		return err
	}

	backend, err := clipboard.New(clipboard.Options{GOOS: runtime.GOOS})
	if err != nil {
		return err
	}
	return backend.Copy(text)
}
