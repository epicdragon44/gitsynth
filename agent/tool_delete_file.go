package main

import (
	"encoding/json"
	"fmt"
	"os"
)

var DeleteFileDefinition = ToolDefinition{
	Name: "delete_file",
	Description: `Delete a file at the given path.

Removes the specified file from the filesystem. Returns an error if the file does not exist.
`,
	InputSchema: DeleteFileInputSchema,
	Function:    DeleteFile,
}

type DeleteFileInput struct {
	Path string `json:"path" jsonschema_description:"The path to the file to delete"`
}

var DeleteFileInputSchema = GenerateSchema[DeleteFileInput]()

func DeleteFile(input json.RawMessage) (string, error) {
	deleteFileInput := DeleteFileInput{}
	err := json.Unmarshal(input, &deleteFileInput)
	if err != nil {
		return "", err
	}

	if deleteFileInput.Path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Check if file exists before attempting to delete
	_, err = os.Stat(deleteFileInput.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", deleteFileInput.Path)
		}
		return "", fmt.Errorf("failed to access file: %w", err)
	}

	// Delete the file
	err = os.Remove(deleteFileInput.Path)
	if err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	return fmt.Sprintf("Successfully deleted file %s", deleteFileInput.Path), nil
}