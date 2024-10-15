package simplelog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger is the main struct for the logging system
type Logger struct {
	level      LogLevel
	output     io.Writer
	file       *os.File
	mu         sync.Mutex
	timeFormat string
}

var (
	logFile     string
	maxFileSize int64 = 10 * 1024 * 1024 // 10MB
)

// New creates a new Logger instance
func New(level LogLevel, filename string) *Logger {
	logFile = filename
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	return &Logger{
		level:      level,
		output:     io.MultiWriter(os.Stdout, file),
		file:       file,
		timeFormat: "2006-01-02 15:04:05",
	}
}

func (l *Logger) log(level LogLevel, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Check file size and rotate if necessary
	if fi, err := l.file.Stat(); err == nil && fi.Size() > maxFileSize {
		l.rotateLog()
	}

	// Get caller information
	_, file, line, _ := runtime.Caller(2)

	// Format the log message
	msg := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] %s %s:%d: %s\n",
		time.Now().Format(l.timeFormat),
		levelToString(level),
		filepath.Base(file),
		line,
		msg)

	// Write to output
	fmt.Fprint(l.output, logEntry)
}

func (l *Logger) rotateLog() {
	l.file.Close()
	os.Rename(logFile, logFile+"."+time.Now().Format("2006-01-02-15-04-05"))
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		panic(err)
	}
	l.file = file
	l.output = io.MultiWriter(os.Stdout, file)
}

// Debug logs a debug-level message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DEBUG, format, args...)
}

// Info logs an info-level message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(INFO, format, args...)
}

// Warn logs a warn-level message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WARN, format, args...)
}

// Error logs an error-level message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ERROR, format, args...)
}

func levelToString(level LogLevel) string {
	switch level {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// SetMaxFileSize sets the maximum size of the log file before rotation
func (l *Logger) SetMaxFileSize(size int64) {
	maxFileSize = size
}

// SetTimeFormat sets the time format used in log entries
func (l *Logger) SetTimeFormat(format string) {
	l.timeFormat = format
}
