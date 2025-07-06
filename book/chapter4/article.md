# Chapter 4: 最重要機能 `editFile` 

## はじめに

この章では、ついにエージェントに「既存ファイルを編集する能力」を与えます。
これは単なる機能追加ではなく、コーディングのための最重要機能となっています。

しかし、ファイル編集は非常にデリケートな操作です。一歩間違えれば、大切なコードを壊してしまう危険性があります。
そこで今回、**「Read-Modify-Write」**という安全で確実なパターンを学び、実装していきます。


この章を終える頃には、nebulaエージェントは以下のことができるようになります。

- 既存ファイルを安全に編集する
- ユーザーの許可を得てから危険な操作を実行する
- 複数のツールを連続して呼び出す
- Read-Modify-Writeパターンを自然に実行する


## 一旦リファクタリング

### Chapter 3終了時の状況

Chapter 3では、4つのツール（`readFile`, `list`, `searchInDirectory`, `writeFile`）を`tools.go`単一ファイルで実装しました。
これにより、nebulaエージェントは基本的なファイル操作能力を獲得しましたが、tools.goが肥大化してしまいました。
ですので、editツールを作成する前にここで一旦リファクタリングをしてしまいましょう。

**現在のファイル構造:**
```
nebula/
├── main.go                 # メインプログラム
├── tools.go               # 全ツール実装（約400行）
├── go.mod
└── go.sum
```

**Chapter 4目標構造:**
```
nebula/
├── main.go                 # メインプログラム
├── tools/                  # ツールパッケージ
│   ├── common.go          # 共通型定義
│   ├── readfile.go        # readFileツール
│   ├── list.go            # listツール
│   ├── search.go          # searchInDirectoryツール
│   ├── writefile.go       # writeFileツール
│   ├── editfile.go        # editFileツール（新規）
│   └── registry.go        # ツール登録管理
├── go.mod
└── go.sum
```

### 1. toolsディレクトリの作成

まず、新しいパッケージ用のディレクトリを作成します。

```bash
mkdir tools
```

### 2. 共通型定義の分離

`tools/common.go`を作成し、共通の型定義を配置します。

```go
package tools

import (
	"github.com/sashabaranov/go-openai"
)

// ToolDefinition はLLMが呼び出せるツールを表す構造体
type ToolDefinition struct {
	Schema   openai.Tool
	Function func(args string) (string, error)
}
```

### 3. 既存ツールの分割

Chapter 3で実装した各ツールを独立したファイルに分割します。

**tools/readfile.go** (Chapter 2から移行)
```go
package tools

import (
	// 必要なimport...
)

// ReadFileArgs, ReadFileResult, ReadFile関数, GetReadFileTool関数
```

**tools/list.go** (Chapter 3から移行)
```go
package tools

import (
	// 必要なimport...
)

// ListArgs, ListResult, List関数, GetListTool関数
```

**tools/search.go** (Chapter 3から移行)
```go
package tools

import (
	// 必要なimport...
)

// SearchInDirectoryArgs, SearchInDirectoryResult, SearchInDirectory関数, GetSearchInDirectoryTool関数
```

**tools/writefile.go** (Chapter 3から移行)
```go
package tools

import (
	// 必要なimport...
)

// WriteFileArgs, WriteFileResult, WriteFile関数, GetWriteFileTool関数
```

### 4. ツール登録管理の作成

`tools/registry.go`を作成し、全ツールの登録管理を行います。

```go
package tools

// GetAvailableTools は利用可能な全てのツールを返す
func GetAvailableTools() map[string]ToolDefinition {
	return map[string]ToolDefinition{
		"readFile":           GetReadFileTool(),
		"list":               GetListTool(),
		"searchInDirectory":  GetSearchInDirectoryTool(),
		"writeFile":          GetWriteFileTool(),
		"editFile":           GetEditFileTool(), // 新規追加
	}
}
```

### 5. main.goの更新

`main.go`のimport文を更新し、新しいパッケージ構造に対応します。

```go
import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
	"nebula/tools" // 新しいパッケージ
)
```

ツール取得部分も更新。

```go
// 利用可能なツールを取得
toolsMap := tools.GetAvailableTools()

// ツールのスキーマを配列に変換
var toolSchemas []openai.Tool
for _, tool := range toolsMap {
	toolSchemas = append(toolSchemas, tool.Schema)
}
```


## editFile実装

さて、リファクタリングが完了したので、いよいよ本章の核心である`editFile`ツールを実装していきます。

### Step 1: `editFile`ツールの実装

:::message alert
⚠️ **重要：editFileツールの危険性**
`editFile`は既存ファイルを完全に上書きする非常に破壊的な操作です。ですので実行前に必ずユーザーの許可を得る設計としていきます。
ただeditFile完成後に、実際のeditの際はgit管理下に置きリセット可能な状態にすること推奨です。
:::

モジュラー構造への移行が完了したら、いよいよ`editFile`ツールを実装します。

`tools/editfile.go`を作成してください。

```go
package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// EditFileArgs はeditFileツールの引数を表す構造体
type EditFileArgs struct {
	Path       string `json:"path" description:"編集するファイルのパス"`
	NewContent string `json:"new_content" description:"ファイルの新しい内容（完全な内容）"`
}

// EditFileResult はeditFileツールの結果を表す構造体
type EditFileResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// EditFile は既存ファイルの内容を完全に上書きする（ユーザー許可が必要）
func EditFile(args string) (string, error) {
	var editArgs EditFileArgs
	if err := json.Unmarshal([]byte(args), &editArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	// ファイルが存在するかチェック
	if _, err := os.Stat(editArgs.Path); os.IsNotExist(err) {
		result := EditFileResult{
			Success: false,
			Error:   "ファイルが存在しません。新しいファイルの作成にはwriteFileを使用してください。",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// ユーザーに許可を求める
	fmt.Printf("\n既存ファイルを編集します: %s\n", editArgs.Path)
	fmt.Print("実行してもよろしいですか？ (y/N): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		result := EditFileResult{
			Success: false,
			Error:   "ユーザー入力の読み取りに失敗しました",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	userResponse := strings.TrimSpace(scanner.Text())
	if userResponse != "y" && userResponse != "Y" {
		result := EditFileResult{
			Success: false,
			Error:   "ユーザーによってキャンセルされました",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// ファイルを開いて完全に上書き
	file, err := os.Create(editArgs.Path)
	if err != nil {
		result := EditFileResult{
			Success: false,
			Error:   fmt.Sprintf("ファイルのオープンに失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}
	defer file.Close()

	// 新しい内容を書き込み
	if _, err := file.WriteString(editArgs.NewContent); err != nil {
		result := EditFileResult{
			Success: false,
			Error:   fmt.Sprintf("ファイルの書き込みに失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	result := EditFileResult{
		Success: true,
		Error:   "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// GetEditFileTool はeditFileツールの定義を返す
func GetEditFileTool() ToolDefinition {
	return ToolDefinition{
		Schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name: "editFile",
				Description: "既存ファイルの内容を完全に上書きします。重要: ファイルを破壊しないために、必ず以下のワークフローに従ってください: 1. 'readFile'を使用して現在の完全な内容を取得する。2. 思考プロセスで、読み取った内容を基に新しいファイルの完全版を構築する。3. このツールを使用して完全な新しい内容を書き込む。部分的な編集には使用しないでください。常にファイル全体の内容を提供してください。実行前にユーザーの許可を求めます。",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "編集する既存ファイルのパス",
						},
						"new_content": {
							Type:        jsonschema.String,
							Description: "既存ファイル全体を上書きする新しい完全な内容",
						},
					},
					Required: []string{"path", "new_content"},
				},
			},
		},
		Function: EditFile,
	}
}
```

**ポイント解説**：

1. **ユーザー許可システム**: `writeFile`と同様に、危険な操作の前にユーザーの確認を求めます
2. **ファイル存在チェック**: 新規作成と編集を明確に分離しています
3. **完全上書き**: `os.Create`を使用してファイル全体を置き換えます

次に、このツールをシステムに登録しましょう。`tools/registry.go`を編集します。

```go
package tools

// GetAvailableTools は利用可能な全てのツールを返す
func GetAvailableTools() map[string]ToolDefinition {
	return map[string]ToolDefinition{
		"readFile":           GetReadFileTool(),
		"list":               GetListTool(),
		"searchInDirectory":  GetSearchInDirectoryTool(),
		"writeFile":          GetWriteFileTool(),
		"editFile":           GetEditFileTool(),  // 新しく追加
	}
}
```

そして、`main.go`の利用可能ツールの表示も更新します。

```go
fmt.Println("Available tools: readFile, list, searchInDirectory, writeFile, editFile")
```

### Step 2:  なぜ部分編集ではなく、ファイル全体の上書きを選んだのか

ここで、なぜ「ファイル全体の上書き」という一見非効率的な方法を選んだのかを説明します。

#### 部分編集ツールとは

部分編集ツールとは、以下のようなものです。

```go
// 行番号指定編集
func EditLines(filePath string, startLine, endLine int, newContent string)

// 正規表現置換
func ReplacePattern(filePath string, pattern, replacement string)

// 差分適用
func ApplyPatch(filePath string, patch string)
```

部分編集では、以下のようなエラーが頻発します。

- 構文エラーを引き起こす不完全な編集
- インデントの不整合
- インポート文の重複

これはプロンプトを念入りに作成したり、編集に失敗した際、失敗内容別にコード上でLLMに修正依頼を行う等で回避が可能ですが、実装や理解が難しいです。
例えばGeminiCLIでは編集行の前後数行のコンテキストを要求する手法を採用していますが、なかなか理解が難しいので今回は見送り、
パフォーマンスやコストは悪いものの、理解しやすい方法としました。

:::message
**参考リンク**
- [Gemini CLI ソースコード](https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/tools/edit.ts)
:::


### Step 2: Function Calling Loopの実装

突然ですが、実装した`editFile`ツールをテストする前に、極めて重要な問題を解決する必要があります。
それは**「readFileの後にeditFileが呼ばれない」**という根本的な問題です。

Chapter 2-3の実装では、各ツールが単発で実行されることを前提としていました。
しかし、実際のコーディング作業では、**複数のツールを連続して使用する**ことが必要です。

**具体的な失敗例：Read-Modify-Write パターン**

ユーザーが「sample.txtファイルの内容を変更してください」と依頼した場合：

```
期待される動作:
1. ユーザー入力: "sample.txtの内容を変更してください"
2. LLM判断: "まずファイルを読む必要がある" → readFile実行
3. readFile結果: "現在の内容: Hello"
4. LLM判断: "内容を変更しよう" → editFile実行
5. editFile実行: ファイル更新完了

現在の実装での実際の動作:
1. ユーザー入力: "sample.txtの内容を変更してください"
2. LLM判断: "まずファイルを読む必要がある" → readFile実行
3. readFile結果: "現在の内容: Hello"
4. LLM判断: "editFileを実行する必要がある" → ツールコール生成
5. ⚠️ 問題: ツールコールを検出するが実行せずに終了
```

#### 技術的な原因分析

問題は下記のように**1度のツール実行**で終了してしまっている部分にあります。

**Chapter 2-3の`main.go`実装:**
```go
// 1回目のAPI呼び出し
resp, err := client.CreateChatCompletion(...)
responseMessage := resp.Choices[0].Message

// ツールコールがある場合のみ実行
if len(responseMessage.ToolCalls) > 0 {
    // ツール実行後、再度APIを呼び出して最終回答を取得
    resp, err = client.CreateChatCompletion(
          context.Background(),
          openai.ChatCompletionRequest{
             Model:    openai.GPT4Dot1Nano,
             Messages: messages,
             Tools:    toolSchemas,
            },
           )
    ...
    if len(resp.Choices) > 0 {
          finalMessage := resp.Choices[0].Message
          messages = append(messages, finalMessage)
          fmt.Printf("Assistant: %s\n\n", finalMessage.Content)   <-- ここで終了！
     }
}
```

**問題の本質:**
- LLMは「editFileを実行してください」という**指示**を返す
- しかし、**実行は行われない**
- ユーザーには「editFileを実行してください」というメッセージが表示されるだけ


この問題を解決するには、**ツールコールがなくなるまで繰り返し実行する**ループが必要です。

```go
// 改善後の実装概念
for {
    resp, err := client.CreateChatCompletion(...)
    responseMessage := resp.Choices[0].Message
    
    if len(responseMessage.ToolCalls) > 0 {
        // ツール実行
        // ループ継続 - 次のAPI呼び出しへ
    } else {
        // ツールコールがない = 最終応答
        fmt.Println(responseMessage.Content)
        break // ループ終了
    }
}
```

#### Read-Modify-Write パターンの実現

Function Calling Loopにより、以下の自然なワークフローが可能になります。

```
ユーザー: "config.jsonファイルでdatabase_pathを更新してください"
    ↓
LLM: readFileでconfig.jsonを読み込みます
    ↓ (ループ1回目)
readFile実行: {"model": "gpt-4.1-nano"}
    ↓
LLM: 現在の内容を確認しました。database_pathを追加してeditFileで更新します
    ↓ (ループ2回目)
editFile実行: 新しい内容で更新
    ↓
LLM: ファイルの更新が完了しました
    ↓ (ループ終了)
```

この連続したツール呼び出しを可能にするため、`main.go`を**Function Calling Loop**対応に改修しましょう。

#### ツール実行の関数分離

まず、ツール実行の関数を分離します。

```go
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
```

次にLLMとのやり取り部分をhandleConversation関数に抽出し、レスポンスを処理を無限ループにします。


```go
// handleConversation はLLMとの対話セッションを処理する
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
```

そして、main関数内の複雑な処理を置き換えます。

```go
		// 対話セッションを処理
		messages = handleConversation(client, toolSchemas, toolsMap, userInput, messages)
```

### Step 3: Read-Modify-Writeパターンの動作確認

Function Callingループ実装が完了したら、実際にRead-Modify-Writeパターンの動作を確認してみましょう。

まず、テスト用のファイルを作成します。

**Linux/macOS の場合:**
```bash
echo "Hello World" > sample.txt
```

**Windows の場合:**
```cmd
echo Hello World > sample.txt
```

次に、プロジェクトをビルドします。

**Linux/macOS の場合:**
```bash
go build -o nebula .
```

**Windows の場合:**
```bash
go build -o nebula.exe .
```

そして実行します。

**Linux/macOS の場合:**
```bash
./nebula
```

**Windows の場合:**
```cmd
nebula.exe
```

実行後、以下の指示を与えてみてください。

```
sample.txtのHello WorldをHello Nebulaに変更してください
```

今度は、以下のような完璧なRead-Modify-Writeパターンが観察できるはずです。

```
Assistant is using tools...
Executing tool: readFile with arguments: {"path":"sample.txt"}
Tool 'readFile' executed with result: {"content":"Hello World\n"}

Assistant is using tools...  // 追加のツール使用
Executing tool: editFile with arguments: {"path":"sample.txt","new_content":"Hello Nebula\n"}

既存ファイルを編集します: sample.txt
実行してもよろしいですか？ (y/N): y

Tool 'editFile' executed with result: {"success":true}
Assistant: sample.txtファイルの内容を「Hello World」から「Hello Nebula」に変更しました。編集が正常に完了しました。
```

以下のポイントを押さえて動作していることが確認できると思います！
- readFileが最初に実行される
- その結果を受けて、LLMが新しい内容を構築する
- editFileが自動的に呼び出される
- ユーザーの許可を得てから実際の編集が実行される

#### 追加テスト：ファイルに新しい行を追加

続いて、以下のテストを実行してみてください。

```
sample.txtにNewContentという文を追加してください
```

正常に動作していれば、以下のような流れでRead-Modify-Writeパターンが自然に実行されます。

```
Assistant is using tools...
Executing tool: readFile with arguments: {"path":"sample.txt"}
Tool 'readFile' executed with result: {"content":"Hello Nebula\n"}

Assistant is using tools...
Executing tool: editFile with arguments: {"path":"sample.txt","new_content":"Hello Nebula\n新しい行\n"}

既存ファイルを編集します: sample.txt
実行してもよろしいですか？ (y/N): y

Tool 'editFile' executed with result: {"success":true}
Assistant: sample.txtに新しい行を追加しました。
```

**Read-Modify-Writeパターンの重要なポイント**：
- `readFile`で現在の内容を取得
- 既存内容に新しい行を追加した完全なファイルを構築
- `editFile`で完全な新しい内容に置き換え

この動作により、ファイルの内容が安全に保たれながら編集が実行されます。


### Step 4: 現在のエージェントの限界と次のステップ

現在のエージェントは基本的なファイル操作ができるようになりましたが、まだ重要な限界があります。

#### 複雑なタスクでの問題

以下のような複雑なタスクを試してみてください。

```
tools/writeFile.goを参考に、tools/copyFile.goを作成してください。ファイルをコピーする機能を実装し、tools/registry.goに登録してください
```

現在の実装では、以下のような問題が発生します。

1. **体系的な探索の欠如**: 参考ファイルを読まずにいきなり実装を始める
2. **プロジェクト理解の不足**: 既存のコードスタイルやパターンを理解しない
3. **段階的な実装プロセスの欠如**: 計画→実装→検証の流れがない

#### なぜこれらの限界が存在するのか

現在のnebulaには**システムプロンプト**が欠けています。システムプロンプトは、エージェントの「思考プロセス」や「行動指針」を定義する重要な仕組みです。

**システムプロンプトで実現できること**：
- 体系的な探索→計画→実装の段階的プロセス
- プロジェクト構造の理解と一貫性の確保
- エラーハンドリングと品質管理
- 明確な役割と制約に基づく判断

#### 次のChapterでの進化

Chapter 5では、これらの限界を克服するために**システムプロンプト**を実装します。
システムプロンプトにより、nebulaは単なるツール実行者から、本格的な**コーディングエージェント**へと進化します。

現在のnebulaでも基本的なファイル操作は可能ですが、次のChapterでより高度で信頼性の高いエージェントになる基盤が整いました。


## この章のまとめと次のステップ

### 達成したこと

この章では、エージェントに重要な新機能を追加しました。

**実装した機能**：
- **`editFile`ツール**: 既存ファイルの安全な編集
- **ユーザー許可システム**: 危険な操作の事前確認
- **Function Callingループ**: 複数ツールの連続実行
- **モジュラー構造**: ツールの分割と管理の改善

**学んだ重要概念**：
- **Read-Modify-Writeパターン**: 安全で確実なファイル編集手法
- **Function Callingの最適化**: ループ処理による連続ツール実行
- **ユーザー安全性**: 破壊的操作の事前確認の重要性

### 次のステップ

Chapter 5では、**システムプロンプト**を実装します。これにより、エージェントは以下のような高度な能力を獲得します。

- **体系的思考**: 探索→計画→実装の段階的プロセス
- **一貫した行動**: 明確な役割と制約に基づく判断
- **プロジェクト理解**: 既存コードの構造とパターンの把握

Chapter 4で基本的なツール機能を実装し、Chapter 5でより高度な思考プロセスを追加することで、nebulaは本格的なコーディングエージェントへと進化します。

それでは次のChapterに行きましょう！