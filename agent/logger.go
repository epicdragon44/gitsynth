package main

import (
	"fmt"
	"strings"
)

// Logger manages logging output based on debug flag
type Logger struct {
	debugMode bool
}

// NewLogger creates a new Logger with the provided debug state
func NewLogger(debugMode bool) *Logger {
	return &Logger{
		debugMode: debugMode,
	}
}

// Debug logs a message only when debug mode is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debugMode {
		fmt.Printf(format, args...)
	}
}

// Info logs a message regardless of debug mode
func (l *Logger) Info(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// ToolCall logs a tool call in debug mode
func (l *Logger) ToolCall(name string, input interface{}) {
	if l.debugMode {
		fmt.Printf("\u001b[92mTool\u001b[0m: %s(%v)\n", name, input)
	}
}

// ToolResult logs a tool result in debug mode
func (l *Logger) ToolResult(name string, result string, isError bool) {
	if l.debugMode {
		prefix := "\u001b[92mResult\u001b[0m"
		if isError {
			prefix = "\u001b[91mError\u001b[0m"
		}
		fmt.Printf("%s (%s): %s\n", prefix, name, result)
	}
}

// Error logs an error message regardless of debug mode
func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Printf("\u001b[91mError\u001b[0m: "+format+"\n", args...)
}

// AgentMessage logs a message from the AI agent
func (l *Logger) AgentMessage(message string) {
	if strings.Contains(message, "[ALL DONE]") {
		fmt.Println("Done!")
	} else if l.debugMode {
		fmt.Printf("\u001b[93mGitSynth\u001b[0m: %s\n", message)
	}
}
