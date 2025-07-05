# Chapter 5: システムプロンプト設計と安全性強化

## はじめに

この章では、nebulaエージェントにいよいよコーディングエージェントらしさを与えていきます。

この章を終える頃には、nebulaは複数ファイル編集能力を獲得し、複雑なタスクもこなせるようなエージェントとなります。
まずは現状はなぜ複雑なタスクができないか、から確認していきましょう。


## 現在のnebulaの問題点

Chapter 4までは下記のようなプロンプトを実行すると、うまく実装できなかったはずです。

```
tools/writeFile.goを参考に、tools/copyFile.goを作成してください。ファイルをコピーする機能を実装し、tools/registry.goに登録してください
```

これは現在のシステム（システムプロンプトなし）では、以下のような問題が発生するためです。

**1. 推測に基づく実装**
- 既存のコードを「知っているつもり」になりがち
- 実際のファイル内容を読まずに実装を開始
- 結果として矛盾したコードや動作しないコードを生成


**2. 断片的な作業**
- ファイル単体での編集に留まり、プロジェクト全体としての一貫性を欠く
- 一つのファイルを編集した後、他のファイルとの連携を考慮しない

これらの問題は、エージェントに**「どのように考え、どのように行動すべきか」**という指針がないことが原因です。
ですので、システムプロンプトを使い、行動指針を与えてあげましょう。

## システムプロンプトの実装

### なぜシステムプロンプトが効果的なのか

システムプロンプトは、LLMに「思考のフレームワーク」を与える仕組みです。

**システムプロンプトによる解決：**
- **一貫した思考プロセス**: 毎回同じ手順でタスクに取り組む
- **強制的な安全性**: 危険な操作を事前に防止
- **品質の向上**: 情報収集を強制することで実装品質が向上

### 設計した行動指針の詳細

さて、これから行動指針を与えるわけですが、下記のような指針を持たせたいと思います。

#### 1. 非交渉可能なルール（Critical Rules）

**狙い**: LLMの「推測したがる」性質を抑制し、必ず事実に基づく実装を強制する。

```text
# Critical Rules (Non-Negotiable)
1. **NEVER assume or guess file contents, names, or locations** - You must explore to understand them
2. **Information gathering is MANDATORY before implementation** - Guessing leads to immediate failure
3. **Before using writeFile or editFile, you MUST have used readFile on reference files**
4. **NEVER ask for permission between steps** - Proceed automatically through the entire workflow
5. **Complete the entire task in one continuous flow** - No pausing for confirmation
```

**重要な追加項目**:
- **ファイル名・場所の推測禁止**: GPT-4.1-nanoが起こしやすい「todo.ts」のような推測を防ぐ
- **自動実行の強制**: 情報収集後の「よろしいですか？」による停止を防ぐ

#### 2. 情報収集の重要性と自動実行プロトコル

**狙い**: 推測を防ぎ、現実の状況把握を強制し、実行の途切れを防ぐ。

```text
# Why Information Gathering is Critical
- **File structures vary**: What you expect vs. what exists are often different
- **Extensions matter**: .js vs .ts vs .go vs .py affects implementation
- **Directory layout matters**: Different projects have different organization
- **Assumption costs**: Guessing wrong means complete rework

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
```

**新しい重要な概念：**
- **なぜ情報収集が重要なのかを明示**: LLMの理解を深める
- **"Phase"から"Step"に変更**: より流れのある実行を暗示
- **自動実行の強調**: 各ステップ間で停止しない設計

#### 3. 具体的な禁止事項と失敗パターン

**狙い**: LLMが犯しやすい典型的なミスを具体例で防止する。

```text
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
```

**追加された重要な禁止事項：**
- **具体的な推測例**: GPT-4.1-nanoが起こしやすい具体的なミスを明示
- **許可確認の禁止**: 自動実行を阻害する行動の防止
- **失敗の理由説明**: なぜ推測が問題なのかをLLMに理解させる

### 実装済みシステムプロンプトの詳細

上記の設計思想を踏まえ、実際のシステムプロンプトは以下のようになっています。

```go
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
```

このシステムプロンプトは、`handleConversation`関数で会話の開始時に自動的に設定されます：

```go
// handleConversation はLLMとの対話セッションを処理する
func handleConversation(client *openai.Client, toolSchemas []openai.Tool, toolsMap map[string]tools.ToolDefinition, userInput string, messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	// システムプロンプトが設定されていない場合は最初に追加
	if len(messages) == 0 {
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: getSystemPrompt(),
		})
	}
	// 以下省略...
}
```

プロンプトについてはGemini CLIが参考になるので興味のある方は見てみてください。
:::message
**参考リンク**
- [Gemini CLI ソースコード](https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/core/prompts.ts)
:::

### プロンプト設計の改善過程と解決策、モデルによるプロンプト準拠の違い

当初のプロンプトは下記のような問題があり、試行錯誤の末現在のプロンプトに落ち着きました。
ここでは失敗パターンとどうすれば改善できたのかを見てきましょう。

**初期の失敗パターン:**
1. **情報収集で停止**: 「よろしいですか？」で実装に進まない問題
2. **推測による実装**: 「todo.ts」のようなファイル名を推測して実装しようとする
3. **不完全な作業**: handler層を抜かすなど、一部の必要なファイルを見落とし

**解決のアプローチ:**

#### 1. 自動実行の強制
```text
4. **NEVER ask for permission between steps** - Proceed automatically through the entire workflow
5. **Complete the entire task in one continuous flow** - No pausing for confirmation
```

#### 2. 推測禁止の具体化
```text
❌ **FORBIDDEN**: Guessing file names (e.g., assuming "todo.ts" exists without checking)
❌ **FORBIDDEN**: Guessing file extensions (e.g., assuming .js when it might be .ts)
❌ **FORBIDDEN**: Guessing directory structure (e.g., assuming files are in "src/" without checking)
```

#### 3. 多様な実例による学習
3つの異なるシナリオ（ファイル拡張子発見、参照ファイル読み込み、ディレクトリ構造発見）で正しい手順と間違った手順を明示。

#### モデルによる違い
上記までやったところGPT-4.1-miniでならプロンプト準拠で動き、複雑なタスクもこなせるようになりました。
現在までで使っているようなGPT-4.1-nanoについては、確かにシステムプロンプトがある方が成功の度合いはグンと高まります。
しかし、まだまだ自動で実装段階に行ってくれなかったり、作業を一つだけ抜かして進めてしまったりとプロンプト準拠しない場合もありました。

GPT-4.1-nanoでうまく進めたいのならユーザープロンプトを本当にがちがちに書かないと上手くタスク完了をしてくれない可能性があります。
ですので、できればGPT-4.1-miniを使用するのが良いです。

ただこのままではモデル切り替えもできませんし不便なので、モデル選択機能を作ってみましょう！

#### モデル選択機能の実装と使い方


##### 1. 設定管理システムの実装

まず`config/config.go`を新規作成し、設定管理システムを段階的に実装します。

**Step 1: 基本構造とデータ型の定義**
```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/sashabaranov/go-openai"
)

// Config represents the nebula configuration
type Config struct {
	Model  string `json:"model"`
	APIKey string `json:"-"` // APIキーは設定ファイルに保存しない
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Model: "gpt-4.1-nano", // デフォルトはgpt-4.1-nano
	}
}
```

**Step 2: 設定ファイルのパス管理**
```go
// getConfigPath returns the path to the configuration file
func getConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// フォールバック: カレントディレクトリに.nebulaフォルダを作成
		return ".nebula/config.json"
	}
	return filepath.Join(homeDir, ".nebula", "config.json")
}
```

**Step 3: 設定の保存機能**
```go
// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath := getConfigPath()
	
	// 設定ディレクトリを作成
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// JSONとして保存
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}
```

**Step 4: 設定の読み込み機能**
```go
// LoadConfig loads configuration from file or creates default
func LoadConfig() (*Config, error) {
	configPath := getConfigPath()
	
	// 設定ファイルが存在しない場合はデフォルト設定を作成
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		if err := SaveConfig(config); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
		return config, nil
	}
	
	// 設定ファイルを読み込み
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// APIキーは環境変数から取得
	config.APIKey = os.Getenv("OPENAI_API_KEY")
	
	return &config, nil
}
```

**Step 5: モデル選択とOpenAI API連携**
```go
// GetOpenAIModel returns the appropriate OpenAI model identifier
func (c *Config) GetOpenAIModel() string {
	switch c.Model {
	case "gpt-4.1-nano":
		return openai.GPT4Dot1Nano
	case "gpt-4.1-mini":
		return openai.GPT4Dot1Mini
	default:
		return openai.GPT4Dot1Nano // デフォルト
	}
}

// SetModel updates the model in configuration
func (c *Config) SetModel(model string) error {
	validModels := []string{"gpt-4.1-nano", "gpt-4.1-mini"}
	
	if slices.Contains(validModels, model) {
		c.Model = model
		return SaveConfig(c)
	}
	
	return fmt.Errorf("invalid model: %s. Valid models: %v", model, validModels)
}
```

- **設定ファイル場所**: `~/.nebula/config.json`（ホームディレクトリ）
- **APIキー管理**: セキュリティのため環境変数のみ（設定ファイルには保存しない）
- **デフォルトモデル**: `gpt-4.1-nano`（コスト効率重視）

##### 2. main.goへの統合

`main.go`に設定管理とモデル切り替え機能を統合します。

**変更1: インポートの追加**
```go
import (
	// 既存のインポート
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"nebula/config"  // ← 追加: 設定管理パッケージ
	"nebula/tools"

	"github.com/sashabaranov/go-openai"
)
```

**変更2: main関数での設定読み込み**
```go
func main() {
	// 旧: apiKey := os.Getenv("OPENAI_API_KEY")
	// 新: 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// APIキーチェック (設定構造体経由)
	if cfg.APIKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set")
		os.Exit(1)
	}

	// 旧: client := openai.NewClient(apiKey)
	// 新: 設定からAPIキーを使用
	client := openai.NewClient(cfg.APIKey)
}
```

**変更3: handleConversation関数の更新**
```go
// 関数シグネチャに cfg *config.Config を追加
func handleConversation(client *openai.Client, cfg *config.Config, toolSchemas []openai.Tool, toolsMap map[string]tools.ToolDefinition, userInput string, messages []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {

	// API呼び出し時にモデルを動的取得 (2箇所とも変更)
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    cfg.GetOpenAIModel(), // ← 設定から動的に取得
			Messages: messages,
			Tools:    toolSchemas,
		},
	)
}
```

**変更4: 起動メッセージとメインループ**
```go
// 起動メッセージにモデル情報を追加
fmt.Println("nebula - OpenAI Chat CLI with Function Calling")
fmt.Printf("Current model: %s\n", cfg.Model)  // ← 追加
fmt.Println("Type 'model' to switch between models")  // ← 追加

// メインループでの関数呼び出しにcfg引数を追加
messages = handleConversation(client, cfg, toolSchemas, toolsMap, userInput, messages)
```

##### 3. 対話的モデル切り替え機能

最後に、実行中にモデルを切り替える機能を追加します。

**変更5: メインループにモデル切り替え処理を追加**
```go
for {
	fmt.Print("You: ")
	if !scanner.Scan() {
		break
	}
	userInput := strings.TrimSpace(scanner.Text())

	// 終了コマンド
	if userInput == "exit" || userInput == "quit" {
		fmt.Println("Goodbye!")
		break
	}

	// モデル切り替えコマンド (新規追加)
	if userInput == "model" {
		handleModelSwitch(cfg)
		continue
	}

	if userInput == "" {
		continue
	}

	// 通常の対話処理
	messages = handleConversation(client, cfg, toolSchemas, toolsMap, userInput, messages)
}
```

**変更6: handleModelSwitch関数を追加 (main関数の前に配置)**
```go
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
		
		// 設定を更新して保存
		if err := cfg.SetModel(newModel); err != nil {
			fmt.Printf("Error setting model: %v\n", err)
		} else {
			fmt.Printf("Model switched to: %s\n", newModel)
		}
	}
}
```

これで対話的なモデル切り替え機能が完成です。実行中に`model`と入力するだけで、簡単にモデルを切り替えることができます。

##### 4. 実際の使用方法

```bash
# nebulaを起動
./nebula

nebula - OpenAI Chat CLI with Function Calling
Current model: gpt-4.1-nano                    # ← 現在のモデル表示
Available tools: readFile, list, searchInDirectory, writeFile, editFile
Type 'exit' or 'quit' to end the conversation
Type 'model' to switch between models          # ← 新機能
---

# モデル切り替え
You: model

Current model: gpt-4.1-nano
Available models:
1. gpt-4.1-nano (default, faster)
2. gpt-4.1-mini (complex tasks)
Select model (1 or 2): 2

Model switched to: gpt-4.1-mini                # ← 設定保存完了

# 以降は新しいモデルで動作
You: tools/writeFile.goを参考に、tools/copyFile.goを作成してください
```

##### 5. 設定ファイルの永続化

選択したモデルは`~/.nebula/config.json`に自動保存されます。

```json
{
  "model": "gpt-4.1-mini"
}
```

次回起動時も同じモデルが使用されるため、毎回設定する必要がありません。

##### 6. モデル選択の指針

| モデル | 特徴 | 使用場面 | コスト |
|--------|------|----------|--------|
| **gpt-4.1-nano** | 高速・軽量（デフォルト） | 単一ファイル編集、基本的な操作 | $0.10/$0.40 |
| **gpt-4.1-mini** | 複雑タスク対応 | 複数ファイル編集、アーキテクチャ理解 | $0.40/$1.60 |

**使い分けの例：**
- **nano**: 簡単なバグ修正、単一ファイルの読み取り
- **mini**: 複数ファイルを跨ぐ機能追加

:::message
**参考リンク**
- [OpenAI 使用料金確認](https://platform.openai.com/usage)
:::



## JSON処理の安全性強化
さてモデル選択が終わったところで、さっそくシステムプロンプト実験...と行きたいのですが、もう一点だけ修正事項があります。
この部分を修正しておかないと、システムプロンプトを存分に試すことができないのでここで修正しておきましょう。
具体的には下記のようなJSON関係の問題です。

**問題: editFile toolでの制御文字混入によるコンパイルエラー**
```
editやwriteでファイル内容に 0x06(ACK) のような制御文字が書き込まれることにより、コンパイルエラーが発生
```

### 問題の根本原因

**OpenAI Function CallingのJSON処理フロー：**
1. **LLMが文字列生成** → OpenAI APIがJSON形式でツールに送信
2. **JSON内で制御文字がエスケープ** → `\u0006`のような形式
3. **Go側でjson.Unmarshal** → エスケープされた制御文字が実際の制御文字に変換
4. **ファイルに書き込み** → `0x06`のような制御文字がファイルに混入

### 実装した解決策

#### 1. シンプルな制御文字除去機能

新しく`tools/json_helpers.go`を作成し、必要最小限の安全性機能を実装します。

```go
package tools

import (
	"strings"
)

// CleanControlCharacters は文字列から制御文字を除去する
func CleanControlCharacters(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1 // 制御文字を除去
		}
		return r
	}, s)
}
```

#### 2. ファイル操作での制御文字除去

writeFileとeditFileで、ファイル書き込み前に制御文字を自動除去：

```go
// writeFile での使用例
writeArgs.Content = CleanControlCharacters(writeArgs.Content)

// editFile での使用例  
editArgs.NewContent = CleanControlCharacters(editArgs.NewContent)
```

**重要な設計判断:**
- **readFile**: 制御文字除去なし（情報の完全性を保持）
- **writeFile/editFile**: 制御文字除去あり（安全なファイル作成）
- **JSON処理**: 標準ライブラリを信頼（Go標準の堅牢性活用）

### 修正の効果

**修正前：**
```json
// OpenAI APIからのJSON
{"path": "test.go", "content": "package main\u0006\nfunc main() {}"}
```
↓ json.Unmarshal
```go
// 制御文字が含まれるファイル内容 → コンパイルエラー
writeArgs.Content = "package main[ACK]\nfunc main() {}"
```

**修正後：**
```json
// 同じJSON入力
{"path": "test.go", "content": "package main\u0006\nfunc main() {}"}
```
↓ CleanControlCharacters で制御文字除去
```go
// クリーンなファイル内容 → コンパイル成功
writeArgs.Content = "package main\nfunc main() {}"
```

## ハンズオン：todo-appでの実践テスト

さて、ようやくシステムプロンプトを試すことができます。
本章の一番最初で示した例でも良いのですが、もう少し大きい例で試してみましょう。

### セットアップ手順

実際にnebulaの改善効果を体験するため、todo-appを使った実践テストを行います。

```bash
# 1. nebulaリポジトリをクローン
git clone <nebula-repo>
cd nebula

# 2. todo-appをコピー
cp -r test/todo-app ./todo-app
cd todo-app

# 3. git初期化（元の.gitディレクトリはない状態）
git init

# 4. 初期コミット（実験のベースライン）
git add .
git commit -m "Initial todo-app for nebula experiments"
```

### todo-appの構成

todo-appは、Clean Architectureに基づいたTODO管理APIです：

```
todo-app/
├── domain/
│   ├── todo.go          # Todo entity
│   └── repository.go    # Repository interface
├── usecase/
│   └── todo_usecase.go  # Business logic
├── handler/
│   └── todo_handler.go  # HTTP handlers
├── repository/
│   └── memory_repo.go   # In-memory implementation
├── main.go              # Entry point
├── go.mod               # Module definition
└── README.md            # Documentation
```

### 機能追加テスト

nebulaを起動し、以下のプロンプトを実行してみてください。

```
本プロジェクトに優先度機能を追加してください。具体的には次のように機能追加をお願いします。Todoエンティティに priority フィールド を追加し、domain層、usecase層、handler層すべてに適切な変更を行ってください。
```

todo.go、todo_usecase.go、todo_handler.goにそれぞれpriority関連の処理が追加されるはずです。
ようやく上手く情報を集め、既存のプロジェクトの構成に則り、複数ファイルを編集することができました！


注意点:
たまに探索だけで実装まで行ってくれないときがあるのですが、その時は一回exitして最初からやり直してみてください。
もし上手くいかないようならもうちょっとプロンプトを詳しく書き、下記のようにしてみてください。

```
Goで書かれている本プロジェクトのTODOアプリに優先度機能を追加してください。具体的には次のように機能追加をお願いします。Todoエンティティに priority フィールド を追加し、domain層のtodo.go、usecase層のtodo_usecase.go、handler層のtodo_handler.go すべてに適切な変更を行ってください。
```

また、もしGPT-4.1-nanoを使っていて上手く行かないときはGPT-4.1-miniを使うことも検討してみてください。



### 実験後のリセット

各実験後は、以下のコマンドで元の状態に戻せます：

```bash
# 前回の実験をリセット
git reset --hard HEAD~1
git clean -fd
```

## この章のまとめと次のステップ

### 達成したこと

この章では、nebulaにシステムプロンプトを与えることで、複数ファイル編集機能を元プロジェクトの記法に合わせて達成することができました。

**実装した機能：**
- **改良されたシステムプロンプト**: 自動実行と推測禁止を両立する実行プロトコル
- **段階的思考プロセス**: Step 1（情報収集）→ Step 2（実装）の流れのある分離
- **制御文字安全性**: シンプルな制御文字除去による安全なファイル操作
- **最適化されたJSON処理**: 標準ライブラリを活用したシンプルで効果的な処理
- **複数ファイル編集能力**: プロジェクト全体を意識した連携編集

**学んだ重要概念：**
- **プロンプトの進化**: 初期の問題（停止・推測・不完全性）から改善された自動実行
- **モデル差への対応**: 軽量モデル（nano）と上位モデル（mini以上）の特性理解
- **シンプル設計の価値**: 複雑さを避けた最小限の安全性確保
- **普遍的原則**: 特定モデル向けではない汎用的な改善アプローチ

### 現在のファイル構成

```
nebula/
├── main.go                    # 改良されたシステムプロンプト & モデル選択機能実装済み
├── config/                    # ★ NEW: 設定管理パッケージ
│   └── config.go              # モデル選択と設定ファイル管理
├── tools/                    
│   ├── common.go
│   ├── registry.go
│   ├── json_helpers.go        # ★ NEW: シンプルな制御文字除去機能
│   ├── readfile.go           ）
│   ├── list.go                
│   ├── search.go            
│   ├── writefile.go          
│   └── editfile.go           
└── go.mod
...
```

### 次章でやること

Chapter5でnebulaは複雑なファイル編集機能を獲得しました。
これでコーディングエージェントらしいと言える姿になったのではないでしょうか。
余談ですが、本章の複数ファイル生成の部分でGPT-4.1-nanoを使っていたら、ファイル生成がなかなか上手くいかず、
何度も一週間くらいプロンプトを練り直して、プロンプトの難しさとモデルの性能差を思い知りました...

次のChapter6ではもう少しだけ機能追加をしていきます。
複数ファイル編集は本Chapterで達成できているのですが、Planモードと以前の会話からスタートする記憶保持機能を実装していきます。
この2能を実装し、本プロジェクトを締めくくりましょう！