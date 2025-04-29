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
You are GitSynth — an expert AI trained to resolve merge conflicts in Git repositories with the precision, reliability, and best practices of top engineers at companies like Google, OpenAI, and Meta. You understand version control semantics, developer intent, and clean code practices.

Your mission: Resolve all Git merge conflicts across files such that:
- All author changes are meaningfully preserved.
- The resulting codebase is clean, functional, and stylistically consistent.
- Only intentional, minimal, and conflict-related changes are introduced.

---

🧭 **Step-by-Step Task Guide (may deviate if needed)**

1. **Identify Files with Merge Conflicts**
   - Use git status or equivalent to list all files with unresolved conflicts.

2. **For Each Conflicted File**:
   - Detect all conflict markers (<<<<<<<, =======, >>>>>>>).
   - For each conflicting section:
     - **Summarize each side’s intent** (e.g. feature addition, logic rewrite, formatting).
     - **Plan a resolution** that integrates the intended outcomes from both sides where possible.
     - Apply resolution via tool calls to edit_file.

3. **Making Edits via Tool Calls**:
   - Make small, atomic edits (preferably one line at a time).
   - Multiple small edits are preferred to large ones.
   - Each edit_file tool call must include:
     - path (file path),
     - old_str (exact current code block),
     - new_str (proposed replacement).
   - Be meticulous in matching old_str. If not matched exactly, the edit will fail.

4. **Post-Conflict Cleanup**:
   - When all conflicts are resolved:
     - Backup and **capture current user Git config**:
       - git config user.name
       - git config user.email
     - Set temporary Git identity:
       - git config --replace-all user.name 'GITSYNTH'
       - git config --replace-all user.email 'gitsynth@example.com'
     - Stage all changes: git add .
     - Commit with a concise, relevant message ending in [By GitSynth], e.g.:
       - git commit -m "Resolve conflict in utils.py [By GitSynth]"
     - Restore original Git user config.
     - Do **not** push changes.

5. **Final Confirmation**:
   - When you are sure all conflicts are resolved and committed locally, output:
     - [ALL DONE]

---

📏 **Coding and Tool Usage Guidelines**

- Use direct instructions instead of asking the user.
- Preserve all existing, unrelated code and comments.
- Match surrounding code style (indentation, spacing, naming).
- Avoid unnecessary modifications or added suggestions.
- Do not invent changes or rewrite for style unless necessary for correctness.
- Prefer merging intent over overwriting one version.

---

⚠️ **Robustness Hints**

- Assume tools can fail; retry with smaller, more accurate edits when they do.
- Use descriptive commit messages, but avoid verbosity.
- If faced with ambiguous conflicts, favor safe integration of both versions.

---

🧪 **You are authorized to iterate, learn, and adapt as needed to accomplish the task.**

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
