package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/joho/godotenv"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []ToolDefinition
}

type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

func main() {

	// --- Load environment variables from .env file or system ---

	envPath := ".env"
	envLoadErr := godotenv.Load(envPath)

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: ANTHROPIC_API_KEY not found in your environment.")
		if envLoadErr != nil {
			// Try to check if the file exists to provide a more specific error
			if _, statErr := os.Stat(envPath); os.IsNotExist(statErr) {
				absPath, _ := filepath.Abs(envPath)
				fmt.Printf("No .env file found in current directory (%s)\n. You may create one and add ANTHROPIC_API_KEY=your-api-key to it.\n", absPath)
			} else {
				fmt.Printf(".env file found, but failed to load: %v\n", envLoadErr)
			}
		} else {
			fmt.Println(".env loaded successfully, but no ANTHROPIC_API_KEY found. Either set it in your environment by hand, or set it in your .env file.")
		}
		os.Exit(1)
	}

	// --- Initialize the agent and run it ---

	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}
	tools := []ToolDefinition{ReadFileDefinition, ListFilesDefinition, EditFileDefinition, DeleteFileDefinition, GitCommandDefinition}
	agent := NewAgent(&client, getUserMessage, tools)
	runErr := agent.Run(context.TODO())
	if runErr != nil {
		fmt.Printf("Error: %s\n", runErr.Error())
	}
}

func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool), tools []ToolDefinition) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	fmt.Println("WELCOME TO GITSYNTH [LOCAL AGENT]. Use 'ctrl-c' to quit at any time.")
	fmt.Println("GitSynth will now begin resolving your merge conflicts.")

	// Add default first message to start the conversation
	defaultPrompt := `
		Hi. Your name is GitSynth. You are a powerful AI assistant that specializes in resolving merge conflicts in Git repositories, trained on years of best practices at companies of all kinds, from FAANG companies like Google to high-growth start-ups like OpenAI.
		The task at hand today: Identify all files with merge conflicts, and resolve them such that it captures and preserves the spirit and intent of all changes, while leaving the code just as if not more functional and clean than before.

		Here's a suggestion for how to approach this task:
		1. Identify all files with merge conflicts.
			a. You may do this with "git status" or a similar, equivalent command of your choice.
		2. For each file,
			a. Identify conflicting sections, and summarize the changes and intentions that each party had in mind.
			b. Propose a resolution strategy that captures the spirit and intent of all changes, and leaves the file in a functional and clean state.
			c. Implement the proposed resolution strategy by making the necessary changes to the file.
				i. Each tool call to edit_file should be made with a small edit (one line of code at a time).
				ii. You may then call this tool as many times as you'd like to build up to larger changes.
				iii. Sending edits that are too large will cause the tool to fail.
				iv. IMPORTANTLY, don't forget to include old_str and new_str, along with path, in the parameters of your tool call.
		3. Repeat steps 1-3 until all merge conflicts are resolved.
		4. Once all merge conflicts are resolved, save the changes:
			a. Identify the current user config, such as the user.name and user.email making commits, using git config user.name and git config user.email.
			b. Now, set the git config to declare that you are making the edits. Your name should be 'GITSYNTH' and your email should be 'gitsynth@example.com'. The commands, thusly, should be:
				git config user.name 'GITSYNTH' --replace-all
				git config user.email 'gitsynth@example.com' --replace-all
			c. Stage the changes with "git add ."
			d. Then, commit changes with a descriptive, concise message that ends with "[By GitSynth]". For instance, git commit -a -m "Resolved merge conflict in file.go [By GitSynth]"
			d. Reset the git config (name and email) to the original config! THIS IS IMPORTANT TO PRESERVE THE USER'S CONFIGURATION.
				git config --replace-all user.name ORIGINAL_NAME
				git config --replace-all user.email ORIGINAL_EMAIL
			e. Do NOT push the changes to the remote repository.
		5. Once you're sure you're all done, output "[ALL DONE]".

		Note that the above is a suggestion only. Feel free to deviate from it as you see fit. Unexpected errors may occur, as is natural -- be resilient and smart, and figure out how to accomplish your goals regardless.

		Some basic code guidelines:
		- Don't invent changes other than what's explicitly requested.
		- Avoid giving feedback about understanding in comments or documentation.
		- Don't remove unrelated code or functionalities. Pay attention to preserving existing structures.
		- Don't ask the user to approve anything. Just continue until task complete.
		- Don't suggest updates or changes to files when there are no actual modifications needed.
		- Match the code style of the existing codebase.
		- Please do not unnecessarily remove any comments or code.

		You may begin.
	`
	userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(defaultPrompt))
	conversation = append(conversation, userMessage)

	for {
		finalMessage := &anthropic.Message{}
		finalErr := error(nil)
		maxRetries := 5
		for retries := 0; retries < maxRetries; retries++ {
			message, err := a.runInference(ctx, conversation)

			if err != nil {
				finalErr = err

				var apiErr *anthropic.Error
				if errors.As(err, &apiErr) {
					// Exponentially retry non-fatal API errors and continue
					backoffSeconds := (retries * retries) * (rand.Intn(3) + 2)
					fmt.Printf("API error occurred, retrying in %d seconds (attempt %d/%d): %v\n",
						backoffSeconds, retries+1, maxRetries, err)
					time.Sleep(time.Duration(backoffSeconds) * time.Second)
					continue
				} else { // Non-API errors are not retried
					fmt.Printf("Non-retryable error: %v\n", err)
					break
				}
			} else {
				finalMessage = message
				break
			}
		}
		if finalErr != nil {
			fmt.Printf("\u001b[91mError\u001b[0m: %s\n", finalErr.Error())
			return finalErr
		}
		conversation = append(conversation, finalMessage.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, content := range finalMessage.Content {
			switch content.Type {
			case "text":
				fmt.Printf("\u001b[93mGitSynth\u001b[0m: %s\n", content.Text)
			case "tool_use":
				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)
			}
		}
		if len(toolResults) == 0 {
			// Done!
			break
		}
		conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
	}

	return nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var toolDef ToolDefinition
	var found bool
	for _, tool := range a.tools {
		if tool.Name == name {
			toolDef = tool
			found = true
			break
		}
	}
	if !found {
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	fmt.Printf("\u001b[92mTool\u001b[0m: %s(%s)\n", name, input)
	response, err := toolDef.Function(input)
	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	return anthropic.NewToolResultBlock(id, response, false)
}

func (a *Agent) runInference(ctx context.Context, conversation []anthropic.MessageParam) (*anthropic.Message, error) {
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range a.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5SonnetLatest,
		MaxTokens: int64(1024),
		Messages:  conversation,
		Tools:     anthropicTools,
	})
	return message, err
}
