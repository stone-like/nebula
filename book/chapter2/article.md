# Chapter 2: "Function Calling" でLLMの能力を拡張する

## はじめに

前章では、OpenAI APIと基本的な会話ができるCLIアプリケーションを作成しました。

ここからは**Function Calling**（ツール機能）を使うことで、LLMに「道具」を与え、ファイルの読み込み、計算の実行、外部APIの呼び出しなど、様々な「行動」を取らせてみましょう。

この章では、`readFile`ツールを実装し、GPTがファイルシステムを探索できるようにします。これが、自律的なコーディングエージェントの第一歩となります。

`readFile`ツールを実装することにより、「このプロジェクトのREADME.mdを読んで、どんなプロジェクトか教えて」と言うだけで、LLMが自動的にファイルを読み込み、内容を分析して回答してくれるようになっていきます！

それでは、LLMに最初の「道具」を与えていきましょう。

## ハンズオン・チュートリアル

### 1. Function Calling（Tool Calling）の概念を理解する

Function Callingとは、LLMが「特定の関数を呼び出したい」と判断した際に、その関数名と引数をJSON形式で返してくれる仕組みです。アプリケーション側では、そのリクエストを受け取って実際に関数を実行し、結果をLLMに返すことで、LLMは外部の情報や機能を活用できるようになります。

**従来の会話の流れ:**
```
User → LLM → 応答
```

**Function Calling対応後の流れ:**
```
User → LLM → ツール実行要求 → アプリ側で関数実行 → 結果をLLMに返却 → LLMが最終回答
```

OpenAI APIでは、`Tools`パラメータにJSON Schemaで関数の仕様を定義して渡すことで、LLMがその関数を使えるようになります。

### 2. OpenAI Function Calling の詳細仕様

OpenAIのFunction Calling機能は、JSON Schemaに基づいてツールを定義します。基本的な構造は以下の通りです：

```json
{
  "type": "function",
  "function": {
    "name": "function_name",
    "description": "Function description that helps the model understand when to use this tool",
    "parameters": {
      "type": "object",
      "properties": {
        "parameter_name": {
          "type": "string",
          "description": "Parameter description"
        }
      },
      "required": ["parameter_name"]
    }
  }
}
```

**重要なポイント：**

- **`name`**: 関数名（英数字とアンダースコア、64文字以内）
- **`description`**: LLMが「いつこのツールを使うべきか」を判断する重要な情報
- **`parameters`**: JSON Schema仕様に準拠したパラメータ定義
- **`required`**: 必須パラメータの配列

LLMがツールを呼び出す際は、以下のような形式でレスポンスを返します：

```json
{
  "tool_calls": [
    {
      "id": "call_abc123",
      "type": "function",
      "function": {
        "name": "readFile",
        "arguments": "{\"path\": \"sample.txt\"}"
      }
    }
  ]
}
```

:::message
**参考リンク**
- [OpenAI Function Calling 公式ドキュメント](https://platform.openai.com/docs/guides/function-calling)
:::

### 2. `readFile`ツールのJSONスキーマをGoの構造体で定義する

まず、新しいファイル`tools.go`を作成し、ツール関連の定義をまとめましょう。

```go
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// ReadFileArgs はreadFileツールの引数を表す構造体
type ReadFileArgs struct {
	Path string `json:"path" description:"読み込むファイルのパス"`
}

// ReadFileResult はreadFileツールの結果を表す構造体
type ReadFileResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// ToolDefinition はLLMが呼び出せるツールを表す構造体
type ToolDefinition struct {
	Schema   openai.Tool
	Function func(args string) (string, error)
}
```

この構造体設計のポイントを説明します：

**`ReadFileArgs`**: LLMから渡される引数を受け取る構造体です。今回は「ファイルパス」のみですが、将来的に引数が増えても拡張しやすい設計になっています。

**`ReadFileResult`**: 関数実行の結果を表します。`Content`で正常な結果を、`Error`でエラー情報を返すことで、LLMがエラーハンドリングも適切に行えます。

**`ToolDefinition`**: 「OpenAI APIに送信するスキーマ」と「実際に実行する関数」をセットで管理する構造体です。これにより、ツールの追加が簡単になります。

### 3. `readFile`の本体となる、指定されたファイルを読み込むGoの関数を実装する

次に、実際にファイルを読み込む関数を実装します。続けて`tools.go`に追加してください：

```go
// ReadFile は指定されたパスのファイル内容を読み込む
func ReadFile(args string) (string, error) {
	var readFileArgs ReadFileArgs
	if err := json.Unmarshal([]byte(args), &readFileArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	file, err := os.Open(readFileArgs.Path)
	if err != nil {
		result := ReadFileResult{
			Content: "",
			Error:   fmt.Sprintf("ファイルを開けませんでした: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		result := ReadFileResult{
			Content: "",
			Error:   fmt.Sprintf("ファイルの読み込みに失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	result := ReadFileResult{
		Content: string(content),
		Error:   "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// GetReadFileTool はreadFileツールの定義を返す
func GetReadFileTool() ToolDefinition {
	return ToolDefinition{
		Schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "readFile",
				Description: "指定されたファイルの内容全体を読み込みます。",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "読み込むファイルのパス",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		Function: ReadFile,
	}
}

// GetAvailableTools は利用可能な全てのツールを返す
func GetAvailableTools() map[string]ToolDefinition {
	return map[string]ToolDefinition{
		"readFile": GetReadFileTool(),
	}
}
```

### 関数の設計ポイント

**エラーハンドリング**: ファイルが存在しない場合でも、プログラムがクラッシュしないよう、エラー情報をJSON形式で返しています。これにより、LLMは「ファイルが見つからない」という状況も理解し、適切に対応できます。

**JSON形式での戻り値**: 関数の戻り値は常にJSON文字列にすることで、LLMが結果を構造化して理解できます。成功時は`content`に内容が、失敗時は`error`にエラーメッセージが入ります。

**Schema定義**: `GetReadFileTool()`では、OpenAI APIが理解できるJSON Schema形式でツールの仕様を定義しています。`Description`は特に重要で、LLMがいつこのツールを使うべきか判断する材料になります。

### 4. LLMからのツールコール要求を処理し、関数の結果をLLMに返す基本的な対話ループを構築する

最後に、`main.go`を拡張してFunction Calling対応の対話ループを実装します：

```go
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
```

### コードの重要な変更点

**ツールスキーマの登録**: `Tools: toolSchemas`をAPIリクエストに追加することで、LLMが利用可能なツールを認識します。

**会話履歴の管理**: `messages`配列で全ての会話履歴を保持することで、LLMが文脈を理解し続けます。

**ツールコール検出**: `responseMessage.ToolCalls`をチェックし、LLMがツールを使いたがっている場合を検出します。

**ツール実行と結果の返却**: ツール実行結果を`ChatMessageRoleTool`として履歴に追加し、再度LLMに送信することで、最終的な回答を生成させます。

## 動作確認

これでこの章の機能が動くようになりました！実際に試してみましょう。

まず、テスト用のファイルを作成します。ここではsample.txtとします。

```sample.txt
"これはテスト用のファイルです。
nebulaのreadFileツールがこのファイルを正しく読み込めるかをテストしています。

ファイルの内容:
- 行1: Hello World
- 行2: こんにちは世界
- 行3: Function Calling is working!" > sample.txt
```

プログラムをビルド:

**Linux/macOS の場合:**
```bash
go build -o nebula .
```

**Windows の場合:**
```bash
go build -o nebula.exe .
```

実行：

**Linux/macOS の場合:**
```bash
./nebula
```

**Windows の場合:**
```cmd
nebula.exe
```

成功すると、次のような出力が表示されます。

```
nebula - OpenAI Chat CLI with Function Calling
Available tools: readFile
Type 'exit' or 'quit' to end the conversation
---
You: 
```

試しにファイル読み込みを依頼すると下記のように読み込めると思います。

```
You: sample.txtの内容を読んで教えてください
Assistant is using tools...
Tool 'readFile' executed with result: {"content":"これはテスト用のファイルです。\nnebulaのreadFileツールがこのファイルを正しく読み込める...","error":""}
Assistant: sample.txtの内容を読み込みました。以下がファイルの内容です：

これはテスト用のファイルです。
nebulaのreadFileツールがこのファイルを正しく読み込めるかをテストしています。

ファイルの内容:
- 行1: Hello World
- 行2: こんにちは世界  
- 行3: Function Calling is working!

ファイルが正常に読み込まれ、Function Callingが動作していることが確認できます！

You: 存在しないファイルを読み込んでみて
Assistant is using tools...
Tool 'readFile' executed with result: {"content":"","error":"ファイルを開けませんでした: open 存在しないファイル: no such file or directory"}
Assistant: 申し訳ありませんが、指定されたファイルを読み込むことができませんでした。エラーの詳細は以下の通りです：

ファイルを開けませんでした: open 存在しないファイル: no such file or directory

ファイルが存在しないか、パスが間違っている可能性があります。正しいファイルパスを指定してください。
```

このようにLLMがツールを自動的に使い分け、エラーハンドリングも適切に行っています。

:::message alert
**トラブルシューティング**
- ツールが実行されない場合：使用しているモデルがFunction Calling対応であることを確認してください（GPT-4シリーズ推奨）
- JSON解析エラーが出る場合：ツールスキーマの定義が正しいか確認してください
:::

## この章のまとめと次のステップ

この章では、LLMに初めての「道具」を与えることに成功しました。完成したコードの構造は以下の通りです：

### 作成したファイル一覧

```
nebula/
├── go.mod              # Goモジュール定義
├── go.sum              # 依存関係のチェックサム
├── main.go             # Function Calling対応のメインプログラム
├── tools.go            # ツール定義とreadFile実装
└── sample.txt          # テスト用ファイル
```

### 達成できたこと

✅ **Function Callingの理解** - LLMがツールを使う仕組みの習得  
✅ **構造化されたツール設計** - JSON Schema + Go構造体による型安全なツール定義  
✅ **readFileツールの実装** - ファイルシステムへのアクセス機能  
✅ **エラーハンドリング** - ファイル読み込み失敗時の適切な処理  
✅ **会話履歴管理** - コンテキストを保持した対話の実現  

これで、LLMは単純な質問応答から、ファイル読み取り能力を獲得することができました。
現在は1つのツールだけですが、次のChapter 3では、`list`（ディレクトリ一覧）、`searchInDirectory`（キーワード検索）、`writeFile`（新規ファイル作成）を追加し、エージェントに探索、書き込みの能力を与えていきます。

そして、Chapter 4では最重要機能である`editFile`を実装します。これにより、nebulaは既存のコードを「読んで」「理解して」「修正する」ことができるようになります。

それではChapter3に行きましょう！