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
	green = "\x1b[32m"
	red   = "\x1b[31m"
	reset = "\x1b[0m"
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
	if r.isVerbose() {
		r.writeConsole(green, "INFO", message)
		return
	}
	if !r.spinner.Enabled() {
		r.writeConsole(green, "INFO", message)
		return
	}

	r.spinner.Suffix = " " + message
	r.spinner.Start()
	r.started = true
}

func (r *Reporter) Printf(format string, args ...any) {
	r.Infof(format, args...)
}

func (r *Reporter) Infof(format string, args ...any) {
	r.writeMessages("INFO", green, fmt.Sprintf(format, args...))
}

func (r *Reporter) Errorf(format string, args ...any) {
	r.writeMessages("ERROR", red, fmt.Sprintf(format, args...))
}

func (r *Reporter) Success(message string) {
	r.finish("INFO", green, "✓ "+message)
}

func (r *Reporter) Fail(message string) {
	r.finish("ERROR", red, "✗ "+message)
}

func (r *Reporter) Write(data []byte) (int, error) {
	r.Infof("%s", string(data))
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
