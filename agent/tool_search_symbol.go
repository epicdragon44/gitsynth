package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type SearchSymbolParams struct {
	// The symbol to search for. Can be a literal string or a regex pattern
	Symbol string `json:"symbol" jsonschema:"description=The symbol to search for (e.g. function name, class name, variable). Can be a regular expression."`
	
	// Whether the symbol should be treated as a regex pattern
	IsRegex bool `json:"is_regex,omitempty" jsonschema:"description=If true, the symbol will be treated as a regular expression pattern."`
	
	// Optional glob pattern to filter which files to search in (e.g. "*.go", "src/**/*.ts")
	FilePattern string `json:"file_pattern,omitempty" jsonschema:"description=Optional glob pattern to filter which files to search in (e.g. '*.go', 'src/**/*.ts')."`
	
	// Whether the search should be case-sensitive
	CaseSensitive bool `json:"case_sensitive,omitempty" jsonschema:"description=Whether the search should be case-sensitive. Defaults to false."`
}

var SearchSymbolDefinition = ToolDefinition{
	Name: "search_symbol",
	Description: `Search for a symbol (function name, class name, variable, etc.) across the project.
- Can search using literal strings or regular expressions
- Optionally filter files by glob pattern
- Returns matching lines with file paths and line numbers
- Useful for finding declarations and usages of symbols`,
	InputSchema: GenerateSchema[SearchSymbolParams](),
	Function: func(input json.RawMessage) (string, error) {
		var params SearchSymbolParams
		if err := json.Unmarshal(input, &params); err != nil {
			return "", fmt.Errorf("failed to parse search symbol parameters: %w", err)
		}

		if params.Symbol == "" {
			return "", fmt.Errorf("symbol parameter cannot be empty")
		}

		// Prepare search pattern
		searchPattern := params.Symbol
		if !params.IsRegex {
			// Escape special regex characters if it's a literal search
			searchPattern = regexp.QuoteMeta(searchPattern)
		}

		// Match whole words by default if it's not a regex search
		if !params.IsRegex {
			searchPattern = fmt.Sprintf("\\b%s\\b", searchPattern)
		}

		// Use grep to perform the search
		includePattern := params.FilePattern
		if includePattern == "" {
			includePattern = "*" // Default to all files in current directory
		}

		res, err := grep(searchPattern, includePattern, params.CaseSensitive)
		if err != nil {
			return "", fmt.Errorf("search failed: %w", err)
		}

		// If no results found
		if len(res) == 0 {
			var details strings.Builder
			details.WriteString(fmt.Sprintf("No matches found for symbol '%s'\n", params.Symbol))
			details.WriteString(fmt.Sprintf("Search parameters:\n"))
			details.WriteString(fmt.Sprintf("- Pattern: %s\n", searchPattern))
			details.WriteString(fmt.Sprintf("- File filter: %s\n", includePattern))
			details.WriteString(fmt.Sprintf("- Case sensitive: %v\n", params.CaseSensitive))
			details.WriteString(fmt.Sprintf("- Regex mode: %v\n", params.IsRegex))
			return details.String(), nil
		}

		// Format results
		var output strings.Builder
		output.WriteString(fmt.Sprintf("Found %d matches for symbol '%s':\n\n", len(res), params.Symbol))

		for _, match := range res {
			// Clean up the path for display
			relPath := match.Path
			if abs, err := filepath.Abs(relPath); err == nil {
				if rel, err := filepath.Rel(".", abs); err == nil {
					relPath = rel
				}
			}

			// Format the line with some context
			content := strings.TrimSpace(match.Content)
			if len(content) > 120 { // Truncate very long lines
				content = content[:117] + "..."
			}

			output.WriteString(fmt.Sprintf("%s:%d: %s\n", relPath, match.Line, content))
		}

		return output.String(), nil
	},
}