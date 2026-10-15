package ui

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

const (
	blue   = "\x1b[1;34m"
	green  = "\x1b[32m"
	yellow = "\x1b[33m"
	red    = "\x1b[31m"
	dim    = "\x1b[2m"
	reset  = "\x1b[0m"
)

type Reporter struct {
	mu      sync.Mutex
	verbose *bool
	spinner *spinner.Spinner
	file    *os.File
	logger  *log.Logger
	started bool
}

func NewReporter(verbose *bool) (*Reporter, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return nil, fmt.Errorf("get user cache directory: %w", err)
	}

	logDir := filepath.Join(cacheDir, "kapellmeister", "logs")
	if err := os.MkdirAll(logDir, 0o750); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	logPath := filepath.Join(
		logDir,
		time.Now().Format("2006-01-02_15-04-05.000")+".log",
	)
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}

	s := spinner.New(
		spinner.CharSets[14],
		100*time.Millisecond,
		spinner.WithWriterFile(os.Stderr),
		spinner.WithHiddenCursor(true),
	)

	return &Reporter{
		verbose: verbose,
		spinner: s,
		file:    file,
		logger:  log.New(file, "", log.Ldate|log.Ltime|log.Lmicroseconds),
	}, nil
}

func (r *Reporter) Start(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.writeFile("INFO", message)
	if r.isVerbose() || !r.spinner.Enabled() {
		r.writeConsole(green, "INFO", message)
		return
	}

	r.spinner.Suffix = " " + message
	r.spinner.Start()
	if !r.spinner.Active() {
		r.writeConsole(green, "INFO", message)
		return
	}
	r.started = true
}

func (r *Reporter) Printf(format string, args ...any) {
	r.writeMessages("INFO", green, fmt.Sprintf(format, args...))
}

func (r *Reporter) Success(message string) {
	r.finish("INFO", green, "✓ "+message)
}

func (r *Reporter) Fail(message string) {
	r.finish("ERROR", red, "✗ "+message)
}

func (r *Reporter) Write(data []byte) (int, error) {
	r.writeMessages("INFO", green, string(data))
	return len(data), nil
}

func (r *Reporter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		r.spinner.Stop()
		r.started = false
	}

	return r.file.Close()
}

func (r *Reporter) writeMessages(level string, color string, message string) {
	// разбиваем лог на строки для файла и спиннера
	lines := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		r.mu.Lock()
		r.writeFile(level, line)
		if r.isVerbose() || !r.started {
			r.writeConsole(color, level, line)
		} else {
			r.spinner.Lock()
			r.spinner.Suffix = " " + line
			r.spinner.Unlock()
		}
		r.mu.Unlock()
	}
}

func (r *Reporter) finish(level string, color string, message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.writeFile(level, message)
	if r.isVerbose() || !r.started {
		r.writeConsole(color, level, message)
		return
	}

	r.spinner.FinalMSG = color + message + reset + "\n"
	r.spinner.Stop()
	r.started = false
}

func (r *Reporter) writeFile(level string, message string) {
	r.logger.Printf("[%s] %s", level, message)
}

func (r *Reporter) writeConsole(color string, level string, message string) {
	fmt.Fprintf(os.Stderr, "%s[%s] %s%s\n", color, level, message, reset)
}

func (r *Reporter) isVerbose() bool {
	return r.verbose != nil && *r.verbose
}

func (r *Reporter) Section(title string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopProgressLocked()
	r.writeFile("INFO", title)

	fmt.Fprintf(
		os.Stderr,
		"\n%s%s%s\n",
		blue,
		title,
		reset,
	)
}

func (r *Reporter) Field(label string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	text := fmt.Sprint(value)
	r.writeFile("INFO", fmt.Sprintf("%s: %s", label, text))

	fmt.Fprintf(
		os.Stderr,
		"  %s%-12s%s %s\n",
		dim,
		label,
		reset,
		text,
	)
}

func (r *Reporter) Passed(label string, detail string) {
	r.checkResult("PASS", green, "✓", label, detail)
}

func (r *Reporter) Failed(label string, detail string) {
	r.checkResult("FAIL", red, "✗", label, detail)
}

func (r *Reporter) Warning(label string, detail string) {
	r.checkResult("WARN", yellow, "!", label, detail)
}

func (r *Reporter) checkResult(level string, color string, symbol string, label string, detail string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.writeFile(
		level,
		fmt.Sprintf("%s: %s", label, detail),
	)

	fmt.Fprintf(
		os.Stderr,
		"  %s%s%s %-24s %s\n",
		color,
		symbol,
		reset,
		label,
		detail,
	)
}

func (r *Reporter) Complete(message string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stopProgressLocked()
	r.writeFile("PASS", message)

	fmt.Fprintf(
		os.Stderr,
		"\n%s✓ %s%s\n",
		green,
		message,
		reset,
	)
}

func (r *Reporter) stopProgressLocked() {
	if !r.started {
		return
	}

	r.spinner.FinalMSG = ""
	r.spinner.Stop()
	r.started = false
}
