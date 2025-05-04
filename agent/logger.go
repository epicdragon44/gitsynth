package main

import (
	"fmt"
)

// Logger manages logging output based on debug flag
type Logger struct {
	debugMode   bool
	currentLine string // Track the current ephemeral line for replacements
}

// NewLogger creates a new Logger with the provided debug state
func NewLogger(debugMode bool) *Logger {
	return &Logger{
		debugMode:   debugMode,
		currentLine: "",
	}
}

// Debug logs a message only when debug mode is enabled
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debugMode {
		fmt.Printf(format, args...)
		// Reset ephemeral line tracking after a permanent log
		l.currentLine = ""
	}
}

// Info logs a message regardless of debug mode
func (l *Logger) Info(format string, args ...interface{}) {
	fmt.Printf(format, args...)
	// Reset ephemeral line tracking after a permanent log
	l.currentLine = ""
}

// replaceLine replaces the previous ephemeral log with a new one
func (l *Logger) replaceLine(text string) {
	// If we don't have a current ephemeral line, just print normally
	if l.currentLine == "" {
		fmt.Println(text)
		l.currentLine = text
		return
	}

	// Clear the current line
	fmt.Print(clearLine)

	// Move up and clear the previous ephemeral log
	fmt.Print(moveUpOnce + clearLine)

	// Print the new text
	fmt.Println(text)

	// Store the current ephemeral line
	l.currentLine = text
}

// ToolCall logs a tool call in debug mode
func (l *Logger) ToolCall(name string, input interface{}) {
	if l.debugMode {
		l.replaceLine(fmt.Sprintf("\u001b[92mTool\u001b[0m: %s(%v)", name, input))
	}
}

// ToolResult logs a tool result in debug mode
func (l *Logger) ToolResult(name string, result string, isError bool) {
	if l.debugMode {
		prefix := "\u001b[92mResult\u001b[0m"
		if isError {
			prefix = "\u001b[91mError\u001b[0m"
		}
		l.replaceLine(fmt.Sprintf("%s (%s): %s", prefix, name, result))
	}
}

// Error logs an error message regardless of debug mode
func (l *Logger) Error(format string, args ...interface{}) {
	fmt.Printf("\u001b[91mError\u001b[0m: "+format+"\n", args...)
	// Reset ephemeral line tracking after a permanent log
	l.currentLine = ""
}

// AgentMessage logs a message from the AI agent
func (l *Logger) AgentMessage(message string) {
	if l.debugMode {
		l.replaceLine(fmt.Sprintf("\u001b[93mGitSynth\u001b[0m: %s", message))
	}
}
