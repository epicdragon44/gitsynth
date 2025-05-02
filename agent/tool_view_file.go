package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var ViewFileDefinition = ToolDefinition{
	Name:        "view_file",
	Description: "View the contents of a file with line numbers (shown by default). Optionally includes git blame information to see who edited each line. You can disable line numbers by setting with_line_numbers to false.",
	InputSchema: ViewFileInputSchema,
	Function:    ViewFile,
}

type ViewFileInput struct {
	Path            string `json:"path" jsonschema_description:"The path to the file to view"`
	WithBlame       bool   `json:"with_blame,omitempty" jsonschema_description:"Whether to include git blame information (who edited each line)"`
	WithLineNumbers *bool  `json:"with_line_numbers,omitempty" jsonschema_description:"Whether to display line numbers at the beginning of each line (defaults to true unless explicitly set to false)"`
}

var ViewFileInputSchema = GenerateSchema[ViewFileInput]()

func ViewFile(input json.RawMessage) (string, error) {
	var params ViewFileInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate file exists
	if err := ValidateFileExists(params.Path); err != nil {
		return "", err
	}

	// Read file contents
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Process content to add line numbers unless explicitly disabled
	fileContent := string(content)
	
	// Only skip line numbers if WithLineNumbers is explicitly set to false
	shouldShowLineNumbers := true
	if params.WithLineNumbers != nil && *params.WithLineNumbers == false {
		shouldShowLineNumbers = false
	}
	
	if shouldShowLineNumbers {
		fileContent = addLineNumbers(fileContent)
	}

	// If blame is requested, get git blame and return it along with the content
	if params.WithBlame {
		blame, err := GetFileBlame(params.Path)
		if err != nil {
			return "", fmt.Errorf("failed to get git blame: %w", err)
		}
		return fmt.Sprintf("File: %s\n\nContents:\n%s\n\nBlame:\n%s", 
			params.Path, fileContent, blame), nil
	}

	return fmt.Sprintf("File: %s\n\nContents:\n%s", params.Path, fileContent), nil
}

// addLineNumbers adds line numbers at the beginning of each line
func addLineNumbers(content string) string {
	lines := strings.Split(content, "\n")
	formattedLines := make([]string, len(lines))
	
	// Determine width for line number formatting (based on total number of lines)
	width := len(fmt.Sprintf("%d", len(lines)))
	
	// Format each line with its line number
	for i, line := range lines {
		lineNum := i + 1 // 1-indexed line numbers
		formattedLines[i] = fmt.Sprintf("%*d | %s", width, lineNum, line)
	}
	
	return strings.Join(formattedLines, "\n")
}