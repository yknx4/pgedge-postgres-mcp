//-------------------------------------------------------------------------
//
// pgEdge Natural Language Agent
//
// Copyright (c) 2025 - 2026, pgEdge, Inc.
// This software is released under The PostgreSQL License
//
//-------------------------------------------------------------------------

package chat

import (
	"os"
	"strings"
	"sync/atomic"
)

// LogLevel controls how chatty the LLM-tracing layer is. Read once at
// package init from PGEDGE_LLM_LOG_LEVEL ("none" / "info" / "debug" /
// "trace"; default "none"). Tests may override via SetLogLevel.
type LogLevel int32

const (
	LogLevelNone LogLevel = iota
	LogLevelInfo
	LogLevelDebug
	LogLevelTrace
)

// globalLogLevel is read by tracingRoundTripper.RoundTrip on every
// request. atomic.Int32 lets tests flip it safely.
var globalLogLevel atomic.Int32

func init() {
	globalLogLevel.Store(int32(parseLogLevel(os.Getenv("PGEDGE_LLM_LOG_LEVEL"))))
}

func parseLogLevel(s string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "info":
		return LogLevelInfo
	case "debug":
		return LogLevelDebug
	case "trace":
		return LogLevelTrace
	default:
		return LogLevelNone
	}
}

// GetLogLevel returns the current chat-package log level.
func GetLogLevel() LogLevel {
	return LogLevel(globalLogLevel.Load())
}

// SetLogLevel sets the chat-package log level. Intended for tests.
func SetLogLevel(l LogLevel) {
	globalLogLevel.Store(int32(l))
}
