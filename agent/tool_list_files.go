package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

var ListFilesInputSchema = GenerateSchema[ListFilesInput]()

func ListFiles(input json.RawMessage) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		panic(err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	// Load .gitignore patterns if available
	ignorePatterns := loadGitignorePatterns()

	// Read directory entries (non-recursively)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var files []string
	for _, entry := range entries {
		name := entry.Name()
		path := filepath.Join(dir, name)
		relPath, _ := filepath.Rel(dir, path)

		// Skip if matches gitignore patterns
		if shouldIgnore(relPath, entry.IsDir(), ignorePatterns) {
			continue
		}

		if entry.IsDir() {
			files = append(files, relPath+"/")
		} else {
			files = append(files, relPath)
		}
	}

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// loadGitignorePatterns loads patterns from the .gitignore file if it exists
func loadGitignorePatterns() []string {
	var patterns []string
	gitignorePath := ".gitignore"

	// Check if .gitignore exists
	file, err := os.Open(gitignorePath)
	if err != nil {
		// .gitignore doesn't exist or can't be opened, return empty patterns
		return patterns
	}
	defer file.Close()

	// Read patterns line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns
}

// shouldIgnore checks if a file or directory should be ignored based on gitignore patterns
func shouldIgnore(path string, isDir bool, patterns []string) bool {
	// Convert Windows path separators to Unix style for matching
	path = filepath.ToSlash(path)
	
	// Always check the file/dir name itself
	name := filepath.Base(path)

	for _, pattern := range patterns {
		// Handle negation patterns (those starting with !)
		if strings.HasPrefix(pattern, "!") {
			// Negation patterns negate previous matches
			continue
		}

		// Handle directory-specific patterns (ending with /)
		if strings.HasSuffix(pattern, "/") {
			if !isDir {
				continue // Pattern only applies to directories
			}
			pattern = strings.TrimSuffix(pattern, "/")
		}

		// Handle simple glob patterns
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}

		// Handle extension ignores (like *.go)
		if strings.HasPrefix(pattern, "*.") {
			ext := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(name, ext) {
				return true
			}
		}

		// Handle direct path matches
		if pattern == path {
			return true
		}
	}

	return false
}
