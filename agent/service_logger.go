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

type gsSummarizationRequest struct {
	text        string
	prefix      string // For adding emoji/icons
	placeholder string // What to show while summarizing
	callback    func(string)
}

type GsLogger struct {
	debugMode bool
	isLoading bool
	spinner   *spinner.Spinner
	client    *anthropic.Client

	// Channels for handling async operations
	summarizationQueue chan gsSummarizationRequest

	// Mutex for thread-safe console output
	mu sync.Mutex

	// Track the current line for replacements
	currentLine string
}

// ANSI escape codes
const (
	gsLoggerClearLine    = "\r\033[K"
	gsLoggerMoveUpOnce   = "\033[1A"
	gsLoggerMoveDownOnce = "\033[1B"
)

// Color setup
var (
	gsLoggerEmeraldColor = color.New(color.FgHiGreen)
	gsLoggerYellowColor  = color.New(color.FgHiYellow)
	gsLoggerRedColor     = color.New(color.FgHiRed)
	gsLoggerWhiteColor   = color.New(color.FgWhite)
)

func NewGsLogger(debugMode bool, client *anthropic.Client) *GsLogger {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Color("cyan")

	logger := &GsLogger{
		debugMode:          debugMode,
		spinner:           s,
		client:            client,
		summarizationQueue: make(chan gsSummarizationRequest, 100),
	}

	// Start background summarization worker
	go logger.summarizationWorker()

	return logger
}

func (l *GsLogger) Info(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hideSpinner()
	gsLoggerEmeraldColor.Printf(format, args...)
	l.showSpinner()
}

func (l *GsLogger) Debug(format string, args ...interface{}) {
	if !l.debugMode {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.hideSpinner()
	gsLoggerYellowColor.Printf(format, args...)
	l.showSpinner()
}

func (l *GsLogger) Error(format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hideSpinner()
	gsLoggerRedColor.Printf(format, args...)
	l.showSpinner()
}

func (l *GsLogger) AgentMessage(msg string) {
	// Queue summarization of the agent's message
	l.queueSummarization(gsSummarizationRequest{
		text:        msg,
		prefix:      "💭",
		placeholder: "Agent is thinking...",
		callback: func(summary string) {
			l.mu.Lock()
			defer l.mu.Unlock()

			l.hideSpinner()
			l.replaceLine(gsLoggerWhiteColor.Sprintf("%s %s", "💭", summary))
			l.showSpinner()
		},
	})

	// Show immediate feedback
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hideSpinner()
	l.replaceLine(gsLoggerWhiteColor.Sprintf("💭 Agent is thinking..."))
	l.showSpinner()
}

func (l *GsLogger) ToolCall(name, input string) {
	// Queue summarization
	l.queueSummarization(gsSummarizationRequest{
		text:        fmt.Sprintf("Tool Call: %s\nInput: %s", name, input),
		prefix:      "🔧",
		placeholder: fmt.Sprintf("%s: Processing...", name),
		callback: func(summary string) {
			l.mu.Lock()
			defer l.mu.Unlock()

			l.hideSpinner()
			l.replaceLine(gsLoggerWhiteColor.Sprintf("🔧 %s: %s", name, summary))
			l.showSpinner()
		},
	})

	// Show immediate feedback
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hideSpinner()
	l.replaceLine(gsLoggerWhiteColor.Sprintf("🔧 %s: Processing...", name))
	l.showSpinner()
}

func (l *GsLogger) ToolResult(name, result string, isError bool) {
	prefix := "✅"
	if isError {
		prefix = "❌"
	}

	// Queue summarization
	l.queueSummarization(gsSummarizationRequest{
		text:        result,
		prefix:      prefix,
		placeholder: fmt.Sprintf("%s: Processing result...", name),
		callback: func(summary string) {
			l.mu.Lock()
			defer l.mu.Unlock()

			l.hideSpinner()
			l.replaceLine(gsLoggerWhiteColor.Sprintf("%s %s: %s", prefix, name, summary))
			l.showSpinner()
		},
	})

	// Show immediate feedback
	l.mu.Lock()
	defer l.mu.Unlock()
	l.hideSpinner()
	l.replaceLine(gsLoggerWhiteColor.Sprintf("⏳ %s: Processing result...", name))
	l.showSpinner()
}

// Private helper methods

func (l *GsLogger) hideSpinner() {
	if l.isLoading {
		l.spinner.Stop()
		l.isLoading = false
	}
}

func (l *GsLogger) showSpinner() {
	if !l.isLoading {
		l.spinner.Start()
		l.isLoading = true
	}
}

func (l *GsLogger) replaceLine(text string) {
	// Clear the current line
	fmt.Print(gsLoggerClearLine)
	
	// If we had a previous line, move up and clear it
	if l.currentLine != "" {
		fmt.Print(gsLoggerMoveUpOnce + gsLoggerClearLine)
	}

	// Print the new text and move down
	fmt.Println(text)
	
	// Store the current line
	l.currentLine = text
}

func (l *GsLogger) queueSummarization(req gsSummarizationRequest) {
	l.summarizationQueue <- req
}

func (l *GsLogger) summarizationWorker() {
	for req := range l.summarizationQueue {
		summary := l.summarizeText(req.text)
		req.callback(summary)
	}
}

func (l *GsLogger) summarizeText(text string) string {
	// Skip summarization for short text
	if len(text) < 100 {
		return text
	}

	prompt := fmt.Sprintf(
		"Please summarize the following text in a brief, user-friendly way (max 100 chars):\n\n%s",
		text,
	)

	message, err := l.client.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5SonnetLatest,
		MaxTokens: int64(100),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})

	if err != nil {
		return fmt.Sprintf("(Failed to summarize: %s)", strings.Split(text, "\n")[0])
	}

	if len(message.Content) > 0 {
		return message.Content[0].Text
	}

	return fmt.Sprintf("(Failed to summarize: %s)", strings.Split(text, "\n")[0])
}