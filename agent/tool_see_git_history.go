package main

import (
	"encoding/json"
	"fmt"
)

var SeeGitHistoryDefinition = ToolDefinition{
	Name:        "see_git_history",
	Description: "View the commit history of the repository, with details like commit hash, author, message, and modified files. Optionally filter to show only commits affecting a specific file, limit the number of commits shown (defaults to 15), and choose whether to display files touched by each commit (defaults to false).",
	InputSchema: SeeGitHistoryInputSchema,
	Function:    SeeGitHistory,
}

type SeeGitHistoryInput struct {
	Path      string `json:"path,omitempty" jsonschema_description:"Optional path to a file to filter commit history for only changes to that file"`
	Limit     int    `json:"limit,omitempty" jsonschema_description:"Optional limit on the number of commits to show, defaults to 15 if not specified"`
	ShowFiles bool   `json:"show_files,omitempty" jsonschema_description:"Optional flag to show the files touched in each commit, defaults to false"`
}

var SeeGitHistoryInputSchema = GenerateSchema[SeeGitHistoryInput]()

func SeeGitHistory(input json.RawMessage) (string, error) {
	var params SeeGitHistoryInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Check file existence if a path is provided
	if params.Path != "" {
		if err := ValidateFileExists(params.Path); err != nil {
			return "", err
		}
	}

	// Get commit history
	rawHistory, err := GetCommitHistory(params.Path, params.Limit)
	if err != nil {
		return "", fmt.Errorf("failed to get commit history: %w", err)
	}

	// Format the commit history
	formattedHistory, err := FormatCommitHistory(rawHistory, params.ShowFiles)
	if err != nil {
		return "", fmt.Errorf("failed to format commit history: %w", err)
	}

	if params.Path != "" {
		return fmt.Sprintf("Git history for file: %s\n\n%s", params.Path, formattedHistory), nil
	}

	return fmt.Sprintf("Git repository history:\n\n%s", formattedHistory), nil
}