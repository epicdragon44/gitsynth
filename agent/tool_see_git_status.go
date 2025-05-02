package main

import (
	"encoding/json"
	"fmt"
)

var SeeGitStatusDefinition = ToolDefinition{
	Name:        "see_git_status",
	Description: "Runs Git Status, which lists all files that have unresolved git merge conflicts, the current branch, and other information.",
	InputSchema: SeeGitStatusInputSchema,
	Function:    SeeGitStatus,
}

type SeeGitStatusInput struct {
	// No parameters needed for this tool
}

var SeeGitStatusInputSchema = GenerateSchema[SeeGitStatusInput]()

func SeeGitStatus(input json.RawMessage) (string, error) {
	// Simply run git status and return the output
	output, err := ExecuteGitCommand("status")
	if err != nil {
		return "", fmt.Errorf("failed to run git status: %w", err)
	}

	return output, nil
}
