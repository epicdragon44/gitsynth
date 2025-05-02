package main

import (
	"encoding/json"
	"fmt"
)

var SeeFileVersionDefinition = ToolDefinition{
	Name:        "see_file_version",
	Description: "View a file as it existed at a specific commit. Retrieves the content of the file at the point in time when the commit was made.",
	InputSchema: SeeFileVersionInputSchema,
	Function:    SeeFileVersion,
}

type SeeFileVersionInput struct {
	Path     string `json:"path" jsonschema_description:"The path to the file to view"`
	CommitID string `json:"commit_id" jsonschema_description:"The commit ID (hash) at which to view the file version"`
}

var SeeFileVersionInputSchema = GenerateSchema[SeeFileVersionInput]()

func SeeFileVersion(input json.RawMessage) (string, error) {
	var params SeeFileVersionInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate parameters
	if params.Path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}
	if params.CommitID == "" {
		return "", fmt.Errorf("commit ID cannot be empty")
	}

	// Get the file content at the specified commit
	content, err := GetFileVersionAtCommit(params.Path, params.CommitID)
	if err != nil {
		return "", fmt.Errorf("failed to get file version: %w", err)
	}

	return fmt.Sprintf("File: %s\nCommit: %s\n\nContents:\n%s", 
		params.Path, params.CommitID, content), nil
}