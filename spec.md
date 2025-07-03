
---

# コーディングエージェント「nebula」仕様書 (最終版)

## 1. 概要

### 1.1. エージェント名
**nebula**

### 1.2. 目的
本プロジェクトは、自律的に複数のファイルを編集・作成するコーディングエージェントの仕組みを学ぶための学習用プロジェクトである。最終的に、曖昧な指示からプロジェクトに新機能を追加したり、小規模なプロジェクトを対話形式で一から構築したりする能力を持つ、**言語非依存の**エージェントの開発を目指す。

*   **目標1（最優先）:** クリーンアーキテクチャのような階層化されたプロジェクトにおいて、コンテキストと少し曖昧な指示を基に、自律的に新機能の追加・変更を行えるようにする。
*   **目標2:** 複数回の対話を通じて、小規模なプロジェクト（例: CLIツール、簡単なWeb APIサーバーなど）をゼロから自律的に構築できるようにする。

### 1.3. 技術スタック
*   **LLM:** OpenAI GPTシリーズ (Tool Callingに対応したモデル)
*   **開発言語:** Go言語 (nebula自体の開発言語)

## 2. アーキテクチャと主要機能

### 2.1. 動作モード
ユーザーはエージェントの動作モードを切り替えることで、タスクの進行を制御する。

*   **planモード (計画モード)**
    *   ファイルシステムへの書き込みを一切行わない、読み取り専用のモード。
    *   ユーザーの指示に基づき、プロジェクト構造の解析、関連ファイルの特定、変更計画の立案に特化する。

*   **agentモード (実行モード)**
    *   `plan`モードで立てた計画やユーザーの指示に基づき、実際にファイルの作成・編集を行う。
    *   複数のファイルを横断して、自律的にコーディングタスクを遂行する。

### 2.2. コンテキスト初期化と動作モード判断
エージェントは起動時に、以下のフローで動作モードを判断し、コンテキストを準備する。

1.  **ディレクトリ状態の確認:** 作業ディレクトリが空（無視リストを除く）か、ファイルが存在するかを確認する。
2.  **動作モード分岐:**
    *   **既存プロジェクトモード (ファイルが存在する場合):**
        *   `NEBULA.md` の有無を確認し、あれば読み込み、なければ内部`init`を実行して、プロジェクトの概要を「初期コンテキスト」としてメモリに保持する。
    *   **新規プロジェクトモード (ディレクトリが空の場合):**
        *   「初期コンテキスト」は空の状態で開始する。エージェントは対話を通じてユーザーに要件を質問することからタスクを開始する。

### 2.3. コンテキスト生成 (`init`機能)
*   **目的:** プロジェクトの初期解析を行い、エージェントの思考の起点となるコンテキスト情報を生成する。
*   **実行トリガーと振る舞い:**
    *   **`nebula init` コマンド実行時:** Goプログラムがプロジェクトを解析し、LLMに要約させて生成したレポートを**`NEBULA.md`ファイルとして物理的に保存する。**
    *   **`plan`/`agent`モードでの内部実行時:** `NEBULA.md` が存在しない場合に自動実行。Goプログラムが収集した情報を基にLLMが生成したレポートを、**ファイルには書き出さず、そのセッション中のメモリ上の知識（初期コンテキスト）として保持する。**

## 3. 機能仕様 (Tool Calling)

### 3.1. 基本ツールセット
LLMがファイルシステムを操作するために、Go言語で実装しTool Callingとして提供する基本ツールセットです。

| 関数名 | 引数 | 戻り値 | 説明 |
| :--- | :--- | :--- | :--- |
| `list` | `path: string`, `recursive: bool`| `[]string` | 指定した`path`内のファイルとディレクトリの一覧を返す。 |
| `readFile` | `path: string` | `string` | 指定した`path`（ファイル）の完全な内容を文字列として返す。 |
| `writeFile` | `path: string`, `content: string` | `bool` | 指定した`path`に新しいファイルを作成し、`content`を書き込む。親ディレクトリは自動で作成する。 |
| `editFile` | `path: string`, `new_content: string` | `bool` | 指定した`path`（既存ファイル）の内容を`new_content`で完全に上書きする。 |
| `searchInDirectory` | `directory: string`, `keyword: string` | `[]string` | 指定した`directory`配下を再帰的に検索し、`keyword`を含むファイルパスのリストを返す。 |


writeFileとeditFileの際、実際にファイル作成/編集にユーザの許可をとること(diffは見せなくてよい。)

### 3.2. ツールスキーマ定義 (Go言語)
LLMに渡すための各ツールのスキーマ定義です。

#### `list`
```go
var ListTool = ToolDefinition{
    // Function: list,
    Schema: openai.Tool{
        Type: openai.ToolTypeFunction,
        Function: &openai.FunctionDefinition{
            Name:        "list",
            Description: "List files and directories in a specified path. Can be recursive.",
            Parameters: jsonschema.Definition{
                Type: jsonschema.Object,
                Properties: map[string]jsonschema.Definition{
                    "path": {
                        Type:        jsonschema.String,
                        Description: "The path of the directory to list.",
                    },
                    "recursive": {
                        Type:        jsonschema.Boolean,
                        Description: "If true, lists files and directories recursively. Defaults to false if not provided.",
                    },
                },
                Required: []string{"path"},
            },
        },
    },
}
```

#### `readFile`
```go
var ReadFileTool = ToolDefinition{
    // Function: readFile,
    Schema: openai.Tool{
        Type: openai.ToolTypeFunction,
        Function: &openai.FunctionDefinition{
            Name:        "readFile",
            Description: "Read the entire content of a specified file.",
            Parameters: jsonschema.Definition{
                Type: jsonschema.Object,
                Properties: map[string]jsonschema.Definition{
                    "path": {
                        Type:        jsonschema.String,
                        Description: "The path of the file to read.",
                    },
                },
                Required: []string{"path"},
            },
        },
    },
}
```

#### `writeFile`
```go
var WriteFileTool = ToolDefinition{
    // Function: writeFile,
    Schema: openai.Tool{
        Type: openai.ToolTypeFunction,
        Function: &openai.FunctionDefinition{
            Name: "writeFile",
            Description: "Write content to a new file. This will create parent directories if they do not exist. This operation fails if the file already exists.",
            Parameters: jsonschema.Definition{
                Type: jsonschema.Object,
                Properties: map[string]jsonschema.Definition{
                    "path": {
                        Type:        jsonschema.String,
                        Description: "The full path of the new file to be created.",
                    },
                    "content": {
                        Type:        jsonschema.String,
                        Description: "The content to write into the new file.",
                    },
                },
                Required: []string{"path", "content"},
            },
        },
    },
}
```

#### `editFile`
```go
var EditFileTool = ToolDefinition{
    // Function: editFile,
    Schema: openai.Tool{
        Type: openai.ToolTypeFunction,
        Function: &openai.FunctionDefinition{
            Name: "editFile",
            Description: "Completely overwrite an existing file with new content. CRITICAL: To avoid destroying the file, you MUST follow this workflow: 1. Use 'readFile' to get the full current content. 2. Construct the complete new version of the file in your thought process. 3. Use this tool to write the complete new version. Do not use this for partial edits; provide the entire file content.",
            Parameters: jsonschema.Definition{
                Type: jsonschema.Object,
                Properties: map[string]jsonschema.Definition{
                    "path": {
                        Type:        jsonschema.String,
                        Description: "The path of the existing file to edit.",
                    },
                    "new_content": {
                        Type:        jsonschema.String,
                        Description: "The new, full content that will overwrite the entire existing file.",
                    },
                },
                Required: []string{"path", "new_content"},
            },
        },
    },
}
```

#### `searchInDirectory`
```go
var SearchInDirectoryTool = ToolDefinition{
    // Function: searchInDirectory,
    Schema: openai.Tool{
        Type: openai.ToolTypeFunction,
        Function: &openai.FunctionDefinition{
            Name: "searchInDirectory",
            Description: "Search for a keyword recursively within a specified directory. Returns a list of file paths that contain the keyword.",
            Parameters: jsonschema.Definition{
                Type: jsonschema.Object,
                Properties: map[string]jsonschema.Definition{
                    "directory": {
                        Type:        jsonschema.String,
                        Description: "The starting directory for the recursive search.",
                    },
                    "keyword": {
                        Type:        jsonschema.String,
                        Description: "The keyword to search for within the files.",
                    },
                },
                Required: []string{"directory", "keyword"},
            },
        },
    },
}
```

## 4. メモリ機能
*   **会話履歴（短期記憶）:**
    *   ユーザーとの現在のセッションにおける対話の履歴全体を保持する。これにより、過去の指示や文脈を踏まえた応答が可能になる。LLMへのリクエストに含める形で実装。

## 5. プロンプト設計と実行フロー

### 5.1. プロンプトの基本方針
`nebula`は静的なプロンプトを使用せず、**「プロンプトテンプレート」**と、実行時に取得する**「動的コンテキスト」**を組み合わせて、最終的なプロンプトを生成します。

### 5.2. `init`用プロンプトテンプレート
```text
#役割
あなたは、既存のソースコードプロジェクトを分析し、その概要を開発者に分かりやすく説明する専門家です。

#指示
これから提示する「解析対象データ」を基に、このプロジェクトに関する概要レポートをMarkdown形式で作成してください。
レポートには、次の項目を必ず含めてください。

- **プロジェクト概要:** このプロジェクトが何をするためのものか（Webサーバー、CLIツールなど）を推測してください。
- **使用言語とフレームワーク:** 主要なプログラミング言語と、使用されている可能性が高いフレームワークやライブラリを挙げてください。
- **ディレクトリ構造の推測:** 各主要ディレクトリがどのような役割（例: `src`はソースコード、`pkg`はライブラリなど）を持っているかを推測してください。
- **次のステップの提案:** このプロジェクトを開発する上で、最初に行うべきこと（依存関係のインストール、ビルド方法など）を提案してください。

#解析対象データ
---
{ここにファイル構造のリストや主要ファイルの内容など、Goプログラムが収集した生データを動的に挿入}
---
```

### 5.3. エージェント用システムプロンプトテンプレート
```text
#役割
あなたは「nebula」という名前の、熟練したソフトウェア開発者であり、自律型コーディングエージェントです。あなたの目的は、ユーザーの指示とプロジェクトのコンテキストに基づき、ソースコードを計画的かつ正確に変更、または新規作成することです。

#初期コンテキスト
まず、以下のプロジェクト概要を読んで全体像を把握してください。これは、あなたの思考の出発点となります。
---
{dynamic_context}
---

#ファイル編集の厳格なルール
既存ファイルを編集する際は、必ず以下の「Read-Modify-Write」パターンに従ってください。
1.  `readFile`を使用して、対象ファイルの完全な最新の内容を読み取ります。
2.  あなたの思考の中で、読み取った内容を基に変更後の「ファイルの完成形」を構築します。
3.  `editFile`を呼び出し、`new_content`引数にその「ファイルの完成形」の全体を渡して、ファイルを完全に上書きします。
部分的な編集や追記のためにこのツールを使用してはいけません。常にファイル全体を置き換えるようにしてください。

#思考と実行のプロセス
以下のプロセスに従って、思考と行動を決定してください。

1.  **思考(Thought):**
    *   **もし初期コンテキストが空の場合（新規プロジェクト作成時）:** まず、どのようなプロジェクトを作成したいか（言語、目的、フレームワークなど）をユーザーに質問してください。対話を通じて要件を定義することから始めます。
    *   **もし初期コンテキストが存在する場合:** 初期コンテキストとユーザーの要求を分析し、現在のタスクを明確に定義します。

2.  **計画(Plan):**
    定義されたタスクを達成するためのステップを計画します。新規作成の場合は、まずディレクトリ構造の作成と主要な設定ファイルの生成から計画します。

3.  **探索フェーズ(Exploration Phase):**
    このフェーズの目的は、実装に必要な情報をすべて集めることです。ファイルの書き込みや編集は**絶対に**行いません。
    - `list`を使ってプロジェクトのディレクトリ構造を把握します。
    - `searchInDirectory`や`readFile`を駆使して、関連する具体的なファイルやコード箇所を特定します。

4.  **実装フェーズ(Implementation Phase):**
    探索フェーズで得た情報に基づき、具体的なコードの作成・編集を行います。
    - 編集の場合は、必ず「ファイル編集の厳格なルール」に従ってください。
    - `writeFile`（新規作成）または`editFile`（既存ファイルの編集）を使って、変更を適用します。

5.  **完了報告:**
    現在のステップが完了したら、ユーザーに結果を報告し、次の指示を待ちます。
```

### 5.4. エージェントの実行フロー

```mermaid
graph TD
    A[ユーザーが `nebula agent "..."` を実行] --> B{作業ディレクトリは空か？};
    B -- No/既存 --> C{NEBULA.mdは存在するか？};
    B -- Yes/新規 --> F[空のコンテキストをセット];
    C -- Yes --> G[NEBULA.mdを読み込み<br>コンテキストとして保持];
    C -- No --> D[内部initプロセスを開始];
    D -- ファイル構造と内容を収集 --> E[LLMに要約を依頼<br><b>(1回目のLLMコール)</b>];
    E -- 生成された要約レポート --> H[レポートを<br>コンテキストとして保持];
    F --> I[コンテキストとユーザー指示を<br>プロンプトに埋め込む];
    G --> I;
    H --> I;
    I --> J[エージェントの思考サイクルを開始<br><b>(2回目以降のLLMコール)</b>];
    J <--> K[Tool Calling<br>(list, readFile, editFile...)];
    K --> J;
    J --> L[ユーザーに応答];
```

## 6. 学習段階の設計理論

### 6.1. 段階的能力獲得アプローチ

`nebula`の開発は、エージェントの能力を段階的に獲得する設計になっている。各段階で前段階の限界を体験し、次段階の必要性を理解することで、自然な学習フローを実現する。

### 6.2. 学習段階の構造

#### Stage 1: 基本ツール実装 (Chapter 1-3)
- **目標**: LLMにファイルシステム操作能力を与える
- **成果物**: readFile, list, searchInDirectory, writeFile
- **限界**: 個別ツールは動作するが、連携した思考プロセスがない

#### Stage 2: Read-Modify-Write確立 (Chapter 4)
- **前段階の問題**: editFileツールを実装しても、LLMがreadFile→editFileの流れを実行しない
- **解決策**: 最小限のシステムプロンプト（基本的なRead-Modify-Write指示）
- **成果物**: 単一ファイル編集が可能
- **限界**: 複数ファイル編集や複雑な思考プロセスで破綻

#### Stage 3: 思考プロセス確立 (Chapter 5)
- **前段階の問題**: 最小限プロンプトでは複雑なタスクでエージェントが迷子になる
- **解決策**: 本格的システムプロンプト（思考→計画→探索→実装）
- **成果物**: 明確な指示での複数ファイル編集が可能
- **限界**: 曖昧な指示では「何をすべきか」がわからない

#### Stage 4: コンテキスト理解獲得 (Chapter 6)
- **前段階の問題**: 「/healthエンドポイントを追加して」のような曖昧な指示では、どのファイルを編集すべきかわからない
- **解決策**: init機能によるプロジェクト自己分析とコンテキスト生成
- **成果物**: 曖昧な指示での複雑な機能追加が可能

### 6.3. 「限界体験→解決」サイクルの重要性

各段階で意図的に前段階の限界を体験することで：

1. **問題の実感**: 理論だけでなく、実際の問題を体験する
2. **解決策の必要性理解**: なぜその機能が必要なのかを体感的に理解する
3. **学習動機の維持**: 「動かない→動く」の成功体験によるモチベーション向上
4. **設計思想の理解**: なぜその設計にしたのかの背景を理解する

### 6.4. システムプロンプト vs コンテキストの役割分担

#### システムプロンプトの役割
- **思考プロセスの制御**: どのような順序で何を考えるか
- **行動規範の設定**: ファイル編集時の安全性ルールなど
- **ツール使用方法の指示**: Read-Modify-Writeパターンの強制など

#### コンテキスト（init機能）の役割
- **プロジェクト固有知識の提供**: アーキテクチャ、ファイル構造、技術スタック
- **関連ファイルの特定支援**: 曖昧な指示から具体的な編集対象を特定
- **ドメイン知識の補完**: そのプロジェクトでの慣例やパターン

この役割分担により、汎用的な思考能力（システムプロンプト）と、プロジェクト固有の知識（コンテキスト）を分離し、再利用可能な設計を実現している。