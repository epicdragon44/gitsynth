package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var EditFileLineDefinition = ToolDefinition{
	Name:        "edit_file_line",
	Description: "Edit a specific line or range of lines in a file. Replaces the content of the specified line(s) with new content. Line numbers are 1-indexed.",
	InputSchema: EditFileLineInputSchema,
	Function:    EditFileLine,
}

type EditFileLineInput struct {
	Path       string `json:"path" jsonschema_description:"The path to the file to edit"`
	StartLine  int    `json:"start_line" jsonschema_description:"The starting line number to replace (1-indexed)"`
	EndLine    int    `json:"end_line,omitempty" jsonschema_description:"Optional end line number for replacing a range (inclusive, 1-indexed). If omitted, only the start line is replaced."`
	NewContent string `json:"new_content" jsonschema_description:"The new content to replace the specified line(s) with. Can contain multiple lines (use \n for line breaks)."`
}

var EditFileLineInputSchema = GenerateSchema[EditFileLineInput]()

func EditFileLine(input json.RawMessage) (string, error) {
	var params EditFileLineInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate file exists
	if err := ValidateFileExists(params.Path); err != nil {
		return "", err
	}

	// Validate line numbers
	if params.StartLine < 1 {
		return "", fmt.Errorf("start_line must be at least 1")
	}

	// If EndLine is not specified or is 0, set it to StartLine (edit only one line)
	if params.EndLine == 0 {
		params.EndLine = params.StartLine
	}

	// Ensure EndLine is not less than StartLine
	if params.EndLine < params.StartLine {
		return "", fmt.Errorf("end_line cannot be less than start_line")
	}

	// Read file content
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	// Check if startLine is out of range
	if params.StartLine > len(lines) {
		return "", fmt.Errorf("start_line %d is beyond the file length of %d lines", 
			params.StartLine, len(lines))
	}

	// Check if endLine is out of range
	if params.EndLine > len(lines) {
		return "", fmt.Errorf("end_line %d is beyond the file length of %d lines", 
			params.EndLine, len(lines))
	}

	// Convert to 0-based indexing for array access
	startIndex := params.StartLine - 1
	endIndex := params.EndLine - 1

	// The lines to replace
	newLines := strings.Split(params.NewContent, "\n")
	
	// Construct the new content
	result := append(append([]string{}, lines[:startIndex]...), newLines...)
	if endIndex < len(lines)-1 {
		result = append(result, lines[endIndex+1:]...)
	}

	// Write the updated content back to the file
	err = os.WriteFile(params.Path, []byte(strings.Join(result, "\n")), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write updated content to file: %w", err)
	}

	// Build result message
	var actionMsg string
	if params.StartLine == params.EndLine {
		actionMsg = fmt.Sprintf("line %d", params.StartLine)
	} else {
		actionMsg = fmt.Sprintf("lines %d-%d", params.StartLine, params.EndLine)
	}

	return fmt.Sprintf("Successfully edited %s in file %s", 
		actionMsg, params.Path), nil
}