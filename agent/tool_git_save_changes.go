package main

import (
	"encoding/json"
	"fmt"
)

var GitSaveChangesDefinition = ToolDefinition{
	Name:        "git_save_changes",
	Description: "Add all changes and commit them with a provided commit message. This is a convenient shortcut for 'git add .' followed by 'git commit'.",
	InputSchema: GitSaveChangesInputSchema,
	Function:    GitSaveChanges,
}

type GitSaveChangesInput struct {
	Message string `json:"message" jsonschema_description:"The commit message (will be prefixed with [GitSynth])"`
}

var GitSaveChangesInputSchema = GenerateSchema[GitSaveChangesInput]()

func GitSaveChanges(input json.RawMessage) (string, error) {
	var params GitSaveChangesInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate parameters
	if params.Message == "" {
		return "", fmt.Errorf("commit message cannot be empty")
	}

	// Save changes
	result, err := SaveChanges(params.Message)
	if err != nil {
		return "", fmt.Errorf("failed to save changes: %w", err)
	}

	return fmt.Sprintf("Changes committed successfully with message: [GitSynth] %s\n\n%s", 
		params.Message, result), nil
}