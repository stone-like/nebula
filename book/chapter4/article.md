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


## ハンズオン・チュートリアル

### Step 1: `editFile`ツールの実装

まずは、既存ファイルを編集するための`editFile`ツールを実装します。これまでのツールと同じパターンに従って、新しいファイルを作成しましょう。

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

次に、このツールをシステムに登録しましょう。`tools/registry.go`を編集します：

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

そして、`main.go`の利用可能ツールの表示も更新します：

```go
fmt.Println("Available tools: readFile, list, searchInDirectory, writeFile, editFile")
```

### Step 2:  なぜ部分編集ではなく、ファイル全体の上書きを選んだのか

ここで、なぜ「ファイル全体の上書き」という一見非効率的な方法を選んだのかを説明します。

#### 部分編集ツールとは

部分編集ツールとは、以下のようなものです：

```go
// 行番号指定編集
func EditLines(filePath string, startLine, endLine int, newContent string)

// 正規表現置換
func ReplacePattern(filePath string, pattern, replacement string)

// 差分適用
func ApplyPatch(filePath string, patch string)
```

#### なぜ学習用プロジェクトでは悪手なのか

これはエラー処理の複雑化による理解のし難さが最大の原因です。

部分編集では、以下のようなエラーが頻発します。

- 構文エラーを引き起こす不完全な編集
- インデントの不整合
- インポート文の重複

これはプロンプトを念入りに作成したり、編集に失敗した際にコード上で修正対応を行う等で回避が可能ですが、理解が難しいです。
例えばGeminiCLIでは編集行の前後数行のコンテキストを要求する手法を採用していますが、なかなか理解が難しいので今回は見送り、
パフォーマンスやコストは悪いものの、理解しやすい方法としました。

:::message
**参考リンク**
- [Gemini CLI ソースコード](https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/tools/edit.ts)
:::


### Step 3: Tool Callingループの実装

実装したeditFileツールをテストする前に、重要な問題を解決する必要があります。それは**「readFileの後にeditFileが呼ばれない」**という問題です。

現在のmain.goでは、ツール実行後に1回だけ追加のAPI呼び出しを行っていますが、重要な問題があります。**2回目のAPI呼び出しで返されたツールコールを実行していない**のです。

実際の問題：

```
1. ユーザー入力 → LLM判断 → readFile実行
2. readFile結果 → LLM判断 → 「editFileを実行せよ」と応答
3. ツールコールを検出するが実行せずに終了 ←問題！
```

LLMは正しく「editFileを実行せよ」と指示していますが、それを**実行せずに終了**してしまっています。

この連続したツール呼び出しを可能にするため、main.goを改修しましょう。

#### ツール実行の関数分離

まず、ツール実行の関数を分離します：

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

### Step 4: Read-Modify-Writeパターンの動作確認

 Tool Callingループ実装が完了したら、実際にRead-Modify-Writeパターンの動作を確認してみましょう。

まず、テスト用のファイルを作成します。

```bash
echo "Hello World" > sample.txt
```

nebulaを起動して、以下の指示を与えてみてください：

```bash
go build -o nebula .
./nebula
```

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

### Step 5: ツールスキーマDescriptionの重要性

`editFile`ツールの`Description`をもう一度見てください。

```go
Description: "既存ファイルの内容を完全に上書きします。重要: ファイルを破壊しないために、必ず以下のワークフローに従ってください: 1. 'readFile'を使用して現在の完全な内容を取得する。2. 思考プロセスで、読み取った内容を基に新しいファイルの完全版を構築する。3. このツールを使用して完全な新しい内容を書き込む。部分的な編集には使用しないでください。常にファイル全体の内容を提供してください。実行前にユーザーの許可を求めます。"
```

これは単なる説明文ではありません。**LLMの行動を制御する重要な指示**となっています。

#### Descriptionが果たす役割

**1. 行動パターンの強制**
「必ず以下のワークフローに従ってください」という文言により、LLMにRead-Modify-Writeパターンを強制しています。

**2. 危険な使用方法の防止**
「部分的な編集には使用しないでください」により、誤った使用を防いでいます。

**3. 前提条件の明確化**
「readFileを使用して現在の完全な内容を取得する」により、必要な前処理を指示しています。

#### 実験：Descriptionを簡素にしてみる

試しに、`editFile`のDescriptionを簡潔にしてみてください。

```go
Description: "ファイルを編集します。"
```

パラメータのDescriptionも簡素にします。

```go
"new_content": {
    Type:        jsonschema.String,
    Description: "新しい内容",
}
```

この状態で、sample.txtを下記のようにしてみてください。

```
これはテスト用のファイルです。
nebulaのreadFileツールがこのファイルを正しく読み込めるかをテストしています。

ファイルの内容:
- 行1: Hello World
- 行2: こんにちは世界
- 行3: Function Calling is working! 
```


ここまでできたら以下のテストを実行してみてください。

```
sample.txtに「Hello」という文字列を追加してください
```

**簡素なDescriptionでの実際の動作例**：

```
Assistant is using tools...
Executing tool: writeFile with arguments: {"content":"Hello","path":"sample.txt"}
Tool 'writeFile' executed with result: {"success":false,"error":"ファイルが既に存在します。既存ファイルの編集にはeditFileを使用してください。"}

Assistant is using tools...
Executing tool: editFile with arguments: {"new_content":"Hello","path":"sample.txt"}

既存ファイルを編集します: sample.txt
実行してもよろしいですか？ (y/N): y

Tool 'editFile' executed with result: {"success":true}
```

**問題点**：
1. **Read-Modify-Writeの省略**: `readFile`を呼ばずに直接`editFile`
2. **既存内容の消失**: 「追加」のつもりが完全上書きに

**詳細なDescriptionでの動作**：

```
Assistant is using tools...
Executing tool: readFile with arguments: {"path":"sample.txt"}
Tool 'readFile' executed with result: {"content":"元の内容\n"}

Assistant is using tools...
Executing tool: editFile with arguments: {"path":"sample.txt","new_content":"元の内容\nHello\n"}
```

**改善点**：
- 最初から正しいツールを選択
- 既存内容を保持した追加
- 安全なワークフロー

簡素なDescriptionでもLLMの賢さのために上手くいく時はあるものの、やはり詳細なDescriptionとは成功率が違うので、なるべく詳細に書くことをお勧めします。
(詳細に書きすぎても指示忘れが起こってしまう等があるのが難しいところですが)


### Step 6: 今のエージェントの限界

現在のエージェントは素晴らしい能力を持っていますが、まだ限界があります。以下のような複雑なタスクを試してみてください。

#### テスト1: 複数ファイル編集

```
tools/writeFile.goを参考に、tools/copyFile.goを作成してください。ファイルをコピーする機能を実装し、tools/registry.goに登録してください
```

このテストを実行すると、以下のような問題が発生するはずです。

1. **思考プロセスの欠如**: 参考ファイルを読まずにいきなり実装を始める
2. **プロジェクト構造の誤解**: 間違ったパッケージ宣言や不適切なインポート
3. **一貫性の欠如**: 既存のコードスタイルと異なる実装

#### テスト2: 複雑な機能追加

```
このプロジェクトにログ機能を追加してください。適切な場所にログファイルを作成し、main.goとツール群でログを出力するようにしてください
```

このような曖昧な指示では、現在のエージェントは以下の理由で困惑します：

1. **「適切な場所」がわからない**: プロジェクト構造の理解が不足
2. **既存コードとの統合方法がわからない**: アーキテクチャの理解が不足
3. **一貫した実装ができない**: 統一された思考プロセスがない

#### なぜこれらの限界が存在するのか

現在のnebulaには、以下が欠けています：

**1. システムプロンプト（思考プロセス）**
- どのような順序で考え、行動すべきかの指針
- 探索 → 計画 → 実装 の段階的プロセス
- エラーハンドリングや一貫性の確保

**2. プロジェクトコンテキスト**
- このプロジェクトがどのような構造を持っているか
- どのようなアーキテクチャを採用しているか
- どこに何を配置すべきかの知識

次章からは上記のような複雑なタスクもできるようにしていきます。

**Chapter 5**では、nebulaに「思考プロセス」を与えます。システムプロンプトにより、エージェントは以下のような能力を獲得します。

- 体系的な探索と分析
- 段階的な計画立案
- 一貫した実装パターン

**Chapter 6**では、さらに「プロジェクト理解能力」を追加し、曖昧な指示からでも適切な機能追加ができるようになります。

現在のnebulaでも十分実用的ですが、これらの限界体験により「なぜ次の機能が必要なのか」が明確になったのではないでしょうか。


## この章のまとめと次のステップ

### 達成したこと

この章では、エージェントに重要な新機能を追加しました。

**実装した機能**：
- **`editFile`ツール**: 既存ファイルの安全な編集
- **ユーザー許可システム**: 危険な操作の事前確認
- **Tool Callingループ**: 複数ツールの連続実行
- **Read-Modify-Write強制**: ツールスキーマでの行動制御

**学んだ重要概念**：
- **Read-Modify-Writeパターン**: 安全で確実なファイル編集手法
- **ツールスキーマの重要性**: LLMの行動制御における詳細な説明文の価値
- **状態管理の複雑さ**: 部分編集ツールが学習用途で悪手である理由
- **Function Callingの最適化**: ループ処理による連続ツール実行

### 現在のファイル構成

```
nebula/
├── main.go                  # リファクタリング済みのメインプログラム
├── tools/
│   ├── common.go           # 共通型定義
│   ├── registry.go         # ツール登録（editFile追加済み）
│   ├── readfile.go         # ファイル読み取り
│   ├── list.go             # ディレクトリ一覧
│   ├── search.go           # キーワード検索
│   ├── writefile.go        # 新規ファイル作成
│   └── editfile.go         # ★ ファイル編集（新規追加）
├── go.mod
├── go.sum
└── sample.txt              # テスト用ファイル
```

### 次のステップ

Chapter5では、システムプロンプトを記載していきます。システムプロンプトにより、エージェントは以下のような能力を持ちます。

- **体系的思考**: 探索 → 計画 → 実装の段階的プロセス
- **一貫した行動**: 明確な役割と制約に基づく判断

現在はまだまだ公にあるコーディングエージェントとは遠く感じるしれませんが、次回からどんどんコーディングエージェントらしさを増していきます。
それでは次のChapterに行きましょう！