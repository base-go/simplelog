package simplelog

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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

// GinMiddleware returns a Gin middleware function for logging HTTP requests
func (l *Logger) GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		if raw != "" {
			path = path + "?" + raw
		}

		ua := c.Request.UserAgent()
		os, browser := parseUserAgent(ua)

		l.log(INFO, "Request: %s %s %d %s %s %s %s %s",
			c.Request.Method,
			path,
			c.Writer.Status(),
			c.ClientIP(),
			latency.String(),
			os,
			browser,
			c.Errors.String(),
		)
	}
}

// parseUserAgent extracts OS and browser information from the user agent string
func parseUserAgent(ua string) (os, browser string) {
	ua = strings.ToLower(ua)
	// OS detection
	switch {
	case strings.Contains(ua, "windows"):
		os = "Windows"
	case strings.Contains(ua, "mac os"):
		os = "macOS"
	case strings.Contains(ua, "linux"):
		os = "Linux"
	case strings.Contains(ua, "android"):
		os = "Android"
	case strings.Contains(ua, "ios"):
		os = "iOS"
	default:
		os = "Unknown"
	}
	// Browser detection
	switch {
	case strings.Contains(ua, "firefox"):
		browser = "Firefox"
	case strings.Contains(ua, "chrome"):
		browser = "Chrome"
	case strings.Contains(ua, "safari"):
		browser = "Safari"
	case strings.Contains(ua, "opera"):
		browser = "Opera"
	case strings.Contains(ua, "edge"):
		browser = "Edge"
	case strings.Contains(ua, "msie") || strings.Contains(ua, "trident"):
		browser = "Internet Explorer"
	default:
		browser = "Unknown"
	}
	return
}
