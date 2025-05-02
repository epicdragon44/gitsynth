package main

import (
	"encoding/json"
	"fmt"
)

var EditFileChunkDefinition = ToolDefinition{
	Name:        "edit_file_chunk",
	Description: "Resolve a specific conflict chunk in a file by replacing it with new content. Identifies the chunk by its ID number (starting from 0 for the first chunk at the top of the file).",
	InputSchema: EditFileChunkInputSchema,
	Function:    EditFileChunk,
}

type EditFileChunkInput struct {
	Path       string `json:"path" jsonschema_description:"The path to the file containing the conflict chunk"`
	ChunkID    int    `json:"chunk_id" jsonschema_description:"The ID of the conflict chunk to edit (zero-indexed, with chunk 0 being the first chunk from the top of the file)"`
	NewContent string `json:"new_content" jsonschema_description:"The content to replace the entire conflict chunk with"`
}

var EditFileChunkInputSchema = GenerateSchema[EditFileChunkInput]()

func EditFileChunk(input json.RawMessage) (string, error) {
	var params EditFileChunkInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate file exists
	if err := ValidateFileExists(params.Path); err != nil {
		return "", err
	}

	// Validate that file has conflict markers
	hasConflicts, err := HasMergeConflicts(params.Path)
	if err != nil {
		return "", err
	}
	if !hasConflicts {
		return "", fmt.Errorf("no merge conflicts found in file: %s", params.Path)
	}

	// Replace the conflict chunk
	if err := ReplaceConflictChunk(params.Path, params.ChunkID, params.NewContent); err != nil {
		return "", fmt.Errorf("failed to replace conflict chunk: %w", err)
	}

	return fmt.Sprintf("Successfully replaced conflict chunk %d in file %s", 
		params.ChunkID, params.Path), nil
}