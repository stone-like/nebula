package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"nebula/tools"

	"github.com/sashabaranov/go-openai"
)

// executeToolCall は単一のツールコールを実行する
func executeToolCall(toolCall openai.ToolCall, toolsMap map[string]tools.ToolDefinition) openai.ChatCompletionMessage {
	if tool, exists := toolsMap[toolCall.Function.Name]; exists {
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
func processToolCalls(toolCalls []openai.ToolCall, toolsMap map[string]tools.ToolDefinition) []openai.ChatCompletionMessage {
	var toolMessages []openai.ChatCompletionMessage

	for _, toolCall := range toolCalls {
		toolMessage := executeToolCall(toolCall, toolsMap)
		toolMessages = append(toolMessages, toolMessage)
	}

	return toolMessages
}

// handleConversation はLLMとの対話セッションを処理する（完全な1回追加呼び出し版）
func handleConversation(client *openai.Client, toolSchemas []openai.Tool, toolsMap map[string]tools.ToolDefinition, userInput string, messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	// ユーザーメッセージを履歴に追加
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})

	// 最初のAPI呼び出し
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT4Dot1Nano,
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
			toolMessages := processToolCalls(responseMessage.ToolCalls, toolsMap)
			messages = append(messages, toolMessages...)

			// 次のAPI呼び出し
			resp, err = client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    openai.GPT4Dot1Nano,
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
			break
		}
	}

	return messages
}

func main() {
	// 環境変数からAPIキーを取得
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set")
		fmt.Println("Please set your OpenAI API key: export OPENAI_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	// OpenAIクライアントを初期化
	client := openai.NewClient(apiKey)

	// 利用可能なツールを取得
	toolsMap := tools.GetAvailableTools()

	// ツールのスキーマを配列に変換
	var toolSchemas []openai.Tool
	for _, tool := range toolsMap {
		toolSchemas = append(toolSchemas, tool.Schema)
	}

	fmt.Println("nebula - OpenAI Chat CLI with Function Calling")
	fmt.Println("Available tools: readFile, list, searchInDirectory, writeFile, editFile")
	fmt.Println("Type 'exit' or 'quit' to end the conversation")
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)

	// 会話履歴を保持
	var messages []openai.ChatCompletionMessage

	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// 終了コマンドをチェック
		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		if userInput == "" {
			continue
		}

		// 対話セッションを処理
		messages = handleConversation(client, toolSchemas, toolsMap, userInput, messages)
	}
}
