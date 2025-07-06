package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"nebula/config"
	"nebula/memory"
	"nebula/tools"

	"github.com/sashabaranov/go-openai"
)

// executeToolCall は単一のツールコールを実行する
func executeToolCall(toolCall openai.ToolCall, toolsMap map[string]tools.ToolDefinition, planMode bool) openai.ChatCompletionMessage {
	if tool, exists := toolsMap[toolCall.Function.Name]; exists {
		// planモードでは書き込み系ツールの実行を制限
		if planMode && (toolCall.Function.Name == "writeFile" || toolCall.Function.Name == "editFile") {
			result := fmt.Sprintf(`{"error": "Tool '%s' is not allowed in plan mode. Plan mode is read-only."}`, toolCall.Function.Name)
			fmt.Printf("Plan mode: Blocked execution of '%s'\n", toolCall.Function.Name)
			return openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			}
		}

		fmt.Printf("Executing tool: %s with arguments: %s\n", toolCall.Function.Name, toolCall.Function.Arguments)

		result, err := tool.Function(toolCall.Function.Arguments)
		if err != nil {
			result = fmt.Sprintf(`{"error": "Tool execution failed: %v"}`, err)
			fmt.Printf("Tool execution error: %v\n", err)
		}

		fmt.Printf("Tool '%s' executed with result: %s\n", toolCall.Function.Name, result)

		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result,
			ToolCallID: toolCall.ID,
		}
	} else {
		fmt.Printf("Unknown tool requested: %s\n", toolCall.Function.Name)
		return openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    fmt.Sprintf(`{"error": "Unknown tool: %s"}`, toolCall.Function.Name),
			ToolCallID: toolCall.ID,
		}
	}
}

// processToolCalls は複数のツールコールを処理する
func processToolCalls(toolCalls []openai.ToolCall, toolsMap map[string]tools.ToolDefinition, planMode bool) []openai.ChatCompletionMessage {
	var toolMessages []openai.ChatCompletionMessage

	for _, toolCall := range toolCalls {
		toolMessage := executeToolCall(toolCall, toolsMap, planMode)
		toolMessages = append(toolMessages, toolMessage)
	}

	return toolMessages
}

// getSystemPrompt はnebulaエージェント用のシステムプロンプトを返す
func getSystemPrompt() string {
	return `# Role
You are "nebula", an expert software developer and autonomous coding agent.

# Critical Rules (Non-Negotiable)
1. **NEVER assume or guess file contents, names, or locations** - You must explore to understand them
2. **Information gathering is MANDATORY before implementation** - Guessing leads to immediate failure
3. **Before using writeFile or editFile, you MUST have used readFile on reference files**
4. **NEVER ask for permission between steps** - Proceed automatically through the entire workflow
5. **Complete the entire task in one continuous flow** - No pausing for confirmation

# Why Information Gathering is Critical
- **File structures vary**: What you expect vs. what exists are often different
- **Extensions matter**: .js vs .ts vs .go vs .py affects implementation
- **Directory layout matters**: Different projects have different organization
- **Assumption costs**: Guessing wrong means complete rework

# Execution Protocol
When you receive a request, follow this mandatory sequence and proceed automatically without asking for permission:

## Step 1: Information Gathering (Required, but proceed automatically)
- **Discover project structure**: Use 'list' to understand what files exist and their organization when working with multiple files or unclear requirements
- **Use 'readFile'**: Read ALL reference files mentioned in the request to understand actual content
- **Use 'searchInDirectory'**: Find related files when unsure about locations or patterns
- **Verify reality**: What you discover often differs from assumptions

**Internal Verification (check silently, do not ask user):**
□ Have I discovered the project structure when needed? (Required: YES when ambiguous)
□ Have I read the reference file contents with readFile? (Required: YES)
□ Do I understand the existing code structure? (Required: YES)
□ Have I gathered all necessary information? (Required: YES)

## Step 2: Implementation (Proceed automatically after Step 1)
- Use 'writeFile' for new file creation
- Use 'editFile' for existing file modification
- Complete all related changes

**IMPORTANT: Proceed from Step 1 to Step 2 automatically without asking for permission or confirmation.**

# Common Mistakes to Avoid
❌ **FORBIDDEN**: Guessing file names (e.g., assuming "todo.ts" exists without checking)
❌ **FORBIDDEN**: Guessing file extensions (e.g., assuming .js when it might be .ts)
❌ **FORBIDDEN**: Guessing directory structure (e.g., assuming files are in "src/" without checking)
❌ **FORBIDDEN**: Seeing "refer to X file" and implementing without actually reading X
❌ **FORBIDDEN**: Using your knowledge to guess file contents
❌ **FORBIDDEN**: Skipping the readFile step because the task seems simple
❌ **FORBIDDEN**: Asking "Should I proceed with implementation?" after information gathering
❌ **FORBIDDEN**: Pausing for confirmation between information gathering and implementation

# Why Guessing Fails
- **Wrong file extension**: Implementing .js when the project uses .ts
- **Wrong directory**: Creating files in wrong locations breaks project structure  
- **Wrong patterns**: Assuming patterns that don't match the actual codebase
- **Wasted effort**: Implementation based on wrong assumptions requires complete rework

# Execution Examples

## Example 1: File Extension Discovery
Request: "Add a todo feature to the app"
**Correct sequence:**
1. list(".") ← Discover if files are .js, .ts, .py, .go, etc.
2. Find actual todo-related files with search or list
3. readFile the discovered files to understand patterns
4. Implement using the correct extension and patterns

**Incorrect sequence:**
1. writeFile("todo.ts", ...) ← FORBIDDEN: Guessed .ts without checking

## Example 2: Reference File Reading  
Request: "Create tools/copyFile.go based on tools/writeFile.go"
**Correct sequence:**
1. readFile("tools/writeFile.go") ← MANDATORY FIRST STEP
2. Analyze the content and structure (silently)
3. writeFile("tools/copyFile.go", <complete_implementation>) ← PROCEED AUTOMATICALLY

**Incorrect sequence:**
1. writeFile("tools/copyFile.go", ...) ← FORBIDDEN: Implemented without reading reference

## Example 3: Directory Structure Discovery
Request: "Add authentication middleware"
**Correct sequence:**
1. list(".") ← Discover project structure
2. list("src/") or searchInDirectory("middleware") ← Find where middleware belongs
3. readFile existing middleware files to understand patterns
4. Implement in the correct location with correct patterns

**Incorrect sequence:**
1. writeFile("src/middleware/auth.js", ...) ← FORBIDDEN: Guessed directory structure

# Your Responsibility
Complete the entire task following this protocol in one continuous flow. No shortcuts, no assumptions, no guessing, and no asking for permission between steps.`
}

// handleConversation はLLMとの対話セッションを処理する
func handleConversation(client *openai.Client, cfg *config.Config, memoryManager *memory.Manager, toolSchemas []openai.Tool, toolsMap map[string]tools.ToolDefinition, userInput string, messages []openai.ChatCompletionMessage, planMode bool) []openai.ChatCompletionMessage {
	// システムプロンプトが設定されていない場合は最初に追加
	// （復元されたメッセージにはシステムプロンプトが含まれていない可能性があるため）
	hasSystemPrompt := false
	if len(messages) > 0 && messages[0].Role == openai.ChatMessageRoleSystem {
		hasSystemPrompt = true
	}

	if !hasSystemPrompt {
		// システムプロンプトを先頭に追加
		// systemMessage := openai.ChatCompletionMessage{
		// 	Role:    openai.ChatMessageRoleSystem,
		// 	Content: getSystemPrompt(),
		// }
		// messages = append([]openai.ChatCompletionMessage{systemMessage}, messages...)
	}

	// ユーザーメッセージを履歴に追加
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	// メモリに保存
	memoryManager.SaveMessage("user", userInput, nil, nil)

	// 最初のAPI呼び出し
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    cfg.GetOpenAIModel(),
			Messages: messages,
			Tools:    toolSchemas,
		},
	)

	if err != nil {
		fmt.Printf("Error calling OpenAI API: %v\n", err)
		return messages
	}

	if len(resp.Choices) == 0 {
		fmt.Println("No response received from OpenAI")
		return messages
	}

	// レスポンスを処理するループ
	for {
		responseMessage := resp.Choices[0].Message
		messages = append(messages, responseMessage)

		// ツールコールがある場合の処理
		if len(responseMessage.ToolCalls) > 0 {
			fmt.Println("Assistant is using tools...")

			// ツールを実行して結果をメッセージ履歴に追加
			toolMessages := processToolCalls(responseMessage.ToolCalls, toolsMap, planMode)
			messages = append(messages, toolMessages...)

			// 次のAPI呼び出し
			resp, err = client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    cfg.GetOpenAIModel(),
					Messages: messages,
					Tools:    toolSchemas,
				},
			)

			if err != nil {
				fmt.Printf("Error calling OpenAI API after tool execution: %v\n", err)
				break
			}

			if len(resp.Choices) == 0 {
				break
			}
		} else {
			// ツールコールがない場合は最終応答
			fmt.Printf("Assistant: %s\n\n", responseMessage.Content)

			// アシスタントメッセージをメモリに保存
			memoryManager.SaveMessage("assistant", responseMessage.Content, nil, nil)
			break
		}
	}

	return messages
}

// handleModelSwitch handles interactive model switching
func handleModelSwitch(cfg *config.Config) {
	fmt.Printf("Current model: %s\n", cfg.Model)
	fmt.Println("Available models:")
	fmt.Println("1. gpt-4.1-nano (default, faster)")
	fmt.Println("2. gpt-4.1-mini (complex tasks)")
	fmt.Print("Select model (1 or 2): ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		var newModel string

		switch choice {
		case "1":
			newModel = "gpt-4.1-nano"
		case "2":
			newModel = "gpt-4.1-mini"
		default:
			fmt.Println("Invalid choice. No changes made.")
			return
		}

		if err := cfg.SetModel(newModel); err != nil {
			fmt.Printf("Error setting model: %v\n", err)
		} else {
			fmt.Printf("Model switched to: %s\n", newModel)
		}
	}
}

// handleModeSwitch handles interactive mode switching
func handleModeSwitch(planMode *bool) {
	currentMode := "AGENT"
	if *planMode {
		currentMode = "PLAN"
	}

	fmt.Printf("Current mode: %s\n", currentMode)
	fmt.Println("Available modes:")
	fmt.Println("1. AGENT (full capabilities)")
	fmt.Println("2. PLAN (read-only, safe exploration)")
	fmt.Print("Select mode (1 or 2): ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			*planMode = false
			fmt.Println("Mode switched to: AGENT")
		case "2":
			*planMode = true
			fmt.Println("Mode switched to: PLAN")
		default:
			fmt.Println("Invalid choice. No changes made.")
		}
	}
}

func main() {
	// デフォルトはagentモード
	planMode := false

	// 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// APIキーが設定されているかチェック
	if cfg.APIKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set")
		fmt.Println("Please set your OpenAI API key: export OPENAI_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	// メモリマネージャーを初期化
	memoryManager, err := memory.NewManager(cfg.DatabasePath)
	if err != nil {
		fmt.Printf("Error initializing memory: %v\n", err)
		os.Exit(1)
	}
	defer memoryManager.Close()

	// プロジェクトディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// セッション管理
	var messages []openai.ChatCompletionMessage
	var currentSession *memory.Session

	// 既存セッションを表示
	sessions, err := memoryManager.GetCurrentProjectSessions(5)
	if err != nil {
		fmt.Printf("Error loading sessions: %v\n", err)
	} else if len(sessions) > 0 {
		fmt.Printf("Found %d previous sessions for this project:\n", len(sessions))
		for i, session := range sessions {
			status := "completed"
			if session.EndedAt == nil {
				status = "active"
			}
			lastMsg := session.LastMessage
			if len(lastMsg) > 50 {
				lastMsg = lastMsg[:50] + "..."
			}
			fmt.Printf("%d. %s (%s) - %s\n", i+1, session.ID, status, lastMsg)
		}
		fmt.Print("Start new session or restore (new/1-5): ")

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			choice := strings.TrimSpace(scanner.Text())

			if choice != "new" && choice != "" {
				// セッション番号をパース
				if sessionIndex, err := strconv.Atoi(choice); err == nil {
					if sessionIndex >= 1 && sessionIndex <= len(sessions) {
						selectedSession := sessions[sessionIndex-1]

						// セッションを復元
						restoredSession, err := memoryManager.RestoreSession(selectedSession.ID)
						if err != nil {
							fmt.Printf("Error restoring session: %v\n", err)
						} else {
							currentSession = restoredSession
							fmt.Printf("Restored session: %s\n", restoredSession.ID)

							// 過去の会話履歴を読み込み
							memoryMessages, err := memoryManager.GetSessionMessages(selectedSession.ID)
							if err != nil {
								fmt.Printf("Error loading session messages: %v\n", err)
							} else {
								// OpenAI形式に変換
								messages = convertToOpenAIMessages(memoryMessages)
								fmt.Printf("Loaded %d previous messages\n", len(messages))
							}
						}
					} else {
						fmt.Println("Invalid session number. Starting new session.")
					}
				} else {
					fmt.Println("Invalid input. Starting new session.")
				}
			}
		}
	}

	// 新しいセッションを開始（復元しなかった場合）
	if currentSession == nil {
		session, err := memoryManager.StartSession(currentDir, cfg.Model)
		if err != nil {
			fmt.Printf("Error starting session: %v\n", err)
		} else {
			currentSession = session
			fmt.Printf("Started new session: %s\n", session.ID)
		}
	}

	// OpenAIクライアントを初期化
	client := openai.NewClient(cfg.APIKey)

	// 利用可能なツールを取得
	toolsMap := tools.GetAvailableTools()

	// ツールのスキーマを配列に変換
	var toolSchemas []openai.Tool
	for _, tool := range toolsMap {
		toolSchemas = append(toolSchemas, tool.Schema)
	}

	fmt.Println("nebula - OpenAI Chat CLI with Function Calling")
	fmt.Printf("Current model: %s\n", cfg.Model)
	fmt.Println("Memory: enabled")
	fmt.Println("Mode: AGENT (full capabilities)")
	fmt.Println("Available tools: readFile, list, searchInDirectory, writeFile, editFile")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  'exit' or 'quit' - End the conversation")
	fmt.Println("  'model' - Switch between models")
	fmt.Println("  'mode' - Interactive mode switching")
	fmt.Println("  'plan' - Switch to PLAN mode (read-only)")
	fmt.Println("  'agent' - Switch to AGENT mode (full capabilities)")
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// 現在のモードを表示
		modeIndicator := "AGENT"
		if planMode {
			modeIndicator = "PLAN"
		}
		fmt.Printf("[%s] You: ", modeIndicator)
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// 終了コマンドをチェック
		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		// モデル切り替えコマンドをチェック
		if userInput == "model" {
			handleModelSwitch(cfg)
			continue
		}

		// モード切り替えコマンドをチェック
		if userInput == "mode" {
			handleModeSwitch(&planMode)
			continue
		}

		// 簡単なモード切り替えコマンド
		if userInput == "plan" {
			planMode = true
			fmt.Println("Mode switched to: PLAN (read-only)")
			continue
		}
		if userInput == "agent" {
			planMode = false
			fmt.Println("Mode switched to: AGENT (full capabilities)")
			continue
		}

		if userInput == "" {
			continue
		}

		// 対話セッションを処理
		messages = handleConversation(client, cfg, memoryManager, toolSchemas, toolsMap, userInput, messages, planMode)
	}
}

// convertToOpenAIMessages converts memory messages to OpenAI format
func convertToOpenAIMessages(memoryMessages []*memory.Message) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage

	for _, msg := range memoryMessages {
		// Skip tool messages for now (they are complex to restore properly)
		if msg.Role == "tool" {
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	return messages
}
