package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

var FindMergeConflictsDefinition = ToolDefinition{
	Name:        "find_merge_conflicts",
	Description: "Lists all files that contain Git merge conflict markers (<<<<<<< HEAD, =======, or >>>>>>>). Use this to identify files with unresolved merge conflicts.",
	InputSchema: FindMergeConflictsInputSchema,
	Function:    FindMergeConflicts,
}

type FindMergeConflictsInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to search for merge conflicts. Defaults to current directory if not provided."`
}

var FindMergeConflictsInputSchema = GenerateSchema[FindMergeConflictsInput]()

func FindMergeConflicts(input json.RawMessage) (string, error) {
	findMergeConflictsInput := FindMergeConflictsInput{}
	err := json.Unmarshal(input, &findMergeConflictsInput)
	if err != nil {
		return "", err
	}

	dir := "."
	if findMergeConflictsInput.Path != "" {
		dir = findMergeConflictsInput.Path
	}

	// Regular expression to match Git merge conflict markers
	mergeConflictPattern := regexp.MustCompile(`<<<<<<< HEAD|=======|>>>>>>> `)

	filesWithConflicts := []string{}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		// Check if the file contains merge conflict markers
		if mergeConflictPattern.Match(content) {
			// Get the relative path
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return err
			}
			filesWithConflicts = append(filesWithConflicts, relPath)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	result, err := json.Marshal(filesWithConflicts)
	if err != nil {
		return "", err
	}

	return string(result), nil
}