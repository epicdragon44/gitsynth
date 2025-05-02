package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var SeeFileChunksDefinition = ToolDefinition{
	Name:        "see_file_chunks",
	Description: "View and analyze the conflict chunks in a file. Shows each chunk with its ID, base code, and incoming code.",
	InputSchema: SeeFileChunksInputSchema,
	Function:    SeeFileChunks,
}

type SeeFileChunksInput struct {
	Path string `json:"path" jsonschema_description:"The path to the file with conflict chunks to analyze"`
}

var SeeFileChunksInputSchema = GenerateSchema[SeeFileChunksInput]()

func SeeFileChunks(input json.RawMessage) (string, error) {
	var params SeeFileChunksInput
	if err := json.Unmarshal(input, &params); err != nil {
		return "", fmt.Errorf("failed to parse parameters: %w", err)
	}

	// Validate file exists
	if err := ValidateFileExists(params.Path); err != nil {
		return "", err
	}

	// Check if the file has merge conflicts
	hasConflicts, err := HasMergeConflicts(params.Path)
	if err != nil {
		return "", err
	}
	if !hasConflicts {
		return fmt.Sprintf("No merge conflicts found in file: %s", params.Path), nil
	}

	// Read file contents
	content, err := os.ReadFile(params.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Find and parse conflict chunks
	chunks, err := FindConflictChunks(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse conflict chunks: %w", err)
	}

	// Format the output
	var result strings.Builder
	result.WriteString(fmt.Sprintf("File: %s\n\n", params.Path))
	result.WriteString(fmt.Sprintf("Found %d conflict chunks:\n\n", len(chunks)))

	for _, chunk := range chunks {
		result.WriteString(fmt.Sprintf("Chunk ID: %d (lines %d-%d)\n", 
			chunk.ID, chunk.StartLine, chunk.EndLine))
		result.WriteString("Base Code:\n")
		result.WriteString(fmt.Sprintf("```\n%s\n```\n\n", chunk.BaseCode))
		result.WriteString("Incoming Code:\n")
		result.WriteString(fmt.Sprintf("```\n%s\n```\n\n", chunk.IncomingCode))
		result.WriteString("---\n\n")
	}

	return result.String(), nil
}