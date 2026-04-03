// Package logger provides level-aware logging wrapper.
package logger

import (
	"log"
	"sync/atomic"
)

// Level represents log level
type Level int32

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

var currentLevel atomic.Int32

// SetLevel sets the global log level
func SetLevel(level string) {
	var l Level
	switch level {
	case "debug":
		l = LevelDebug
	case "info":
		l = LevelInfo
	case "warn":
		l = LevelWarn
	case "error":
		l = LevelError
	default:
		l = LevelInfo
	}
	currentLevel.Store(int32(l))
}

// Debugf logs at debug level
func Debugf(format string, v ...interface{}) {
	if Level(currentLevel.Load()) <= LevelDebug {
		log.Printf(format, v...)
	}
}

// Infof logs at info level
func Infof(format string, v ...interface{}) {
	if Level(currentLevel.Load()) <= LevelInfo {
		log.Printf(format, v...)
	}
}

// Errorf logs at error level
func Errorf(format string, v ...interface{}) {
	if Level(currentLevel.Load()) <= LevelError {
		log.Printf(format, v...)
	}
}
