
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

### 2.2. 動作モード
エージェントは起動時に、基本的な対話開始と必要に応じたプロジェクト分析を行う。

*   **既存プロジェクトでの動作:**
    *   プロジェクトの基本構造を理解し、ユーザーの指示に基づいて適切なファイルを特定・編集する。
*   **新規プロジェクトでの動作:**
    *   対話を通じてユーザーに要件を質問し、段階的にプロジェクトを構築する。

### 2.3. 設定管理 ✅ **実装完了**
*   **環境変数:** OPENAI_API_KEYによるAPI認証
*   **モデル選択:** GPT-4.1-nano（デフォルト）およびGPT-4.1-mini（複雑タスク用）の対話的切り替え機能
*   **設定ファイル:** ~/.nebula/config.json による永続的な設定保存
*   **対話的操作:** 実行中に`model`コマンドでモデル切り替え可能

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
*   **永続記憶（Chapter6で実装）:**
    *   SQLite (`modernc.org/sqlite`) を使用したセッション間での会話履歴の保存・復元機能。
    *   プロジェクトディレクトリごとに独立したセッション管理を実現。
    *   シンプルな2テーブル構成: `sessions` (セッション管理) + `messages` (メッセージ履歴)

## 5. プロンプト設計と実行フロー

### 5.1. プロンプトの基本方針
`nebula`は静的なプロンプトを使用せず、**「プロンプトテンプレート」**と、実行時に取得する**「動的コンテキスト」**を組み合わせて、最終的なプロンプトを生成します。

### 5.2. planモード用システムプロンプト（Chapter6で追加）
planモードでは読み取り専用でタスクの計画を立てる専用のプロンプトを使用します。

### 5.3. エージェント用システムプロンプトテンプレート
```text
#役割
あなたは「nebula」という名前の、熟練したソフトウェア開発者であり、自律型コーディングエージェントです。あなたの目的は、ユーザーの指示に基づき、ソースコードを計画的かつ正確に変更、または新規作成することです。

#ファイル編集の厳格なルール
既存ファイルを編集する際は、必ず以下の「Read-Modify-Write」パターンに従ってください。
1.  `readFile`を使用して、対象ファイルの完全な最新の内容を読み取ります。
2.  あなたの思考の中で、読み取った内容を基に変更後の「ファイルの完成形」を構築します。
3.  `editFile`を呼び出し、`new_content`引数にその「ファイルの完成形」の全体を渡して、ファイルを完全に上書きします。
部分的な編集や追記のためにこのツールを使用してはいけません。常にファイル全体を置き換えるようにしてください。

#思考と実行のプロセス
以下のプロセスに従って、思考と行動を決定してください。

1.  **思考(Thought):**
    ユーザーの要求を分析し、現在のタスクを明確に定義します。新規プロジェクトの場合は、対話を通じて要件を定義することから始めます。

2.  **計画(Plan):**
    定義されたタスクを達成するためのステップを計画します。

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
    A[ユーザーが `nebula` を実行] --> B[システムプロンプトでLLM初期化];
    B --> C[ユーザー指示の受付];
    C --> D[思考・計画フェーズ];
    D --> E[探索フェーズ<br>(list, readFile, searchInDirectory)];
    E --> F[実装フェーズ<br>(writeFile, editFile)];
    F --> G[完了報告];
    G --> C;
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

#### Stage 3: 思考プロセス確立 (Chapter 5) ✅ **完了**
- **前段階の問題**: 最小限プロンプトでは複雑なタスクでエージェントが迷子になる
- **解決策**: Gemini CLI-inspiredシステムプロンプト（思考→計画→探索→実装）
- **成果物**: 明確な指示での複数ファイル編集が可能、モデル選択機能、設定管理システム
- **達成した機能**: 
  - 本格的システムプロンプトによる段階的実行プロトコル
  - GPT-4.1-nano/miniの対話的切り替え機能
  - ~/.nebula/config.json による設定の永続化
  - JSON処理の安全性強化（制御文字除去）
  - Clean Architecture プロジェクトでの複数ファイル編集テスト完了

#### Stage 4: 実用機能確立 (Chapter 6)
- **前段階の問題**: 実行前の計画確認ができない、セッション情報が保持されない
- **解決策**: planモードと永続記憶機能の追加
- **成果物**: 安全性とユーザビリティを備えた実用的なツール

### 6.3. 「限界体験→解決」サイクルの重要性

各段階で意図的に前段階の限界を体験することで：

1. **問題の実感**: 理論だけでなく、実際の問題を体験する
2. **解決策の必要性理解**: なぜその機能が必要なのかを体感的に理解する
3. **学習動機の維持**: 「動かない→動く」の成功体験によるモチベーション向上
4. **設計思想の理解**: なぜその設計にしたのかの背景を理解する

### 6.4. 最終的な機能セット

#### Chapter5での完成機能
- **基本ツールセット**: ファイル操作の完全実装
- **Read-Modify-Write確立**: 安全なファイル編集パターン
- **思考プロセス確立**: 探索→実装の段階的実行
- **モデル選択**: GPT-4.1-nano/miniの切り替え（interactive "model" command）
- **設定管理**: ~/.nebula/config.json による永続的な設定保存

#### Chapter6での追加機能
- **永続記憶**: SQLiteベースのセッション間会話履歴保存・復元
- **プロジェクト固有記憶**: ディレクトリベースの独立したセッション管理
- **planモード**: 実行前の安全な計画確認（読み取り専用モード）
- **プロジェクト完成**: ドキュメント整備と最終調整

### 永続記憶機能の詳細設計

#### データベーススキーマ
```sql
-- セッション管理テーブル
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,                    -- セッションID (timestamp-based)
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    ended_at DATETIME,                      -- NULL = アクティブセッション
    project_path TEXT NOT NULL,            -- 作業ディレクトリパス
    model_used TEXT NOT NULL               -- 使用モデル (gpt-4.1-nano/mini)
);

-- メッセージ履歴テーブル  
CREATE TABLE messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT REFERENCES sessions(id),
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    role TEXT NOT NULL,                    -- 'user', 'assistant', 'tool'
    content TEXT,                          -- メッセージ内容
    tool_calls TEXT,                       -- JSON形式のツール呼び出し情報
    tool_results TEXT                      -- JSON形式のツール実行結果
);
```

#### セッション管理の流れ
1. **起動時**: 現在ディレクトリの過去セッション一覧表示
2. **セッション選択**: 新規作成 or 既存セッション復元
3. **会話中**: 全メッセージを自動的にデータベースに保存
4. **終了時**: セッション終了時刻を記録

#### プロジェクト固有記憶の実現
- 各プロジェクトディレクトリで独立したセッション履歴
- 他プロジェクトの会話履歴は表示されない
- ディレクトリベースのコンテキスト分離

この構成により、実用的で信頼性の高い自律型コーディングエージェントが完成する。