# Chapter 1: OpenAIと会話する最初のCLI

## はじめに

この章では、GoからOpenAI APIを呼び出して、ターミナルで対話できる最小のプログラムを作成します。

まずはシンプルなチャット機能から始めることで、LLMと会話する感覚を掴む感覚を得ていきましょう。

この章が終わる頃には。GPTと自由に会話できるプログラムが完成し、次章で実装するFunction Calling（ツール機能）の土台が整います。

## 📁 この章での到達目標構造

```
nebula/
├── main.go                 # 基本的なCLI + OpenAI API呼び出し
├── go.mod                  # モジュール定義
└── go.sum                  # 依存関係ロック
```

**実装する機能:**
- 基本的なOpenAI API呼び出し
- 簡単なCLI実装
- 会話ループ

**依存関係:**
- `github.com/sashabaranov/go-openai`


## 基本CLI実装

### 1. Goプロジェクトのセットアップ

まずは、新しいGoプロジェクトを作成します。作業用のディレクトリを作成し、Goモジュールを初期化しましょう。

```bash
mkdir nebula
cd nebula
go mod init nebula
```

これで`go.mod`ファイルが生成され、Goのモジュール管理が有効になります。`nebula`という名前は、私たちが作る自律コーディングエージェントの名前です。

```go
// go.mod
module nebula

go 1.23.1
```

### 2. OpenAI APIクライアントライブラリの導入

次に、OpenAI APIと通信するためのライブラリを追加します。GoでOpenAI APIを使う場合、`github.com/sashabaranov/go-openai`が最も人気で安定したライブラリです。

```bash
go get github.com/sashabaranov/go-openai
```

実行すると、`go.mod`に依存関係が追加され、`go.sum`ファイルも生成されます。

```go
// go.mod (更新後)
module nebula

go 1.23.1

require github.com/sashabaranov/go-openai v1.40.3 // indirect
```

### 3. APIキーを安全に環境変数から読み込む

OpenAI APIを使うには、APIキーが必要です。セキュリティの観点から、APIキーはコードに直接書かず、環境変数から読み込むのがベストプラクティスです。

まず、OpenAIのWebサイトでAPIキーを取得し、環境変数にセットしましょう。

```bash
export OPENAI_API_KEY=your_api_key_here
```

:::message
**APIキーの取得方法**
1. [OpenAI Platform](https://platform.openai.com/)にログイン
2. 右上の歯車マークをクリック
3. 左サイドバーの「API keys」をクリック
4. 「Create new secret key」でAPIキーを生成
5. 生成されたキーをコピーして環境変数にセット
:::

:::message alert
⚠️ **重要：OpenAI API使用料金について**
2025年1月現在、OpenAI APIを使用するためには最初に5ドルの支払いが必要です。
本書で使用するGPT-4.1-nano、GPT-4.1-miniは1000回呼び出しても約0.5~0.6ドル程度なので、5ドルを超える心配はほとんどありません。
Auto rechargeをOFFにしておけば追加の支払いも発生しません。
:::


### 4. メインプログラムの実装

それでは、`main.go`ファイルを作成し、OpenAI APIと対話するプログラムを実装していきます。

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

	fmt.Println("nebula - OpenAI Chat CLI")
	fmt.Println("Type 'exit' or 'quit' to end the conversation")
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)

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

		// OpenAI APIに送信
		resp, err := client.CreateChatCompletion(
			context.Background(),
			openai.ChatCompletionRequest{
				Model: openai.GPT4Dot1Nano,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: userInput,
					},
				},
			},
		)

		if err != nil {
			fmt.Printf("Error calling OpenAI API: %v\n", err)
			continue
		}

		if len(resp.Choices) > 0 {
			fmt.Printf("Assistant: %s\n\n", resp.Choices[0].Message.Content)
		} else {
			fmt.Println("No response received from OpenAI")
		}
	}
}
```

### コードの解説

このプログラムの各部分を詳しく見ていきましょう。

**1. APIキーの安全な読み込み**
```go
apiKey := os.Getenv("OPENAI_API_KEY")
if apiKey == "" {
    fmt.Println("Error: OPENAI_API_KEY environment variable is not set")
    os.Exit(1)
}
```
環境変数からAPIキーを取得し、設定されていない場合はエラーメッセージと共にプログラムを終了します。

**2. OpenAIクライアントの初期化**
```go
client := openai.NewClient(apiKey)
```
取得したAPIキーを使ってOpenAIクライアントを作成します。

**3. ユーザー入力のループ処理**
```go
scanner := bufio.NewScanner(os.Stdin)
for {
    // ユーザー入力を受け取り、処理する
}
```
`bufio.Scanner`を使ってユーザーの入力を1行ずつ読み取り、無限ループで対話を継続します。

**4. Chat Completions APIの呼び出し**
```go
resp, err := client.CreateChatCompletion(
    context.Background(),
    openai.ChatCompletionRequest{
        Model: openai.GPT4Dot1Nano,
        Messages: []openai.ChatCompletionMessage{
            {
                Role:    openai.ChatMessageRoleUser,
                Content: userInput,
            },
        },
    },
)
```

### Chat Completions APIの詳細仕様

OpenAIのChat Completions APIは、現代的な会話型AIの基盤となるAPIです。

:::details Chat Completions APIの主要パラメータ

**主要パラメータ：**

- **`Model`**: 使用するAIモデル（例：`gpt-4.1-nano`, `gpt-4o`, `gpt-3.5-turbo`）
- **`Messages`**: 会話履歴の配列。各メッセージには`Role`と`Content`が含まれる
- **`Temperature`**: 応答のランダム性（0.0-2.0、デフォルト1.0）
- **`MaxTokens`**: 生成する最大トークン数
- **`Tools`**: Function Calling用のツール定義（Chapter 2で詳しく扱います）

**メッセージロール：**

```go
// システムメッセージ（AIの振る舞いを指定）
openai.ChatMessageRoleSystem

// ユーザーメッセージ（人間からの入力）
openai.ChatMessageRoleUser

// アシスタントメッセージ（AIからの応答）
openai.ChatMessageRoleAssistant

// ツールメッセージ（Function Calling結果、Chapter 2で使用）
openai.ChatMessageRoleTool
```

**レスポンス構造：**

APIのレスポンスは以下のような構造になっています。

```json
{
  "choices": [
    {
      "message": {
        "role": "assistant",
        "content": "こんにちは！何かお手伝いできることはありますか？"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 10,
    "completion_tokens": 15,
    "total_tokens": 25
  }
}
```
:::

:::message
**参考リンク**
- [OpenAI Chat Completions API 公式ドキュメント](https://platform.openai.com/docs/api-reference/chat)
:::

この実装では、ユーザーの入力を`ChatMessageRoleUser`として送信し、GPTからの応答を取得しています。モデルには高性能かつコスト効率の良い`GPT-4.1-nano`を使用しています。

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

実行可能ファイルが生成されるので、実行してみましょう。

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
nebula - OpenAI Chat CLI
Type 'exit' or 'quit' to end the conversation
---
You: 
```

試しに何か質問してると下記のような返答が返ってくるはずです。
これでChatGPTのように会話をすることができました！

```
You: こんにちは！Go言語について教えてください。
Assistant: はい、Go言語は、Googleによって開発されたプログラミング言語です。Go言語は、静的型付け言語であり...

You: exit
Goodbye!
```



:::message alert
**トラブルシューティング**

よくある問題と解決方法。

**問題1**: `Error: OPENAI_API_KEY environment variable is not set`
- **原因**: 環境変数が正しく設定されていない
- **解決策**: `export OPENAI_API_KEY=your_api_key_here` を実行後、`echo $OPENAI_API_KEY` で確認

**問題2**: API呼び出しでエラーが発生
- **原因**: APIキーの問題または残高不足
- **解決策**: 
  1. APIキーが正しいか確認
  2. OpenAI Platformで残高確認

**問題3**: ビルドエラーが発生
- **原因**: Go言語のバージョンが古い
- **解決策**: `go version` で1.23.1以上であることを確認
:::

## この章のまとめと次のステップ

この章では、GoからOpenAI APIを呼び出す基本的なCLIアプリケーションを作成しました。完成したコードの構造は以下の通りです。

### 作成したファイル一覧

```
nebula/
├── go.mod          # Goモジュール定義
├── go.sum          # 依存関係のチェックサム
├── main.go         # メインプログラム
└── nebula          # 実行可能ファイル（ビルド後）
```

### 達成できたこと

✅ **Goプロジェクトのセットアップ** - `go mod init`でモジュール化  
✅ **OpenAI API連携** - 公式ライブラリの導入と基本的な使い方  
✅ **セキュアな設定管理** - 環境変数を使った安全なAPIキー管理  
✅ **対話型CLI** - ユーザー入力とGPT応答のループ処理  

これで、LLMアプリケーション開発の基盤が整いました。
現在のプログラムは「ただの質問応答」ですが、次のChapter 2では、GPTに「道具」を与える**Function Calling**を実装します。

Function Callingを使うことで、GPTが単なる文章生成だけでなく、ファイルの読み込み、書き込みなどができるようになります。


次章では、最初のツール「`readFile`」を実装し、GPTがファイルシステムを探索できるようにしていきます！