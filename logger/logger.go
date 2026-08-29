package logger

import (
	"fmt"
	"strings"
)

// LogLevel defines the severity threshold for package communications.
type LogLevel int

const (
	LevelNone  LogLevel = iota // Completely silent
	LevelFatal                 // Crashing/unrecoverable states
	LevelError                 // Action failed but application survives
	LevelWarn                  // Unideal state, fallback used, or soft issues
	LevelInfo                  // High-level lifecycle milestones
	LevelDebug                 // Granular, step-by-step diagnostic telemetry
)

// ParseLogLevel gracefully converts shell string inputs into our internal hierarchy.
func ParseLogLevel(level string) LogLevel {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG", "TRACE":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "FATAL":
		return LevelFatal
	case "NONE", "SILENT", "FALSE":
		return LevelNone
	default:
		return LevelInfo
	}
}

// baseLogger evaluates if the message meets the user's requested severity threshold.
// Kept lowercase because it is an internal helper that other packages shouldn't call directly.
func baseLogger(configuredLevel, messageLevel LogLevel, prefix, format string, args ...any) {
	if configuredLevel >= messageLevel && configuredLevel != LevelNone {
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// Clean, exported wrappers that abstract away the repetitive levels and string prefixes
func Debug(lvl LogLevel, format string, args ...any) { baseLogger(lvl, LevelDebug, "[DEBUG] ", format, args...) }
func Info(lvl LogLevel, format string, args ...any)  { baseLogger(lvl, LevelInfo, "", format, args...) }
func Warn(lvl LogLevel, format string, args ...any)  { baseLogger(lvl, LevelWarn, "[WARN] ", format, args...) }
func Error(lvl LogLevel, format string, args ...any) { baseLogger(lvl, LevelError, "[ERROR] ", format, args...) }
func Fatal(lvl LogLevel, format string, args ...any) { baseLogger(lvl, LevelError, "[FATAL] ", format, args...) }
