package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

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
	tools := GetAvailableTools()
	
	// ツールのスキーマを配列に変換
	var toolSchemas []openai.Tool
	for _, tool := range tools {
		toolSchemas = append(toolSchemas, tool.Schema)
	}

	fmt.Println("nebula - OpenAI Chat CLI with Function Calling")
	fmt.Println("Available tools: readFile")
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

		// ユーザーメッセージを履歴に追加
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: userInput,
		})

		// OpenAI APIに送信（ツールを含む）
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
			continue
		}

		if len(resp.Choices) == 0 {
			fmt.Println("No response received from OpenAI")
			continue
		}

		responseMessage := resp.Choices[0].Message
		messages = append(messages, responseMessage)

		// ツールコールがある場合の処理
		if len(responseMessage.ToolCalls) > 0 {
			fmt.Println("Assistant is using tools...")
			
			for _, toolCall := range responseMessage.ToolCalls {
				if tool, exists := tools[toolCall.Function.Name]; exists {
					// ツール関数を実行
					result, err := tool.Function(toolCall.Function.Arguments)
					if err != nil {
						result = fmt.Sprintf(`{"error": "Tool execution failed: %v"}`, err)
					}

					// ツール実行結果をメッセージ履歴に追加
					messages = append(messages, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    result,
						ToolCallID: toolCall.ID,
					})

					fmt.Printf("Tool '%s' executed with result: %s\n", toolCall.Function.Name, result)
				}
			}

			// ツール実行後、再度APIを呼び出して最終回答を取得
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
				continue
			}

			if len(resp.Choices) > 0 {
				finalMessage := resp.Choices[0].Message
				messages = append(messages, finalMessage)
				fmt.Printf("Assistant: %s\n\n", finalMessage.Content)
			}
		} else {
			// 通常の会話応答
			fmt.Printf("Assistant: %s\n\n", responseMessage.Content)
		}
	}
}
