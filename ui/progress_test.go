package ui

import (
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/briandowns/spinner"
)

func TestReporterWithoutTerminal(t *testing.T) {
	console, err := os.CreateTemp(t.TempDir(), "console-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer console.Close()

	logFile, err := os.CreateTemp(t.TempDir(), "reporter-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer logFile.Close()

	oldStderr := os.Stderr
	os.Stderr = console
	defer func() { os.Stderr = oldStderr }()

	reporter := &Reporter{
		spinner: spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriterFile(console)),
		file:    logFile,
		logger:  log.New(logFile, "", 0),
	}
	reporter.Start("Checking application")
	reporter.Fail("test failure")

	if reporter.started {
		t.Fatal("spinner marked as started without a terminal")
	}
	if _, err := console.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	output, err := io.ReadAll(console)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(output), "Checking application") || !strings.Contains(string(output), "test failure") {
		t.Fatalf("console output = %q, want start and failure messages", output)
	}
}
