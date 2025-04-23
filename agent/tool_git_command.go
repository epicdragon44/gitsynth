package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

var GitCommandDefinition = ToolDefinition{
	Name:        "git_command",
	Description: "Execute a git command in the shell. Use this to run git operations like init, add, commit, merge, branch, etc. The command should start with 'git'.",
	InputSchema: GitCommandInputSchema,
	Function:    GitCommand,
}

type GitCommandInput struct {
	Command string `json:"command" jsonschema_description:"The git command to execute. Must start with 'git'."` 
}

var GitCommandInputSchema = GenerateSchema[GitCommandInput]()

func GitCommand(input json.RawMessage) (string, error) {
	gitCommandInput := GitCommandInput{}
	err := json.Unmarshal(input, &gitCommandInput)
	if err != nil {
		return "", err
	}

	// Validate that the command starts with git
	if !strings.HasPrefix(gitCommandInput.Command, "git ") {
		return "", fmt.Errorf("command must start with 'git'")
	}

	// Split the command into parts for exec.Command
	cmdParts := strings.Fields(gitCommandInput.Command)
	if len(cmdParts) < 1 {
		return "", fmt.Errorf("invalid command format")
	}

	// Create the command
	cmd := exec.Command(cmdParts[0], cmdParts[1:]...)

	// Get the combined output (stdout and stderr)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %s\nOutput: %s", err.Error(), string(output)), err
	}

	return string(output), nil
}