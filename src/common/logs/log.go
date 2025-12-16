// Package logs provides a common logging facility for ldf applications.
// It supports output to stdout or systemd journald based on configuration.
package logs

import (
	"io"
	"os"
	"os/exec"

	"github.com/charmbracelet/log"
)

// LogOutput defines the output destination for logs
type LogOutput string

const (
	// OutputStdout sends logs to standard output
	OutputStdout LogOutput = "stdout"
	// OutputJournald sends logs to systemd journald
	OutputJournald LogOutput = "journald"
	// OutputAuto automatically selects journald if available, otherwise stdout
	OutputAuto LogOutput = "auto"
)

// Logger wraps the charm log.Logger with additional configuration
type Logger struct {
	*log.Logger
	output LogOutput
}

// Config holds the configuration for the logger
type Config struct {
	// Output specifies where logs should be sent (stdout, journald, auto)
	Output LogOutput
	// Level sets the minimum log level (debug, info, warn, error)
	Level string
	// Prefix sets a prefix for all log messages
	Prefix string
}

// DefaultConfig returns a default configuration
func DefaultConfig() Config {
	return Config{
		Output: OutputAuto,
		Level:  "info",
		Prefix: "",
	}
}

// journaldAvailable checks if systemd-journald is available on the system
func journaldAvailable() bool {
	// Check if systemd-cat command exists (used to send to journald)
	_, err := exec.LookPath("systemd-cat")
	if err != nil {
		return false
	}
	// Check if journald socket exists
	if _, err := os.Stat("/run/systemd/journal/socket"); err != nil {
		return false
	}
	return true
}

// parseLevel converts a string level to log.Level
func parseLevel(level string) log.Level {
	switch level {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	default:
		return log.InfoLevel
	}
}

// New creates a new Logger with the given configuration
func New(cfg Config) *Logger {
	var writer io.Writer
	var output LogOutput

	// Determine output destination
	switch cfg.Output {
	case OutputJournald:
		if journaldAvailable() {
			writer = newJournaldWriter()
			output = OutputJournald
		} else {
			writer = os.Stdout
			output = OutputStdout
		}
	case OutputAuto:
		if journaldAvailable() {
			writer = newJournaldWriter()
			output = OutputJournald
		} else {
			writer = os.Stdout
			output = OutputStdout
		}
	default:
		writer = os.Stdout
		output = OutputStdout
	}

	logger := log.NewWithOptions(writer, log.Options{
		Level:           parseLevel(cfg.Level),
		Prefix:          cfg.Prefix,
		ReportTimestamp: true,
		ReportCaller:    false,
	})

	return &Logger{
		Logger: logger,
		output: output,
	}
}

// NewDefault creates a new Logger with default configuration
func NewDefault() *Logger {
	return New(DefaultConfig())
}

// Output returns the current output destination
func (l *Logger) Output() LogOutput {
	return l.output
}

// journaldWriter implements io.Writer for journald
type journaldWriter struct {
	identifier string
}

// newJournaldWriter creates a writer that sends output to journald
func newJournaldWriter() *journaldWriter {
	return &journaldWriter{
		identifier: "ldf",
	}
}

// Write implements io.Writer for journald
// It uses systemd-cat to send messages to journald
func (w *journaldWriter) Write(p []byte) (n int, err error) {
	cmd := exec.Command("systemd-cat", "-t", w.identifier)
	cmd.Stdin = nil

	stdin, err := cmd.StdinPipe()
	if err != nil {
		// Fallback to stdout if journald write fails
		return os.Stdout.Write(p)
	}

	if err := cmd.Start(); err != nil {
		return os.Stdout.Write(p)
	}

	n, err = stdin.Write(p)
	stdin.Close()

	if err := cmd.Wait(); err != nil {
		// Message was written but journald had an issue
		// Return success since data was sent
	}

	return n, nil
}
