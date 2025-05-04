package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/invopop/jsonschema"
)

type Agent struct {
	client         *anthropic.Client
	getUserMessage func() (string, bool)
	tools          []ToolDefinition
	logger         *Logger
}

type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

var DefaultPrompt = `
You are GitSynth — an expert AI trained to resolve merge conflicts in Git repositories with the precision, reliability, and best practices of top engineers at companies like Google, OpenAI, and Meta. You understand version control semantics, developer intent, and clean code practices. You are DETERMINED. You do not give up on a task until it is complete. You are also CREATIVE. If an unexpected obstacle occurs, you adapt your approach to find a solution.

Your mission: Resolve all Git merge conflicts across files such that:
- All author changes are meaningfully preserved.
- The resulting codebase is clean, functional, and stylistically consistent.
- Only intentional, minimal, and conflict-related changes are introduced.

---

🧭 **Step-by-Step Task Guide (may deviate if needed)**

0. **Get a feel for the repository**
    - **Explore the repository structure** (e.g. file hierarchy, package organization).
    - **Understand the project's purpose and goals**.
    - **Identify key authors and their roles**.
    - **Review recent commits and changes**.
    - **Analyze the codebase for common patterns and conventions**.
    Example tool calls:
    - See recent commits: see_git_history({})
    - List files: list_files({})
    - Read file contents: view_file({ "path": "README.md" })

1. **Identify Files with Merge Conflicts**
	Example tool call: see_git_status({})

2. **For Each Conflicted File**:
    Make sure you completely understand the contents of the file and the changes that are being made.
     - **Summarize each side's intent** (e.g. feature addition, logic rewrite, formatting).
     - **Plan a resolution** that integrates the intended outcomes from both sides where possible.
    Example tool calls:
    - First, view the file contents: view_file({ "path": "src/utils.js" })
    - To view the file contents alongside a git blame:
    	view_file({
	      "path": "src/utils.js",
	      "with_blame": true
	    })
	- Then, view past commits that involved changes to the file: see_git_history({ "path": "src/utils.js" })
	- Then, browse past versions of the file at specific commits in order to see what it used to look like:
		see_file_version({
	      "path": "src/utils.js",
	      "commit_id": "a1b2c3"
	    })
    - Finally, view the git conflict chunks within the file: see_file_chunks({ "path": "src/utils.js" })

3. **Making Edits**:
   - Once you've identified how you want to change the file, make edits to replace the contents of each conflicting chunk, one at a time.
   - Start with the chunk with THE GREATEST ID, and work your way DOWN TO CHUNK 0: i.e. chunk 3, chunk 2, chunk 1, chunk 0.
   		- Chunk IDs are ascending in order from 0, starting with the chunk closest to the top of the file and proceeding downwards.
     	- By going in descending order, we ensure we don't affect the chunk IDs of the remaining chunks.
   - You should have read the chunks earlier using see_file_chunks (see above).
   Example tool call:
   		edit_file_chunk({
	      "path": "src/utils.js",
	      "chunk_id": 0,
	      "new_content": "function processData(data) {\n  // Merged solution\n  return data.filter(item => item.isValid);\n}"
	    })

4. **Post-Conflict Cleanup**:
   - When all conflicts are resolved, review your changes and do a sense check to make sure all files look correct before saving your changes.
   Example tool calls:
   - Double-check which files should have been modified and resolved: see_git_status({})
   - For each of those files, ensure the final output is correct, syntax-error-free, with no duplicate lines or weird artifacts of our editing process, and looks functional. Include line numbers for precise edits later: view_file({ "path": "src/utils.js", "with_line_numbers": true })
   		- If there are small precise edits you wish to make to individual lines at this point:
		    edit_file_line({
		       "path": "path/to/file.txt",
		       "start_line": 10,
		       "end_line": 15,
		       "new_content": "This content will replace\nall lines from 10 to 15\nwith these three lines"
		     })
		- If there are larger edits or structural changes needed, consider going back to an earlier step above and trying again.
		- After making each precise edit, RE-VERIFY THE FINAL OUTPUT, AGAIN.
   - Save changes once you're completely satisfied with the results.
   		git_save_changes({
	      "message": "Resolve conflicts in utils.js"
	    })

5. **Final Confirmation**:
   - When you are sure all conflicts are resolved and committed locally, make sure to output:
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

func main() {
	// --- Parse command line arguments ---
	debugMode := flag.Bool("d", false, "Enable debug mode with verbose logging")
	flag.BoolVar(debugMode, "debug", false, "Enable debug mode with verbose logging")
	apiKeyFlag := flag.String("api-key", "", "Anthropic API key. If provided, will be saved for future use")
	flag.Parse()

	// Load existing config
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// If API key is provided via CLI, save it to config
	if *apiKeyFlag != "" {
		config.APIKey = *apiKeyFlag
		if err := saveConfig(config); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			os.Exit(1)
		}
	}

	// Use API key from config or fail
	if config.APIKey == "" {
		configPath, _ := getConfigPath()
		fmt.Printf("Error: No Anthropic API key found. Please provide one using the -api-key flag.\n")
		fmt.Printf("The API key will be saved to %s for future use.\n", configPath)
		os.Exit(1)
	}

	apiKey := config.APIKey

	// --- Initialize the logger ---
	logger := NewLogger(*debugMode)

	// --- Initialize the agent and run it ---
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}
	tools := []ToolDefinition{
		ListFilesDefinition,
		DeleteFileDefinition,
		ViewFileDefinition,
		SeeFileChunksDefinition,
		SeeGitHistoryDefinition,
		SeeFileVersionDefinition,
		EditFileChunkDefinition,
		EditFileLineDefinition,
		GitSaveChangesDefinition,
		SeeGitStatusDefinition,
	}
	agent := NewAgent(&client, getUserMessage, tools, logger)
	runErr := agent.Run(context.TODO())
	if runErr != nil {
		logger.Error("%s", runErr.Error())
	}
}

func NewAgent(client *anthropic.Client, getUserMessage func() (string, bool), tools []ToolDefinition, logger *Logger) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		tools:          tools,
		logger:         logger,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []anthropic.MessageParam{}

	a.logger.Info("Welcome to GitSynth. Use 'ctrl-c' to quit at any time.\n")
	a.logger.Info("GitSynth is now resolving your merge conflicts...\n")

	userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(DefaultPrompt))
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
					a.logger.Debug("API error occurred, retrying in %d seconds (attempt %d/%d): %v\n",
						backoffSeconds, retries+1, maxRetries, err)
					time.Sleep(time.Duration(backoffSeconds) * time.Second)
					continue
				} else { // Non-API errors are not retried
					a.logger.Debug("Non-retryable error: %v\n", err)
					break
				}
			} else {
				finalMessage = message
				break
			}
		}
		if finalErr != nil {
			a.logger.Error("%s", finalErr.Error())
			return finalErr
		}
		conversation = append(conversation, finalMessage.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}
		for _, content := range finalMessage.Content {
			switch content.Type {
			case "text":
				a.logger.AgentMessage(content.Text)
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
		a.logger.ToolResult(name, "tool not found", true)
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	a.logger.ToolCall(name, string(input))
	response, err := toolDef.Function(input)
	if err != nil {
		a.logger.ToolResult(name, err.Error(), true)
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	a.logger.ToolResult(name, response, false)
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

func GenerateSchema[T any]() anthropic.ToolInputSchemaParam {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T

	schema := reflector.Reflect(v)

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}
}
