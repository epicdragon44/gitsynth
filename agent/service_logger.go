package main

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
)

// EphemeralLogEntry represents a log entry that should be shown after summarization
type EphemeralLogEntry struct {
	text     string      // Original text to summarize
	emoji    string      // Icon/emoji to prefix the message with
	metadata string      // Additional context (e.g., tool name)
	isError  bool        // For error status in tool results
	callback chan string // Channel to receive the summarized text
}

// GsLogger is a logger that handles permanent and ephemeral logs with summarization
type GsLogger struct {
	debugMode bool
	client    *anthropic.Client
	spinner   *spinner.Spinner

	// Mutex for thread-safe console output
	mu sync.Mutex

	// Channels for handling async operations
	ephemeralQueue chan EphemeralLogEntry

	// For tracking display state
	hasEphemeralLog bool // Whether we currently have an ephemeral message displayed
	maxLineLength   int  // Maximum length for a single line before truncation
}

// ANSI escape codes for terminal control
const (
	clearLine  = "\r\033[K"
	moveUpOnce = "\033[1A"
)

// Colors for different log types
var (
	infoColor   = color.New(color.FgHiGreen)
	debugColor  = color.New(color.FgHiYellow)
	errorColor  = color.New(color.FgHiRed)
	normalColor = color.New(color.FgWhite)
)

// NewGsLogger creates a new enhanced logger
func NewGsLogger(debugMode bool, client *anthropic.Client) *GsLogger {
	// Configure spinner
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Color("cyan")

	logger := &GsLogger{
		debugMode:       debugMode,
		client:          client,
		spinner:         s,
		ephemeralQueue:  make(chan EphemeralLogEntry, 100),
		hasEphemeralLog: false,
		maxLineLength:   120, // Reasonable default for most terminals
	}

	// Start background workers
	go logger.ephemeralLogProcessor()

	// Start spinner initially
	logger.spinner.Start()

	return logger
}

// Info logs a permanent informational message
func (l *GsLogger) Info(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Stop spinner, clear any ephemeral log
	l.clearDisplay()

	// Generate the formatted message
	message := fmt.Sprintf(format, args...)

	// Ensure it doesn't contain newlines
	message = l.sanitizeMessage(message)

	// Print permanent message
	infoColor.Print(message)

	// Reset ephemeral log state and restart spinner
	l.hasEphemeralLog = false
	l.spinner.Start()
}

// Debug logs a permanent debug message (only in debug mode)
func (l *GsLogger) Debug(format string, args ...interface{}) {
	if !l.debugMode {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Stop spinner, clear any ephemeral log
	l.clearDisplay()

	// Generate the formatted message
	message := fmt.Sprintf(format, args...)

	// Ensure it doesn't contain newlines
	message = l.sanitizeMessage(message)

	// Print permanent message
	debugColor.Print(message)

	// Reset ephemeral log state and restart spinner
	l.hasEphemeralLog = false
	l.spinner.Start()
}

// Error logs a permanent error message
func (l *GsLogger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Stop spinner, clear any ephemeral log
	l.clearDisplay()

	// Generate the formatted message
	message := fmt.Sprintf("ERROR: "+format, args...)

	// Ensure it doesn't contain newlines
	message = l.sanitizeMessage(message)

	// Print permanent message
	errorColor.Print(message)

	// Reset ephemeral log state and restart spinner
	l.hasEphemeralLog = false
	l.spinner.Start()
}

// AgentMessage queues an agent message to be summarized and displayed
func (l *GsLogger) AgentMessage(msg string) {
	// Create channel for the summary callback
	callbackCh := make(chan string, 1)

	// Queue the message for summarization
	l.ephemeralQueue <- EphemeralLogEntry{
		text:     msg,
		emoji:    "💭",
		callback: callbackCh,
	}

	// Wait for summarization in a goroutine to avoid blocking
	go func() {
		summary := <-callbackCh
		l.showEphemeralLog("💭" + summary)
		close(callbackCh)
	}()
}

// ToolCall queues a tool call to be summarized and displayed
func (l *GsLogger) ToolCall(name, input string) {
	// Create channel for the summary callback
	callbackCh := make(chan string, 1)

	// Format the tool name by replacing underscores with spaces
	formattedName := strings.ReplaceAll(name, "_", " ")

	// Queue the message for summarization
	l.ephemeralQueue <- EphemeralLogEntry{
		text:     fmt.Sprintf("Tool Call: %s\nInput: %s", formattedName, input),
		emoji:    "🔧",
		metadata: formattedName,
		callback: callbackCh,
	}

	// Wait for summarization in a goroutine to avoid blocking
	go func() {
		summary := <-callbackCh
		l.showEphemeralLog(fmt.Sprintf("🔧 Tool Call: %s", summary))
		close(callbackCh)
	}()
}

// ToolResult queues a tool result to be summarized and displayed
func (l *GsLogger) ToolResult(name, result string, isError bool) {
	// Create channel for the summary callback
	callbackCh := make(chan string, 1)

	// Format the tool name by replacing underscores with spaces
	formattedName := strings.ReplaceAll(name, "_", " ")

	// Choose emoji based on error status
	emoji := "✅"
	if isError {
		emoji = "❌"
	}

	// Queue the message for summarization
	l.ephemeralQueue <- EphemeralLogEntry{
		text:     result,
		emoji:    emoji,
		metadata: formattedName,
		isError:  isError,
		callback: callbackCh,
	}

	// Wait for summarization in a goroutine to avoid blocking
	go func() {
		summary := <-callbackCh
		l.showEphemeralLog(fmt.Sprintf("%s Result: %s", emoji, summary))
		close(callbackCh)
	}()
}

// clearDisplay stops the spinner and clears any ephemeral log
// Must be called with the mutex locked
func (l *GsLogger) clearDisplay() {
	// Stop the spinner if it's active
	if l.spinner.Active() {
		l.spinner.Stop()
	}

	// Clear spinner line
	fmt.Print(clearLine)

	// If we have an ephemeral log, clear that exactly one line
	if l.hasEphemeralLog {
		fmt.Print(moveUpOnce + clearLine) // Move up and clear one line only
	}
}

// sanitizeMessage ensures a message is a single line with no line breaks
func (l *GsLogger) sanitizeMessage(message string) string {
	// Replace all newlines with spaces
	message = strings.ReplaceAll(message, "\n", " ")

	// Truncate if longer than max line length
	if len(message) > l.maxLineLength {
		message = message[:l.maxLineLength-3] + "..."
	}

	return message
}

// showEphemeralLog safely displays a log message, replacing any previous one
func (l *GsLogger) showEphemeralLog(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Sanitize the message - make it a single line, truncate if needed
	message = l.sanitizeMessage(message)

	// Clear current display
	l.clearDisplay()

	// Print the new ephemeral message
	normalColor.Println(message) // Println for a single line
	l.hasEphemeralLog = true

	// Restart spinner on the next line
	l.spinner.Start()
}

// ephemeralLogProcessor handles the summarization queue
func (l *GsLogger) ephemeralLogProcessor() {
	for entry := range l.ephemeralQueue {
		// Summarize the text
		summary := l.summarizeText(entry.text)

		// Send the summary through the callback channel
		entry.callback <- summary
	}
}

// summarizeText summarizes text using Anthropic's API
func (l *GsLogger) summarizeText(text string) string {
	// Skip summarization for short text
	if len(text) < 100 {
		return text
	}

	prompt := fmt.Sprintf(
		"Please summarize the following text in a brief, user-friendly way (max 150 chars). IMPORTANT: Use a single line with no line breaks:\n\n%s",
		text,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message, err := l.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5SonnetLatest,
		MaxTokens: int64(150),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		// Return a shortened version of the original text on error
		return fmt.Sprintf("(Summary failed: %s...)", l.sanitizeMessage(text)[:50])
	}

	if len(message.Content) > 0 {
		// Ensure the summary is sanitized
		return l.sanitizeMessage(message.Content[0].Text)
	}

	// Fallback to a simple truncation
	return l.sanitizeMessage(text)
}
