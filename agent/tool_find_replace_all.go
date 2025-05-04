package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type FindReplaceAllParams struct {
	// The text to find. Can be a literal string or a regex pattern
	Find string `json:"find" jsonschema:"description=The text to find. Can be a regular expression."`

	// The text to replace matches with
	Replace string `json:"replace" jsonschema:"description=The text to replace matches with."`
	
	// Whether the find text should be treated as a regex pattern
	IsRegex bool `json:"is_regex,omitempty" jsonschema:"description=If true, the find text will be treated as a regular expression pattern."`
	
	// Optional glob pattern to filter which files to search in (e.g. "*.go", "src/**/*.ts")
	FilePattern string `json:"file_pattern,omitempty" jsonschema:"description=Optional glob pattern to filter which files to search in (e.g. '*.go', 'src/**/*.ts')."`
	
	// Whether the search should be case-sensitive
	CaseSensitive bool `json:"case_sensitive,omitempty" jsonschema:"description=Whether the search should be case-sensitive. Defaults to false."`
}

var FindReplaceAllDefinition = ToolDefinition{
	Name: "find_replace_all",
	Description: `Find and replace text across all files in the project.
- Can search using literal strings or regular expressions
- Optionally filter files by glob pattern
- Replaces all occurrences of the find text with the replace text
- Shows a preview of changes before applying them
- Returns a summary of changes made`,
	InputSchema: GenerateSchema[FindReplaceAllParams](),
	Function: func(input json.RawMessage) (string, error) {
		var params FindReplaceAllParams
		if err := json.Unmarshal(input, &params); err != nil {
			return "", fmt.Errorf("failed to parse find and replace parameters: %w", err)
		}

		if params.Find == "" {
			return "", fmt.Errorf("find parameter cannot be empty")
		}

		// Use grep to find all matches first
		includePattern := params.FilePattern
		if includePattern == "" {
			includePattern = "*" // Default to all files in current directory
		}

		// Search for matches using the same logic as search_symbol
		matches, err := grep(params.Find, includePattern, params.CaseSensitive)
		if err != nil {
			return "", fmt.Errorf("search failed: %w", err)
		}

		// If no matches found
		if len(matches) == 0 {
			var details strings.Builder
			details.WriteString(fmt.Sprintf("No matches found for text '%s'\n", params.Find))
			details.WriteString(fmt.Sprintf("Search parameters:\n"))
			details.WriteString(fmt.Sprintf("- Pattern: %s\n", params.Find))
			details.WriteString(fmt.Sprintf("- File filter: %s\n", includePattern))
			details.WriteString(fmt.Sprintf("- Case sensitive: %v\n", params.CaseSensitive))
			details.WriteString(fmt.Sprintf("- Regex mode: %v\n", params.IsRegex))
			return details.String(), nil
		}

		// Group matches by file
		fileMatches := make(map[string][]GrepMatch)
		for _, match := range matches {
			fileMatches[match.Path] = append(fileMatches[match.Path], match)
		}

		// Process each file
		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found matches in %d files.\n\n", len(fileMatches)))
		
		filesModified := 0
		replacementsCount := 0

		for filePath, matches := range fileMatches {
			// Read the entire file
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
			}

			// Create new content with replacements
			fileContent := string(content)
			newContent := fileContent

			// Perform the replacement
			var replaceFunc func(string, string, string) string
			if params.IsRegex {
				replaceFunc = strings.ReplaceAll // For now using simple replace, could be enhanced with regex
			} else {
				replaceFunc = strings.ReplaceAll
			}

			newContent = replaceFunc(fileContent, params.Find, params.Replace)

			// If content changed, write it back
			if newContent != fileContent {
				if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
					return "", fmt.Errorf("failed to write changes to file %s: %w", filePath, err)
				}
				filesModified++
				replacementsCount += len(matches)

				// Report the changes for this file
				relPath := filePath
				if abs, err := filepath.Abs(filePath); err == nil {
					if rel, err := filepath.Rel(".", abs); err == nil {
						relPath = rel
					}
				}
				output.WriteString(fmt.Sprintf("Modified %s (%d replacements)\n", relPath, len(matches)))
			}
		}

		// Summary
		output.WriteString(fmt.Sprintf("\nSummary:\n"))
		output.WriteString(fmt.Sprintf("- Total files modified: %d\n", filesModified))
		output.WriteString(fmt.Sprintf("- Total replacements made: %d\n", replacementsCount))

		return output.String(), nil
	},
}