package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ConflictChunk represents a git merge conflict chunk
type ConflictChunk struct {
	ID           int    `json:"id"`
	BaseCode     string `json:"base_code"`
	IncomingCode string `json:"incoming_code"`
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
}

// ValidateFileExists checks if a file exists and returns an error if it doesn't
func ValidateFileExists(path string) error {
	if path == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("file %s does not exist", path)
	}
	return nil
}

// ExecuteGitCommand runs a git command and returns its output
func ExecuteGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git command failed: %s\nStderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// FindConflictChunks identifies merge conflict chunks in a file's content
func FindConflictChunks(content string) ([]ConflictChunk, error) {
	lines := strings.Split(content, "\n")
	var chunks []ConflictChunk

	inConflict := false
	var currentChunk ConflictChunk
	var baseLines, incomingLines []string
	currentID := 0

	for i, line := range lines {
		lineNum := i + 1 // 1-based line numbers

		if strings.HasPrefix(line, "<<<<<<<") {
			if inConflict {
				return nil, fmt.Errorf("nested conflict markers found, which is not supported")
			}
			inConflict = true
			currentChunk = ConflictChunk{
				ID:        currentID,
				StartLine: lineNum,
			}
			continue
		}

		if inConflict && strings.HasPrefix(line, "=======") {
			currentChunk.BaseCode = strings.Join(baseLines, "\n")
			baseLines = nil
			continue
		}

		if inConflict && strings.HasPrefix(line, ">>>>>>>") {
			inConflict = false
			currentChunk.IncomingCode = strings.Join(incomingLines, "\n")
			currentChunk.EndLine = lineNum
			chunks = append(chunks, currentChunk)
			incomingLines = nil
			currentID++
			continue
		}

		if inConflict {
			if len(baseLines) == 0 && currentChunk.BaseCode == "" {
				baseLines = append(baseLines, line)
			} else if currentChunk.BaseCode != "" {
				incomingLines = append(incomingLines, line)
			} else {
				baseLines = append(baseLines, line)
			}
		}
	}

	if inConflict {
		return nil, fmt.Errorf("unclosed conflict marker found")
	}

	return chunks, nil
}

// HasMergeConflicts checks if a file has merge conflicts
func HasMergeConflicts(path string) (bool, error) {
	if err := ValidateFileExists(path); err != nil {
		return false, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("failed to read file: %w", err)
	}

	return strings.Contains(string(content), "<<<<<<<"), nil
}

// ReplaceConflictChunk replaces a specific conflict chunk in a file with new content
func ReplaceConflictChunk(path string, chunkID int, newContent string) error {
	if err := ValidateFileExists(path); err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	chunks, err := FindConflictChunks(string(content))
	if err != nil {
		return err
	}

	if chunkID < 0 || chunkID >= len(chunks) {
		return fmt.Errorf("chunk ID %d is out of range (found %d chunks)", chunkID, len(chunks))
	}

	targetChunk := chunks[chunkID]
	lines := strings.Split(string(content), "\n")

	// Find the start and end of the chunk in the file
	startLine := targetChunk.StartLine - 1 // Convert back to 0-based index
	endLine := targetChunk.EndLine - 1     // Convert back to 0-based index

	// Replace the chunk with the new content
	newLines := []string{}
	newLines = append(newLines, lines[:startLine]...)
	newLines = append(newLines, strings.Split(newContent, "\n")...)
	newLines = append(newLines, lines[endLine+1:]...)

	// Write the new content back to the file
	finalContent := strings.Join(newLines, "\n")
	err = os.WriteFile(path, []byte(finalContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// GetFileBlame returns the git blame information for a file
func GetFileBlame(path string) (string, error) {
	if err := ValidateFileExists(path); err != nil {
		return "", err
	}

	return ExecuteGitCommand("blame", "-s", path)
}

// GetCommitHistory returns the commit history for the repository or a specific file
// limit: maximum number of commits to return (defaults to 15 if <= 0)
// path: optional file path to filter commits (if empty, shows commits for entire repo)
func GetCommitHistory(path string, limit int) (string, error) {
	// Handle default case consistently
	if limit <= 0 {
		limit = 15 // Default to 15 commits if not specified or invalid
	}

	// Build git command with limit
	args := []string{"log", fmt.Sprintf("--max-count=%d", limit), "--pretty=format:%h|%an|%s", "--name-only"}

	// Add file path filter if provided
	if path != "" {
		if err := ValidateFileExists(path); err != nil {
			return "", err
		}
		args = append(args, path)
	}

	return ExecuteGitCommand(args...)
}

// GetFileVersionAtCommit returns the content of a file at a specific commit
func GetFileVersionAtCommit(path string, commitID string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	if commitID == "" {
		return "", fmt.Errorf("commit ID cannot be empty")
	}

	return ExecuteGitCommand("show", fmt.Sprintf("%s:%s", commitID, path))
}

// SaveChanges adds and commits all changes
func SaveChanges(message string) (string, error) {
	if message == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	// Add all changes
	_, err := ExecuteGitCommand("add", ".")
	if err != nil {
		return "", err
	}

	// Commit with the provided message
	commitMessage := fmt.Sprintf("[GitSynth] %s", message)
	return ExecuteGitCommand("commit", "-m", commitMessage)
}

// FormatCommitHistory formats the raw git log output into a structured format
// rawHistory: the raw output from git log command
// showFiles: whether to include the list of files in each commit (if false, file lists are omitted)
func FormatCommitHistory(rawHistory string, showFiles bool) (string, error) {
	lines := strings.Split(rawHistory, "\n")
	var result []string
	var currentCommit []string
	var isFileSection bool

	for _, line := range lines {
		if strings.Contains(line, "|") {
			// This is a commit header line
			if len(currentCommit) > 0 {
				result = append(result, strings.Join(currentCommit, "\n"))
				currentCommit = []string{}
			}

			parts := strings.SplitN(line, "|", 3)
			if len(parts) == 3 {
				hash := parts[0]
				author := parts[1]
				message := parts[2]

				currentCommit = append(currentCommit, fmt.Sprintf("Commit: %s", hash))
				currentCommit = append(currentCommit, fmt.Sprintf("Author: %s", author))
				currentCommit = append(currentCommit, fmt.Sprintf("Message: %s", message))

				// Only add the "Files:" header if we're showing files
				if showFiles {
					currentCommit = append(currentCommit, "Files:")
				}

				isFileSection = true
			}
		} else if line != "" && isFileSection && showFiles {
			// This is a file name, only add if showFiles is true
			currentCommit = append(currentCommit, fmt.Sprintf("  %s", line))
		}
	}

	if len(currentCommit) > 0 {
		result = append(result, strings.Join(currentCommit, "\n"))
	}

	return strings.Join(result, "\n\n"), nil
}
