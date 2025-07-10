# Chapter 5: システムプロンプト設計と安全性強化

## はじめに

この章では、nebulaエージェントにいよいよコーディングエージェントらしさを与えていきます。

この章を終える頃には、nebulaは複数ファイル編集能力を獲得し、複雑なタスクもこなせるようなエージェントとなります。
まずは現状はなぜ複雑なタスクができないか、から確認していきましょう。

:::note この章で学ぶこと
**システムプロンプトエンジニアリングの完全ガイド**

🎯 **解決する問題**
- なぜLLMは複雑なタスクで失敗するのか？
- どうすれば推測を防いで確実な情報収集をさせられるか？
- 自動実行を止めずに完了まで実行させる方法は？

🛠️ **実装する機能**
- 強制的な実行プロトコル（情報収集→実装）
- GPT-4.1-nano/mini モデル選択機能
- 設定ファイル管理システム

📚 **学習内容**
- 失敗するプロンプトパターンとその理由
- 効果的なプロンプト設計の5つの原則
- 実際のシステムプロンプトの構成要素別解説
- モデル差への対応とコスト管理

**クイックナビ**: [課題](#システムプロンプトで解決したい課題-16) → [失敗例](#失敗するプロンプトパターンとその理由-26) → [原則](#効果的なプロンプト設計の原則-36) → [解説](#実際のプロンプトの構成要素別解説-46) → [実装](#モデル選択機能の実装と使い方-56) → [テスト](#システムプロンプト実装-66)
:::


## 📁 この章での到達目標構造

```
nebula/
├── main.go                 # システムプロンプト統合
├── config/                 # 新規パッケージ
│   └── config.go          # 設定管理・モデル選択
├── tools/                  
│   ├── common.go          
│   ├── readfile.go        
│   ├── list.go            
│   ├── search.go          
│   ├── writefile.go       
│   ├── editfile.go        
│   └── registry.go        
├── go.mod                 
└── go.sum                 
```

**前章からの変化:**
- Chapter 4: editFile
- Chapter 5: **システムプロンプト + 設定管理** ← 今ここ

**実装する機能:**
- システムプロンプト
- gpt-4.1-nano/mini モデル選択機能
- 設定ファイル管理（~/.nebula/config.json）

**新規追加:**
- `config/config.go`: Config構造体、LoadConfig()、SaveConfig()
- システムプロンプト関数群

**ユーザーディレクトリ:**
```
~/.nebula/
└── config.json            # ユーザー設定ファイル
```

## システムプロンプトで解決したい課題 [1/6]

Chapter 4まで実装されたnebulaは、基本的なファイル操作はできるようになりました。しかし、複雑なタスクを実行しようとすると、以下のような問題に直面します。

### 具体的な問題例

次のような指示を実行してみると、問題が明確になります。

```
tools/writeFile.goを参考に、tools/copyFile.goを作成してください。ファイルをコピーする機能を実装し、tools/registry.goに登録してください
```

**現在のnebulaの実際の失敗動作：**

1. **参照ファイルを読まずに実装開始**
   - `tools/writeFile.go`の内容を確認しない
   - 「知っているつもり」で実装を始める
   - 結果：既存のコードスタイルと異なる実装

2. **推測による間違い**
   - ファイル拡張子を推測（`.ts`と間違える）
   - ディレクトリ構造を推測（`src/`があると思い込む）
   - 関数名やパターンを推測

3. **断片的な作業**
   - `copyFile.go`を作成するが、`registry.go`への登録を忘れる
   - または、registry.goの現在の構造を確認せずに追加

4. **実行の途中停止**
   - 「実装してもよろしいですか？」と途中で確認を求める
   - ユーザーが「はい」と答えるまで作業を停止


これらの問題は、**システムプロンプトがない**ことが原因です。

Chapter 1-4の実装では、LLMに特別な行動指針を与えず、純粋にFunction Callingでツールを使ってもらっていました。

```go
// Chapter 1-4: システムプロンプトなし
messages := []openai.ChatCompletionMessage{
    {
        Role:    openai.ChatMessageRoleUser,  // ユーザーメッセージのみ
        Content: userInput,
    },
}
```

このため、nebulaは公にあるコーディングエージェントとは違い、複雑なタスクだと下記のようにどうしていいかわからない状態に陥ってしまうのです。
- **どのように調査すべきか**がわからない
- **何を禁止すべきか**がわからない
- **どの順序で作業すべきか**がわからない


そこでシステムプロンプトを追加することで、nebulaに「思考のフレームワーク」を与え、以下を実現していきます。

- **一貫した思考プロセス**: 毎回同じ手順でタスクに取り組む
- **推測の禁止**: 事実に基づく実装を強制
- **自動実行**: 途中で停止しない連続的な作業

:::summary 重要ポイント
**システムプロンプトで解決する3つの根本問題**
1. **推測による実装** → 事実に基づく情報収集を強制
2. **断片的な作業** → プロジェクト全体を意識した連携実装
3. **実行の途中停止** → 自動実行による連続的な作業フロー
:::

---

nebulaが複雑なタスクで失敗する根本的な原因を理解したところで、次は**実際によくある失敗パターン**を見ていきましょう。筆者が試行錯誤する中で陥った失敗例から、効果的でないプロンプトの特徴を学びます。

## 失敗するプロンプトパターンとその理由 [2/6]

システムプロンプトを設計していく中で、何も知らなかった筆者が陥ってしまった失敗パターンが何個かあります。
実際の例を見ながら、なぜそれらが効果的でないかを理解しましょう。


### ❌ パターン1: 弱い表現の使用

**失敗例：**
```text
# 基本ルール
1. ファイルを編集する前に、できればreadFileで内容を確認してください
2. 可能であれば、プロジェクト構造を理解してください
3. 必要に応じて、searchInDirectoryで検索してください
```

**なぜ失敗するか：**
- 「できれば」「可能であれば」「必要に応じて」では強制力が弱い
- 推測してしまって既存ファイルの内容考えずにwriteとかeditとかをしてしまう

**実際の失敗動作：**
```
User: "認証機能を追加してください"
→ LLM: 結果：間違った場所にファイルを作成、思った内容と違う内容での編集
```

### ❌ パターン2: 手順の不明確性

**失敗例：**
```text
# 実行手順
1. 情報収集を行う
2. 実装を行う
3. 結果を確認する
```

**なぜ失敗するか：**
- 各ステップが抽象的すぎる
- 「情報収集」の具体的な内容が不明
- ステップ間の移行条件が曖昧
- 情報取集-> 実装の段で自動実行の指示がない　->　なのでユーザーにこれでいいですか？と聞くだけで終わってしまう可能性が高い

**実際の失敗動作：**
```
User: "APIルーターを追加してください"
→ LLM: 情報収集を開始
→ LLM: 「実装を開始してもよろしいですか？」← 途中で停止
→ ユーザーは「はい」と答える必要がある

もしくは、情報収集が不十分のまま実装開始されてしまう。
```


### ❌ パターン3: 日本語でプロンプトを書く
ここまで理解のため日本語でプロンプトを書いてきましたが、
システムプロンプトは長文ということもあるせいか日本語では一部分、だけど大事な部分が伝わってくれないということがありました。
もしかしたら最上位のモデルだったりプロンプトをもっと工夫すれば大丈夫なのかもしれません。

ただ、できるなら英語でプロンプトを書く方が良さそうです。

### 失敗パターン vs 成功パターン 比較表

| 要素 | ❌ 失敗パターン | ✅ 成功パターン | 結果 |
|------|----------------|----------------|------|
| **表現の強さ** | 「できれば」「可能であれば」 | 「NEVER」「MUST」「FORBIDDEN」 | LLMが推測せず確実に実行 |
| **手順の明確さ** | 「情報収集を行う」（抽象的） | 「Use readFile ALL reference files」（具体的） | 必要な情報を確実に収集 |
| **自動実行** | 「実装してもよろしいですか？」 | 「proceed automatically without asking」 | 途中停止なしで完了まで実行 |
| **禁止事項** | 一般的な注意事項 | 具体例付きFORBIDDEN項目 | 典型的ミスを事前に防止 |
| **言語** | 日本語での長文指示 | 英語での構造化指示 | 重要な部分も確実に伝達 |

:::summary 重要ポイント
**失敗するプロンプトの3つの特徴**
1. **弱い表現**：「できれば」「可能であれば」→ LLMが推測に走る
2. **手順が不明確**：抽象的な指示→ 自動実行されずに途中停止
3. **日本語使用**：長文だと重要な部分が伝わらない→ 英語推奨
:::

---

失敗パターンを分析したことで、「何がうまくいかないか」が明確になりました。次は、これらの失敗から学んだ教訓を**実際に使える原則**にまとめていきます。効果的なシステムプロンプトを設計するための具体的な方法論を見ていきましょう。

## 効果的なプロンプト設計の原則 [3/6]

失敗パターンを踏まえ、効果的なシステムプロンプトの設計原則を整理します。


### ファイル編集/書き込みの前に調査をしっかりさせる
調査をしっかりさせることにより、既存のコードを理解した上でのファイル編集/新規作成が可能になります。
そのためにやった方がいいことについて記載していきます。

#### 原則: 守らせたいものは強制的な表現を使う

- 「NEVER」「MUST」「FORBIDDEN」
- 「MANDATORY」「REQUIRED」「Non-Negotiable」

#### 原則: 具体的な禁止事項を明示する

```text
❌ FORBIDDEN: Guessing file names (e.g., assuming "todo.ts" exists without checking)
❌ FORBIDDEN: Guessing file extensions (e.g., assuming .js when it might be .ts)
❌ FORBIDDEN: Guessing directory structure (e.g., assuming files are in "src/" without checking)
```

#### 原則: 段階的で明確な実行プロトコル

```text
## Step 1: Information Gathering (Required, but proceed automatically)
- Use 'list' to understand project structure
- Use 'readFile' to read ALL reference files
- Use 'searchInDirectory' to find related files

## Step 2: Implementation (Proceed automatically after Step 1)
- Use 'writeFile' for new file creation
- Use 'editFile' for existing file modification
```

#### 原則: 実例による説明
下記のように実際の実行プロトコルがどのように進むかの例示を与えてあげています。
これはchain-of-thought (CoT)プロンプトという手法で、LLMにどういう形で思考すればよいかを示してあげることで、
複雑なタスクでより良く動くようにする手法です。
```text
## Example 1: File Extension Discovery
Request: "Add a todo feature to the app"
**Correct sequence:**
1. list(".") ← Discover if files are .js, .ts, .py, .go, etc.
2. Find actual todo-related files with search or list
3. readFile the discovered files to understand patterns
4. Implement using the correct extension and patterns
```


### 調査の段階でストップさせないために

#### 原則: 自動実行の強制
このように調査 -> 実装へはユーザーに聞かないで進んでくださいと書かないと、調査だけで終わってしまう事態が多発したため。
```text
**IMPORTANT: Proceed from Step 1 to Step 2 automatically without asking for permission or confirmation.**
```


:::summary 重要ポイント
**効果的なプロンプト設計の5つの原則**
1. **強制的表現**：NEVER、MUST、FORBIDDEN で推測を禁止
2. **具体的禁止事項**：実例付きで何をしてはいけないかを明示
3. **段階的実行プロトコル**：Step 1（情報収集）→ Step 2（実装）
4. **自動実行の強制**：「proceed automatically without asking」を明記
5. **実例による説明**：正しい手順と間違った手順の対比
:::

---

下記にプロンプトテクニック集が学べるリンクを貼っておきますので、興味のある方は見てみるのも良さそうです。

:::message
**参考リンク**
- [Prompt Engineering Guide](https://www.promptingguide.ai/jp)
:::


設計原則がまとまったところで、次は**実際のシステムプロンプトがどう構成されているか**を詳しく見ていきます。
理論を実践に落とし込む具体的な方法と、各要素がなぜ効果的なのかを解説していきます。



## 実際のプロンプトの構成要素別解説 [4/6]

これらの原則を踏まえ、実際に使用しているシステムプロンプトの各部分を詳しく解説します。

### 基本構成の概要

システムプロンプトは以下の6つの要素で構成されています：

1. **Role（役割定義）** - エージェントの身分と能力を明確化
2. **Critical Rules（非交渉的ルール）** - 絶対に守るべき5つのルール
3. **Why説明（理由の説明）** - 情報収集の重要性を理論的に説明
4. **Execution Protocol（実行プロトコル）** - Step 1→Step 2の強制的な流れ
5. **禁止事項リスト** - 具体例付きでFORBIDDENパターンを明示
6. **実行例** - 正しい手順と間違った手順の対比

### 詳細解説

#### 1. Role（役割定義）

```text
# Role
You are "nebula", an expert software developer and autonomous coding agent.
```

**効果的な理由：**
- 明確な身分・役割の定義
- 「expert」で高い能力を期待
- 「autonomous」で自律的な行動を促進

:::insight 重要な洞察
**なぜ「expert」「autonomous」が重要なのか**

単に「assistant」や「helper」と定義すると、LLMは受動的になりがちです。
「expert software developer」と明示することで、積極的で高度な判断を促し、
「autonomous」で自律的な行動（途中で止まらない）を期待できます。
:::

#### 2. Critical Rules（非交渉可能なルール）

```text
# Critical Rules (Non-Negotiable)
1. **NEVER assume or guess file contents, names, or locations** - You must explore to understand them
2. **Information gathering is MANDATORY before implementation** - Guessing leads to immediate failure
3. **Before using writeFile or editFile, you MUST have used readFile on reference files**
4. **NEVER ask for permission between steps** - Proceed automatically through the entire workflow
5. **Complete the entire task in one continuous flow** - No pausing for confirmation
```

**効果的な理由：**
- 「**Non-Negotiable**」で交渉の余地がないことを強調
- 「**NEVER**」「**MUST**」で強制的な表現
- 各ルールに理由を付加（「- You must explore...」）
- 推測を完全に禁止
- 情報収集を強制
- 自動実行を強制

:::warning 注意点
**弱い表現を使ってはいけない理由**

「please」「try to」「if possible」などの丁寧な表現は、LLMに「optional（任意）」という印象を与えます。
強制的な表現（NEVER、MUST、FORBIDDEN）により、「必須」として認識させることが重要です。
:::

#### 3. Why Information Gathering is Critical（理由の説明）

```text
# Why Information Gathering is Critical
- **File structures vary**: What you expect vs. what exists are often different
- **Extensions matter**: .js vs .ts vs .go vs .py affects implementation
- **Directory layout matters**: Different projects have different organization
- **Assumption costs**: Guessing wrong means complete rework
```

**効果的な理由：**
- 情報収集の重要性を理論的に説明
- 具体的な失敗例を示唆（「.js vs .ts」）


#### 4. Execution Protocol（実行プロトコル）

```text
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
```

**効果的な理由：**
- 「**mandatory sequence**」で強制的な順序を明示
- 「**proceed automatically**」で自動実行を2回強調
- 各ステップの具体的な行動を明示
- 「**Internal Verification**」でチェックリスト形式
- 「**Required: YES**」で必須条件を明確化
- 最後に再度自動実行を強調

:::tip 実装のコツ
**自動実行を確実にする方法**

「proceed automatically」を複数回繰り返し、「without asking for permission」を明示的に書くことで、
LLMが途中で「実装してもよろしいですか？」と聞いて止まる問題を防げます。
また、Internal Verificationのチェックリストで自己確認させることも効果的です。
:::

#### 5. Common Mistakes to Avoid（失敗パターンの具体例）

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
```

**効果的な理由：**
- 「❌ **FORBIDDEN**」で視覚的に禁止を強調
- 具体的な失敗例を括弧内で提示
- LLMが陥りやすい典型的なミスを網羅
- 自動実行の阻害行動も禁止項目に含める

#### 6. Execution Examples（実行例）

```text
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
```

**効果的な理由：**
- 正しい手順と間違った手順を対比
- 具体的なコマンド例を提示
- 「← FORBIDDEN」で禁止理由を明示
- 複数の具体例で理解を深める

この構成により、LLMは：
- **何をすべきか**が明確にわかる
- **何をしてはいけないか**が具体的にわかる
- **なぜその行動が必要か**を理解できる
- **どの順序で行動すべきか**が明確になる

:::summary 重要ポイント
**システムプロンプトの構成要素**
1. **Role**：「expert」「autonomous」で能力と自律性を定義
2. **Critical Rules**：NEVER/MUSTで非交渉的なルールを設定
3. **Why説明**：情報収集の重要性を理論的に説明
4. **Execution Protocol**：Step 1→Step 2の強制的な流れ
5. **禁止事項リスト**：具体例付きでFORBIDDENパターンを明示
6. **実行例**：正しい手順と間違った手順の対比
:::

### 最終的なプロンプトの全体像

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

最終的に上記のようなプロンプトになりました。
このプロンプトに至るまでにGemini CLIやAIにプロンプトを添削してもらったりいろいろ試行錯誤した結果こんな感じで下記の要素を盛り込んでいます。

- **強制的な表現**（NEVER、MUST、FORBIDDEN）
- **具体的な禁止事項**（実例付き）
- **理由の明確化**（Why情報収集が重要か）
- **段階的な実行プロトコル**（Step 1 → Step 2）
- **自動実行の強制**（途中停止の禁止）
- **実行例**（正しい手順と間違った手順の対比）

:::message
**参考リンク**
- [Gemini CLI ソースコード](https://github.com/google-gemini/gemini-cli/blob/main/packages/core/src/core/prompts.ts)
:::

### main.goとの統合

`getSystemPrompt`関数を定義しただけでは、実際にLLMに渡されません。`handleConversation`関数内でシステムプロンプトを適切に統合する必要があります。

#### システムプロンプトの統合コード

```go
// handleConversation関数内でシステムプロンプトを統合
func handleConversation(client *openai.Client, cfg *config.Config, memoryManager *memory.Manager, toolSchemas []openai.Tool, toolsMap map[string]tools.ToolDefinition, userInput string, messages []openai.ChatCompletionMessage, planMode bool) []openai.ChatCompletionMessage {
	// システムプロンプトが設定されていない場合は最初に追加
	// （復元されたメッセージにはシステムプロンプトが含まれていない可能性があるため）
	hasSystemPrompt := false
	if len(messages) > 0 && messages[0].Role == openai.ChatMessageRoleSystem {
		hasSystemPrompt = true
	}

	if !hasSystemPrompt {
		// システムプロンプトを先頭に追加
		systemMessage := openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: getSystemPrompt(),
		}
		messages = append([]openai.ChatCompletionMessage{systemMessage}, messages...)
	}

	// ユーザーメッセージを履歴に追加
	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})
	
	// 以下、API呼び出しの処理が続く...
}
```

#### 統合の重要ポイント

1. **セッション復元時の考慮**: 
   - 復元されたメッセージ履歴にはシステムプロンプトが含まれていない場合がある
   - `hasSystemPrompt`フラグで確認し、必要に応じて先頭に追加

2. **メッセージ配列の構造**:
   - OpenAI APIでは、システムメッセージは配列の先頭に配置する必要がある
   - `append([]openai.ChatCompletionMessage{systemMessage}, messages...)`で既存メッセージの前に挿入

3. **一度だけ追加**:
   - 同じ会話セッション内で複数回システムプロンプトを追加しないよう制御

この統合により、nebulaは起動時から一貫してシステムプロンプトに従った行動を取るようになり、**情報収集→実装**の自動実行フローが確実に動作します。

### モデルによるプロンプト準拠の違い

ここまでシステムプロンプトを作成し、GPT-4.1-miniでならプロンプト準拠で動き、複雑なタスクもこなせるようになりました。

現在使用しているGPT-4.1-nanoについては、確かにシステムプロンプトがある方が成功の度合いはグンと高まります。
しかし、まだまだ自動で実装段階に行ってくれなかったり、作業を一つだけ抜かして進めてしまったりとプロンプト準拠しない場合もありました。

GPT-4.1-nanoでうまく進めたいのならユーザープロンプトを本当にがちがちに書かないと上手くタスク完了をしてくれない可能性があります。
ですので、できればGPT-4.1-miniを使用するのが良いです。

ただこのままではモデル切り替えもできませんし不便なので、モデル選択機能を作ってみましょう！

#### モデル選択機能の実装と使い方 [5/6]


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

まず、プロジェクトをビルドします。

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

成功すると、次のような出力が表示されます。

```
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

実用的なシステムプロンプトが完成し、モデル選択機能も追加できたところで、実際の運用で発生する可能性がある技術的な問題に対処しておきましょう。

**問題: editFile/writeFileでの制御文字混入**

OpenAI Function Callingでは、稀にLLMが生成した文字列にUnicodeの制御文字（`\u0006`など）が含まれることがあります。これらの制御文字がファイルに書き込まれると、Goコンパイラがエラーを出す原因となります。

**解決策: シンプルな制御文字除去**

`tools/json_helpers.go`を作成し、ファイル書き込み前の安全性チェックを追加します。

```go
package tools

import (
	"strings"
)

// CleanControlCharacters は文字列から制御文字を除去する
func CleanControlCharacters(s string) string {
	return strings.Map(func(r rune) rune {
		// タブ、改行、復帰文字以外の制御文字を除去
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1 // 制御文字を除去
		}
		return r
	}, s)
}
```

このヘルパー関数を、`writeFile`と`editFile`のファイル書き込み前に適用することで、安全で問題のないファイルを作成できます。

**適用例:**
```go
// writeFile/editFile での使用
cleanContent := CleanControlCharacters(originalContent)
// cleanContentでファイル書き込み実行
```

**設計原則:**
- **最小限の介入**: 必要な制御文字（改行等）は保持
- **透明性**: ユーザーには見えない安全性の確保
- **信頼性**: Go標準ライブラリの活用

## システムプロンプト実装 [6/6]

上記で設計したシステムプロンプトを実際のコードに実装し、その効果を確認しましょう。

### セットアップ手順

実際にnebulaの改善効果を体験するため、todo-appを使った実践テストを行います。

**Linux/macOS の場合:**
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

**Windows の場合:**
```cmd
# 1. nebulaリポジトリをクローン
git clone <nebula-repo>
cd nebula

# 2. todo-appをコピー
xcopy test\todo-app todo-app /E /I
cd todo-app

# 3. git初期化（元の.gitディレクトリはない状態）
git init

# 4. 初期コミット（実験のベースライン）
git add .
git commit -m "Initial todo-app for nebula experiments"
```

### todo-appの構成

todo-appは、Clean Architectureに基づいたTODO管理APIです。

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
上記で上手く行くはずですが、もし上手くいかないようならもうちょっとプロンプトを詳しく書き、下記のようにしてみてください。

```
Goで書かれている本プロジェクトのTODOアプリに優先度機能を追加してください。具体的には次のように機能追加をお願いします。Todoエンティティに priority フィールド を追加し、domain層のtodo.go、usecase層のtodo_usecase.go、handler層のtodo_handler.go すべてに適切な変更を行ってください。
```

また、もしGPT-4.1-nanoを使っていて上手く行かないときはGPT-4.1-miniを使うことも検討してみてください。



### 実験後のリセット

各実験後は、以下のコマンドで元の状態に戻せます。

```bash
git restore .
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

### 次章でやること

Chapter5でnebulaは複雑なファイル編集機能を獲得しました。
これでコーディングエージェントらしいと言える姿になったのではないでしょうか。
余談ですが、本章の複数ファイル生成の部分でGPT-4.1-nanoを使っていたら、ファイル生成がなかなか上手くいかず、
何度も一週間くらいプロンプトを練り直して、プロンプトの難しさとモデルの性能差を思い知りました...

次のChapter6ではもう少しだけ機能追加をしていきます。
複数ファイル編集は本Chapterで達成できているのですが、Planモードと以前の会話からスタートする記憶保持機能を実装していきます。
この2機能を実装し、本プロジェクトを締めくくりましょう！