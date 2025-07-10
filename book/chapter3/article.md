# Chapter 3: ファイルシステムを探索・作成するツール群

## はじめに

前章では、GPTが最初のツール「`readFile`」を使えるようになりました。
ですが、コード生成のためにはさらなるツール類が必要です。

この章では、エージェントがファイルシステムを自由に探索し、新しいファイルを作成できるようにするために、
listツール、searchツール、writeツールという3つの重要なツールを実装します。


これらのツールが揃うことで、LLMがプロジェクトの構造を理解し、必要なファイルを見つけ、新しいコードを書き出す能力を得ます。

## 📁 この章での到達目標構造

```
nebula/
├── main.go                 # 複数ツール対応
├── tools.go               # 4つのツール実装 (~400行)
├── go.mod                 
└── go.sum                 
```

**前章からの変化:**
- Chapter 2: readFileツールのみ
- Chapter 3: **3つの新ツール追加** ← 今ここ

**実装する機能:**
- `list`ツール: ディレクトリ探索
- `searchInDirectory`ツール: キーワード検索  
- `writeFile`ツール: 新規ファイル作成（ユーザー確認付き）


**tools.go拡張内容:**
- ListArgs/ListResult構造体
- SearchInDirectoryArgs/SearchInDirectoryResult構造体
- WriteFileArgs/WriteFileResult構造体
- GetAvailableTools()関数拡張


それではさっそくツール類を作成していきましょう！

## ファイルシステムツール実装

### 1. `list`ツールの実装（ファイル一覧の取得）

まずは、エージェントが周囲の状況を把握するための`list`ツールを実装しましょう。
このツールは、指定されたディレクトリ内のファイルとフォルダの一覧を取得します。

現在の`tools.go`に新しいツールを追加していきます。

```go
// ListArgs はlistツールの引数を表す構造体
type ListArgs struct {
	Path      string `json:"path" description:"リストするディレクトリのパス"`
	Recursive bool   `json:"recursive" description:"再帰的にリストするかどうか"`
}

// ListResult はlistツールの結果を表す構造体
type ListResult struct {
	Files []string `json:"files"`
	Error string   `json:"error,omitempty"`
}
```

次に、実際のリスト処理を行う関数を実装します。この関数は2つのモード（再帰的・非再帰的）を持ち、それぞれ異なるアプローチでファイル一覧を取得します。

```go
// List は指定されたパス内のファイルとディレクトリをリストする
func List(args string) (string, error) {
	// 1. JSON引数をGoの構造体に変換
	var listArgs ListArgs
	if err := json.Unmarshal([]byte(args), &listArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	var files []string

	if listArgs.Recursive {
		// 2. 再帰的な探索: ディレクトリ階層を深く辿る
		// filepath.Walkは指定されたパス以下のすべてのファイル・ディレクトリを訪問する
		err := filepath.Walk(listArgs.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err // エラーが発生した場合は探索を中断
			}
			// 見つかったパスをすべて配列に追加（ファイルもディレクトリも含む）
			files = append(files, path)
			return nil
		})
		if err != nil {
			// エラーが発生した場合でもJSON形式で結果を返す
			result := ListResult{
				Files: []string{},
				Error: fmt.Sprintf("ディレクトリの読み込みに失敗しました: %v", err),
			}
			resultJSON, _ := json.Marshal(result)
			return string(resultJSON), nil
		}
	} else {
		// 3. 非再帰的な探索: 指定されたディレクトリの直下のみ
		// os.ReadDirは1階層のみを読み込む軽量な方法
		entries, err := os.ReadDir(listArgs.Path)
		if err != nil {
			result := ListResult{
				Files: []string{},
				Error: fmt.Sprintf("ディレクトリの読み込みに失敗しました: %v", err),
			}
			resultJSON, _ := json.Marshal(result)
			return string(resultJSON), nil
		}

		// 各エントリのフルパスを構築して配列に追加
		for _, entry := range entries {
			files = append(files, filepath.Join(listArgs.Path, entry.Name()))
		}
	}

	// 4. 成功時の結果をJSON形式で返却
	result := ListResult{
		Files: files,
		Error: "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
```

**実装のポイント解説:**

- **`filepath.Walk`**: 再帰的探索の標準的な方法。コールバック関数を使って各ファイル・ディレクトリを処理
- **`os.ReadDir`**: 軽量な非再帰探索。
- **エラーハンドリング**: 探索エラーでもプログラムがクラッシュしないよう、JSON形式でエラー情報を返却
- **パス結合**: `filepath.Join`でOS依存のパス区切り文字を適切に処理

そして、OpenAI Function Callingのためのスキーマ定義を追加します。

```go
// GetListTool はlistツールの定義を返す
func GetListTool() ToolDefinition {
	return ToolDefinition{
		Schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list",
				Description: "指定したディレクトリ内のファイルとディレクトリの一覧を返します。recursiveがtrueの場合、再帰的にリストします。",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "リストするディレクトリのパス",
						},
						"recursive": {
							Type:        jsonschema.Boolean,
							Description: "再帰的にリストするかどうか（デフォルト: false）",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		Function: List,
	}
}
```

### 2. `searchInDirectory`ツールの実装（キーワード検索）

次に検索機能を実装しましょう。このツールは、プロジェクト全体から特定のキーワードを含むファイルを見つけ出します。

必要なimportを`tools.go`の先頭に追加します。

```go
import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)
```

検索ツール用の構造体を定義します。

```go
// SearchInDirectoryArgs はsearchInDirectoryツールの引数を表す構造体
type SearchInDirectoryArgs struct {
	Directory string `json:"directory" description:"検索するディレクトリのパス"`
	Keyword   string `json:"keyword" description:"検索するキーワード"`
}

// SearchInDirectoryResult はsearchInDirectoryツールの結果を表す構造体
type SearchInDirectoryResult struct {
	Files []string `json:"files"`
	Error string   `json:"error,omitempty"`
}
```

検索機能の実装です。この関数はプロジェクト全体を走査して、指定されたキーワードを含むファイルを効率的に見つけ出します。

```go
// SearchInDirectory は指定されたディレクトリ配下を再帰的に検索し、キーワードを含むファイルを見つける
func SearchInDirectory(args string) (string, error) {
	// 1. JSON引数をGoの構造体に変換
	var searchArgs SearchInDirectoryArgs
	if err := json.Unmarshal([]byte(args), &searchArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	var matchingFiles []string

	// 2. 指定されたディレクトリ以下のすべてのファイルを走査
	err := filepath.Walk(searchArgs.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // ファイルシステムエラーが発生した場合は処理を中断
		}

		// 3. ディレクトリは検索対象外なのでスキップ
		if info.IsDir() {
			return nil
		}

		// 4. ファイルを開いて内容を読み込み
		file, err := os.Open(path)
		if err != nil {
			// バイナリファイルや権限なしファイルは静かにスキップ
			// エラーを返すと全体の検索が止まってしまうため
			return nil
		}
		defer file.Close()

		// 5. ファイルを1行ずつ読み込んでキーワードを検索
		// bufio.Scannerを使うことで大きなファイルでもメモリ効率良く処理
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// 6. 現在の行にキーワードが含まれているかチェック
			if strings.Contains(scanner.Text(), searchArgs.Keyword) {
				// 7. マッチしたファイルのパスを結果リストに追加
				matchingFiles = append(matchingFiles, path)
				break // 1つのファイルで複数行マッチしても1回だけ記録
			}
		}

		return nil // 次のファイルの処理に進む
	})

	// 8. 検索処理でエラーが発生した場合の処理
	if err != nil {
		result := SearchInDirectoryResult{
			Files: []string{},
			Error: fmt.Sprintf("検索に失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 9. 検索成功時の結果をJSON形式で返却
	result := SearchInDirectoryResult{
		Files: matchingFiles,
		Error: "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
```

**実装のポイント解説:**

- **`bufio.Scanner`**: 大きなファイルでもメモリ効率良く1行ずつ処理。
- **`strings.Contains`**: シンプルな文字列検索。
- **エラーハンドリング**: バイナリファイルや権限エラーでも検索を継続。堅牢性を重視
- **早期終了**: ファイル内で1回でもマッチしたら`break`で次のファイルへ。無駄な処理を避ける

スキーマ定義も追加。

```go
// GetSearchInDirectoryTool はsearchInDirectoryツールの定義を返す
func GetSearchInDirectoryTool() ToolDefinition {
	return ToolDefinition{
		Schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "searchInDirectory",
				Description: "指定されたディレクトリ配下を再帰的に検索し、キーワードを含むファイルのパスのリストを返します。",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"directory": {
							Type:        jsonschema.String,
							Description: "検索を開始するディレクトリのパス",
						},
						"keyword": {
							Type:        jsonschema.String,
							Description: "ファイル内で検索するキーワード",
						},
					},
					Required: []string{"directory", "keyword"},
				},
			},
		},
		Function: SearchInDirectory,
	}
}
```

### 3. `writeFile`ツールの実装（新規ファイル作成・ユーザー許可付き）

:::message alert
⚠️ **重要：破壊的操作について**
`writeFile`ツールはファイルシステムに変更を加える破壊的操作です。実行前に必ずユーザーの明示的な許可を得る設計にしています。これにより、LLMによる予期しないファイル作成を防ぎ、安全な開発環境を維持できます。
:::

最後に`writeFile`ツールを実装します。
このツールは新しいファイルを作成する強力な機能ですが、同時に危険性も持っています。そのため、実行前に必ずユーザーの許可を得るようにします。

構造体の定義。

```go
// WriteFileArgs はwriteFileツールの引数を表す構造体
type WriteFileArgs struct {
	Path    string `json:"path" description:"作成するファイルのパス"`
	Content string `json:"content" description:"ファイルに書き込む内容"`
}

// WriteFileResult はwriteFileツールの結果を表す構造体
type WriteFileResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
```

ユーザー許可機能付きの実装です。この関数はLLMによる自動ファイル作成の危険性を考慮し、必ずユーザーの明示的な承認を得てから実行される安全な設計になっています。

```go
// WriteFile は指定されたパスに新しいファイルを作成する（ユーザー許可が必要）
func WriteFile(args string) (string, error) {
	// 1. JSON引数をGoの構造体に変換
	var writeArgs WriteFileArgs
	if err := json.Unmarshal([]byte(args), &writeArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	// 2. 安全性チェック: 既存ファイルの上書きを防止
	// os.Statでファイルの存在を確認。存在する場合はエラーで終了
	if _, err := os.Stat(writeArgs.Path); err == nil {
		result := WriteFileResult{
			Success: false,
			Error:   "ファイルが既に存在します。既存ファイルの編集にはeditFileを使用してください。",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 3. ユーザー許可の取得: 重要なセキュリティ機能
	fmt.Printf("\n新しいファイルを作成します: %s\n", writeArgs.Path)
	fmt.Print("実行してもよろしいですか？ (y/N): ")

	// 4. 標準入力からユーザーの応答を読み取り
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		result := WriteFileResult{
			Success: false,
			Error:   "ユーザー入力の読み取りに失敗しました",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 5. ユーザー応答の検証: 'y'または'Y'以外はすべてキャンセル扱い
	userResponse := strings.TrimSpace(scanner.Text())
	if userResponse != "y" && userResponse != "Y" {
		result := WriteFileResult{
			Success: false,
			Error:   "ユーザーによってキャンセルされました",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 6. 親ディレクトリの自動作成
	// filepath.Dirでディレクトリ部分を抽出、os.MkdirAllで階層ごと作成
	dir := filepath.Dir(writeArgs.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result := WriteFileResult{
			Success: false,
			Error:   fmt.Sprintf("ディレクトリの作成に失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 7. ファイルの作成
	// os.Createで新規ファイルを作成（既存ファイルは上書き）
	file, err := os.Create(writeArgs.Path)
	if err != nil {
		result := WriteFileResult{
			Success: false,
			Error:   fmt.Sprintf("ファイルの作成に失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}
	defer file.Close() // 確実にファイルハンドルを閉じる

	// 8. 内容の書き込み
	if _, err := file.WriteString(writeArgs.Content); err != nil {
		result := WriteFileResult{
			Success: false,
			Error:   fmt.Sprintf("ファイルの書き込みに失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 9. 成功時の結果を返却
	result := WriteFileResult{
		Success: true,
		Error:   "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}
```

**実装のポイント解説:**

- **既存ファイル保護**: `os.Stat`で事前チェック。意図しない上書きを防ぐ。
- **ユーザー確認**: 破壊的変更なのでユーザーの承認を得る形に。
- **ディレクトリ自動作成**: `os.MkdirAll`で深い階層のディレクトリも一度に作成可能
- **リソース管理**: `defer file.Close()`でファイルハンドルの確実な解放
- **エラー境界**: 各ステップでエラーをキャッチし、適切なJSON形式で返却。

スキーマ定義。

```go
// GetWriteFileTool はwriteFileツールの定義を返す
func GetWriteFileTool() ToolDefinition {
	return ToolDefinition{
		Schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "writeFile",
				Description: "指定されたパスに新しいファイルを作成し、内容を書き込みます。親ディレクトリが存在しない場合は自動で作成します。既存ファイルが存在する場合は失敗します。実行前にユーザーの許可を求めます。",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "作成するファイルの完全なパス",
						},
						"content": {
							Type:        jsonschema.String,
							Description: "ファイルに書き込む内容",
						},
					},
					Required: []string{"path", "content"},
				},
			},
		},
		Function: WriteFile,
	}
}
```

### 4. 複数ツール管理システムの構築

新しいツールを追加したので、`GetAvailableTools`関数を更新して、すべてのツールを管理できるようにします。

```go
// GetAvailableTools は利用可能な全てのツールを返す
func GetAvailableTools() map[string]ToolDefinition {
	return map[string]ToolDefinition{
		"readFile":           GetReadFileTool(),
		"list":               GetListTool(),
		"searchInDirectory":  GetSearchInDirectoryTool(),
		"writeFile":          GetWriteFileTool(),
	}
}
```

また、`main.go`の表示メッセージも更新しましょう。

```go
fmt.Println("nebula - OpenAI Chat CLI with Function Calling")
fmt.Println("Available tools: readFile, list, searchInDirectory, writeFile")
fmt.Println("Type 'exit' or 'quit' to end the conversation")
fmt.Println("---")
```

### 5. 現在の状況確認

この章で追加した3つのツール（`list`、`searchInDirectory`、`writeFile`）により、`tools.go`ファイルが相当大きくなってしまいました。
ですので次章の最初にリファクタリングをして、ツールごとに一ファイルとしましょう。

**現在のファイル構成:**
```
nebula/
├── main.go                 # メインプログラム
├── tools.go               # 全ツール実装（約400行）
├── go.mod
└── go.sum
```

## この章での学習ポイント

**達成できたこと:**
1. **単一ファイル内でのツール管理**: `tools.go`内での複数ツール実装パターン習得
2. **ツール設計原則**: 各ツールの構造体定義、実装、スキーマ定義の統一パターン
3. **エラーハンドリング**: ファイルシステム操作での堅牢なエラー処理
4. **ユーザー安全性**: `writeFile`でのユーザー確認機能

**次章での移行予定:**
Chapter 4では、以下の理由でモジュラー構造に移行します。
- `editFile`ツール追加により、単一ファイルが管理困難になる
- 各ツールの独立性向上（テスト・保守性）
- より実践的なGoプロジェクト構造の学習

## 動作確認

これでこの章の機能が動くようになりました！実際に試してみましょう。

まず、プログラムをビルドします。

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

### テストケース

以下のコマンドを試して、各ツールが正常に動作することを確認してください。

#### 1. listツールのテスト

```
現在のディレクトリのファイル一覧を表示してください
```

期待される動作：ディレクトリ内のファイルとフォルダがJSON形式でリストされます。

```
toolsディレクトリを再帰的にリストしてください
```

期待される動作：toolsディレクトリ内のすべてのファイルが階層的に表示されます。

#### 2. searchInDirectoryツールのテスト

```
プロジェクト内で "OpenAI" という文字列を含むファイルを探してください
```

期待される動作：OpenAIという文字列を含むファイルのパス一覧が表示されます。

```
"GetAvailableTools" という関数が定義されているファイルを見つけてください
```

期待される動作：該当する関数が定義されているファイルが見つかります。

#### 3. writeFileツールのテスト

```
"hello.txt" というファイルを作成して、内容は "Hello, World!" にしてください
```

期待される動作。
1. ユーザー確認プロンプトが表示される
2. 'y'を入力すると、ファイルが作成される
3. 他の文字を入力すると、キャンセルされる

```
"test/example.go" というファイルを作成して、シンプルなHello Worldプログラムを書いてください
```

期待される動作：`test`ディレクトリが自動作成され、Goプログラムが生成されます。

## この章のまとめと次のステップ

この章では、エージェントに3つの重要な能力を与えることができました。

### 達成できたこと

✅ **ディレクトリ探索機能** - `list`ツールでプロジェクト構造を把握  
✅ **キーワード検索機能** - `searchInDirectory`ツールで必要な情報を発見  
✅ **ファイル作成機能** - `writeFile`ツールで新しいコードを生成  
✅ **ユーザー許可システム** - 危険な操作前の安全確認  
✅ **コード整理** - ツールごとの分割によるスケーラブルなアーキテクチャ


この時点で、nebulaエージェントは以下の能力を持っています。

-  `readFile`でファイル内容を読み取り
-  `list`でプロジェクト構造を把握
-  `searchInDirectory`で必要な情報を検索
-  `writeFile`で新しいファイルを作成

しかし、まだ「既存ファイルの編集」という重要な能力が不足しています。

次のChapter 4では、最も重要で複雑な機能である`editFile`ツールを実装します。
このツールは、既存のファイルを安全かつ確実に編集するためのもので、コード生成の要となる機能です。

edit機能まで揃えることにより、コーディングエージェントらしさがぐっと上がってきます！
