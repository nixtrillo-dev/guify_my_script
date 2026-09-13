package logger

import (
	"fmt"
	"strings"
)

// LogLevel defines the severity threshold for package communications.
// Integer values are explicitly assigned to maintain backward compatibility:
// 0 - SILENT (LevelNone)
// 1 - FATAL (LevelFatal)
// 2 - ERROR (LevelError)
// 3 - WARNING (LevelWarn)
// 4 - INFO (LevelInfo)
// 5 - TRACE/DEBUG (LevelDebug)
//
// LevelData uses value 99 as a special out-of-band value to distinguish it from ERROR.
// The baseLogger handles this specially to provide the unique behavior:
// - When DATA mode (99) is configured: only FATAL(1), ERROR(2), and DATA(99) are shown
// - When DATA messages are logged with other modes: treated as INFO level (4)
type LogLevel int

const (
	LevelNone  LogLevel = 0  // Completely silent
	LevelFatal LogLevel = 1  // Crashing/unrecoverable states
	LevelError LogLevel = 2  // Action failed but application survives
	LevelWarn  LogLevel = 3  // Unideal state, fallback used, or soft issues
	LevelInfo  LogLevel = 4  // High-level lifecycle milestones
	LevelDebug LogLevel = 5  // Granular, step-by-step diagnostic telemetry (TRACE)
	LevelData  LogLevel = 99 // Data logging mode - special value, treated as ERROR when DATA is selected, INFO otherwise
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
	case "DATA":
		return LevelData
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
	effectiveMessageLevel := messageLevel
	effectiveConfiguredLevel := configuredLevel

	// Special handling for DATA mode (LevelData = 99):
	// When configuredLevel is DATA (99), treat it as ERROR level (2) for comparison purposes
	// This ensures only FATAL(1), ERROR(2), and DATA(99) are shown in DATA mode
	if configuredLevel == LevelData {
		effectiveConfiguredLevel = LevelError
		// DATA messages in DATA mode should always show
		if messageLevel == LevelData {
			effectiveMessageLevel = LevelError
		}
	}

	// Special handling for DATA messages:
	// When DATA messages are logged with non-DATA modes, treat them as INFO level (4)
	// This ensures DATA messages appear whenever INFO would appear
	if messageLevel == LevelData && configuredLevel != LevelData {
		effectiveMessageLevel = LevelInfo
	}

	if effectiveConfiguredLevel >= effectiveMessageLevel && configuredLevel != LevelNone {
		fmt.Printf(prefix+format+"\n", args...)
	}
}

// Clean, exported wrappers that abstract away the repetitive levels and string prefixes
func Debug(lvl LogLevel, format string, args ...any) {
	baseLogger(lvl, LevelDebug, "[DEBUG] ", format, args...)
}
func Info(lvl LogLevel, format string, args ...any) { baseLogger(lvl, LevelInfo, "", format, args...) }
func Data(lvl LogLevel, format string, args ...any) { baseLogger(lvl, LevelData, "", format, args...) }
func Warn(lvl LogLevel, format string, args ...any) {
	baseLogger(lvl, LevelWarn, "[WARN] ", format, args...)
}
func Error(lvl LogLevel, format string, args ...any) {
	baseLogger(lvl, LevelError, "[ERROR] ", format, args...)
}
func Fatal(lvl LogLevel, format string, args ...any) {
	baseLogger(lvl, LevelError, "[FATAL] ", format, args...)
}
