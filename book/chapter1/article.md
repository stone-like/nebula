# Chapter 1: OpenAIと会話する最初のCLI

## はじめに

この章では、GoからOpenAI APIを呼び出して、ターミナルで対話できる最小のプログラムを作成します。

まずはシンプルなチャット機能から始めることで、LLMと会話する感覚を掴む感覚を得ていきましょう。

この章が終わる頃には。GPTと自由に会話できるプログラムが完成し、次章で実装するFunction Calling（ツール機能）の土台が整います。


## ハンズオン・チュートリアル

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

実行すると、`go.mod`に依存関係が追加され、`go.sum`ファイルも生成されます：

```go
// go.mod (更新後)
module nebula

go 1.23.1

require github.com/sashabaranov/go-openai v1.40.3 // indirect
```

### 3. APIキーを安全に環境変数から読み込む

OpenAI APIを使うには、APIキーが必要です。セキュリティの観点から、APIキーはコードに直接書かず、環境変数から読み込むのがベストプラクティスです。

まず、OpenAIのWebサイトでAPIキーを取得し、環境変数にセットしましょう：

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

注意:
 2025/07現在、APIKeyを使用するためには最初に5ドル支払いが必要です。
 GPT4.1-nanoを使用する限り1000回呼び出しても0.2ドルくらいなので特に5ドルを超える心配はいりません。
 右上の歯車マークをクリック -> Billingをクリックし、Auto recharge is offとなっていれば追加で支払いも行われないので安心です。


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

このプログラムの各部分を詳しく見ていきましょう：

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
ユーザーの入力をOpenAI APIに送信し、GPTからの応答を取得します。ここでは最新の`GPT-4.1-nano`モデルを使用しています。

## 動作確認

これでこの章の機能が動くようになりました！実際に試してみましょう。

まず、プログラムをビルドします：

**Linux/macOS の場合:**
```bash
go build -o nebula .
```

**Windows の場合:**
```bash
go build -o nebula.exe .
```

実行可能ファイルが生成されるので、実行してみましょう：

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
- `Error: OPENAI_API_KEY environment variable is not set`が出る場合：環境変数の設定を確認してください
- API呼び出しでエラーが出る場合：APIキーが正しく設定されているか、OpenAIアカウントに残高があるかを確認してください
:::

## この章のまとめと次のステップ

この章では、GoからOpenAI APIを呼び出す基本的なCLIアプリケーションを作成しました。完成したコードの構造は以下の通りです：

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