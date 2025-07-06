# Chapter 6: 記憶機能とPlanモード

## はじめに

Chapter 5では、nebulaに思考プロセスとモデル選択機能を実装し、かなり実用的なコーディングエージェントが完成しました。

しかし、現在のnebulaには重要な制約があります。それは**「記憶」**です。現在の実装では、nebulaを終了すると、今まで行った会話の履歴がすべて失われてしまいます。また、危険な操作を実行する前に、計画を安全に確認する方法もありません。

この章では、これらの課題を解決し、nebulaをより実用的なエージェントにしていきましょう！

### この章で達成すること

1. **永続記憶機能**: SQLiteを使ってセッション間で会話履歴を保存・復元
2. **プロジェクト固有記憶**: ディレクトリごとに独立したセッション管理
3. **Planモード**: 実行前の安全な計画確認（読み取り専用モード）
4. **動的モード切り替え**: 実行中にいつでもplanモードとagentモードを切り替え

これらの機能により、nebulaは「昨日の作業の続きから始める」ことや、「まず安全に計画を立ててから実行する」といった、実際の開発ワークフローに必要な機能を手に入れます。

## 📁 この章での到達目標構造

```
nebula/
├── main.go                 # セッション管理・モード切り替え統合
├── config/                 
│   └── config.go          # データベース設定追加
├── memory/                 # 新規パッケージ
│   ├── manager.go         # メモリマネージャー
│   ├── models.go          # Session・Message構造体
│   ├── database.go        # SQLite接続管理
│   └── queries.go         # SQL操作
├── tools/                  
│   ├── common.go          
│   ├── readfile.go        
│   ├── list.go            
│   ├── search.go          
│   ├── writefile.go       
│   ├── editfile.go        
│   └── registry.go        
├── go.mod                 # SQLite依存関係追加
└── go.sum                 
```

**前章からの変化:**
- Chapter 5: システムプロンプト + 設定管理
- Chapter 6: **永続記憶 + Planモード** ← 今ここ（最終章）

**実装する機能:**
- SQLite永続記憶システム
- プロジェクト固有セッション管理
- PLAN/AGENTモード動的切り替え
- セッション復元機能
- 会話履歴の完全保存

**新規依存関係:**
- `modernc.org/sqlite`: CGO不要のSQLite実装
- `database/sql`: 標準SQLインターフェース

**ユーザーディレクトリ:**
```
~/.nebula/
├── config.json            # 設定ファイル（DB設定追加）
└── memory.db              # SQLiteデータベース
```

**データベーススキーマ:**
```sql
sessions (id, started_at, ended_at, project_path, model_used)
messages (id, session_id, timestamp, role, content, tool_calls, tool_results)
```

## 記憶機能の設計と動作フロー

実装を始める前に、nebulaの記憶システムがどのように動作するかを理解しておきましょう。

### 記憶システムの全体像

nebulaの記憶システムは、以下の要素で構成されます。

```
記憶システム = セッション管理 + メッセージ保存 + プロジェクト分離
```

:::details 記憶システムの詳細構成

#### 1. セッション管理
- **セッション**: 一回の会話全体を表す単位
- **セッションID**: `session_20240315_143022`のような一意識別子
- **ライフサイクル**: 開始 → 進行中 → 終了

#### 2. メッセージ保存
- **役割別保存**: user、assistant、toolの各メッセージを区別
- **時系列保存**: 会話の流れを正確に復元するため時刻も記録
- **ツール情報**: 関数呼び出しと結果も保存（将来的な機能拡張用）

#### 3. プロジェクト分離
- **ディレクトリベース**: 作業ディレクトリごとに独立した記憶
- **文脈保持**: プロジェクトAの会話とプロジェクトBの会話が混在しない

#### 4. パフォーマンス最適化
- **インデックス**: project_path、session_id、timestampに最適化されたインデックス
- **軽量クエリ**: セッション一覧表示用の効率的なクエリ
- **リソース管理**: 適切なデータベース接続管理
:::


### 実際の動作フロー

#### nebulaの起動から終了まで

```
1. nebula起動
   ↓
2. 現在のディレクトリを確認
   ↓
3. 過去のセッションを検索・表示
   ↓
4. セッション選択（新規 or 復元）
   ↓
5. 会話開始
   ├─ ユーザーメッセージ → データベースに保存
   ├─ アシスタント応答 → データベースに保存
   └─ ツール実行 → 結果を記録
   ↓
6. 終了時にセッションを完了状態に更新
```

#### 記憶の活用パターン

**パターン1: 昨日の続きから**
```
月曜日: 「TODOアプリを作りたい」→ 基本機能を実装
火曜日: セッション復元 → 「優先度機能を追加して」→ 前回の文脈で継続
```

**パターン2: 機能別セッション管理**
```
セッション1: 認証機能の実装
セッション2: UI改善の実装
セッション3: バグ修正作業
→ 必要に応じて適切なセッションを復元
```


### データベース設計の思想

記憶システムは2つのテーブルで構成されます。

#### sessionsテーブル
- **目的**: セッション全体のメタデータ管理
- **キー情報**: いつ、どのプロジェクトで、どのモデルを使った会話か
- **活用**: セッション一覧表示、プロジェクト別フィルタリング

#### messagesテーブル
- **目的**: 会話内容の詳細保存
- **キー情報**: 誰が、いつ、何を言ったか
- **活用**: 会話履歴の完全復元

### 記憶システムの利点

1. **継続性**: 作業を中断しても文脈を失わない
2. **効率性**: 同じ説明を繰り返す必要がない
3. **学習効果**: 過去の試行錯誤を参照できる
4. **プロジェクト管理**: 複数プロジェクトの並行作業が可能

## 記憶機能・Planモード実装

それでは、この記憶システムを実際に実装していきましょう！

### Step 1: SQLite依存関係の追加

まず、永続記憶機能のためにSQLiteライブラリを追加しましょう。今回は、CGOに依存しない`modernc.org/sqlite`を使用します。

```bash
go get modernc.org/sqlite
```

このライブラリは、Go言語のみで実装されているため、クロスコンパイルが簡単で、外部のC言語ライブラリに依存しません。

### Step 2: memoryパッケージの作成

永続記憶機能を管理する専用パッケージを作成します。まずはディレクトリを作成します。

```bash
mkdir memory
```

#### models.go - データ構造の定義

まず、セッションとメッセージを表現するデータ構造を定義しましょう。

```go
// memory/models.go
package memory

import (
	"time"
)

// Session represents a conversation session
type Session struct {
	ID          string    `json:"id"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	ProjectPath string    `json:"project_path"`
	ModelUsed   string    `json:"model_used"`
}

// Message represents a single message in the conversation
type Message struct {
	ID          int       `json:"id"`
	SessionID   string    `json:"session_id"`
	Timestamp   time.Time `json:"timestamp"`
	Role        string    `json:"role"`        // 'user', 'assistant', 'tool'
	Content     string    `json:"content"`
	ToolCalls   *string   `json:"tool_calls,omitempty"`   // JSON string
	ToolResults *string   `json:"tool_results,omitempty"` // JSON string
}

// SessionSummary represents a brief summary of a session for listing
type SessionSummary struct {
	ID          string    `json:"id"`
	StartedAt   time.Time `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	ProjectPath string    `json:"project_path"`
	ModelUsed   string    `json:"model_used"`
	MessageCount int      `json:"message_count"`
	LastMessage  string   `json:"last_message"`
}

// IsActive returns true if the session is still active (not ended)
func (s *Session) IsActive() bool {
	return s.EndedAt == nil
}

// Duration returns the duration of the session
func (s *Session) Duration() time.Duration {
	if s.EndedAt == nil {
		return time.Since(s.StartedAt)
	}
	return s.EndedAt.Sub(s.StartedAt)
}
```

この構造では、`Session`でセッション全体を管理し、`Message`で個々のメッセージを管理します。`SessionSummary`は、セッション一覧表示用の軽量な構造体です。

#### database.go - SQLite接続の管理

次に、SQLiteデータベースの初期化と接続管理を実装します。

```go
// memory/database.go
package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Database handles SQLite database operations
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database instance
func NewDatabase(dbPath string) (*Database, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	database := &Database{db: db}

	// Initialize tables
	if err := database.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initTables creates the necessary tables if they don't exist
func (d *Database) initTables() error {
	// Create sessions table
	sessionTableSQL := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ended_at DATETIME,
		project_path TEXT NOT NULL,
		model_used TEXT NOT NULL
	);`

	if _, err := d.db.Exec(sessionTableSQL); err != nil {
		return fmt.Errorf("failed to create sessions table: %w", err)
	}

	// Create messages table
	messageTableSQL := `
	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT REFERENCES sessions(id),
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		role TEXT NOT NULL,
		content TEXT,
		tool_calls TEXT,
		tool_results TEXT
	);`

	if _, err := d.db.Exec(messageTableSQL); err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create indexes for better performance
	indexSQL := []string{
		"CREATE INDEX IF NOT EXISTS idx_sessions_project_path ON sessions(project_path);",
		"CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);",
		"CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);",
	}

	for _, sql := range indexSQL {
		if _, err := d.db.Exec(sql); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// GetDB returns the underlying database connection
func (d *Database) GetDB() *sql.DB {
	return d.db
}
```

この実装では、データベースファイルが存在しない場合は自動的に作成され、必要なテーブルとインデックスが初期化されます。

#### queries.go - SQL操作の実装

データベースのCRUD操作を実装します。

```go
// memory/queries.go
package memory

import (
	"database/sql"
	"fmt"
)

// CreateSession creates a new session in the database
func (d *Database) CreateSession(session *Session) error {
	query := `
		INSERT INTO sessions (id, started_at, project_path, model_used)
		VALUES (?, ?, ?, ?)
	`
	_, err := d.db.Exec(query, session.ID, session.StartedAt, session.ProjectPath, session.ModelUsed)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

// EndSession marks a session as ended
func (d *Database) EndSession(sessionID string) error {
	query := `UPDATE sessions SET ended_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := d.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}
	return nil
}

// GetSession retrieves a session by ID
func (d *Database) GetSession(sessionID string) (*Session, error) {
	query := `SELECT id, started_at, ended_at, project_path, model_used FROM sessions WHERE id = ?`
	row := d.db.QueryRow(query, sessionID)

	var session Session
	var endedAt sql.NullTime
	err := row.Scan(&session.ID, &session.StartedAt, &endedAt, &session.ProjectPath, &session.ModelUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if endedAt.Valid {
		session.EndedAt = &endedAt.Time
	}

	return &session, nil
}

// GetSessionsByProject retrieves sessions for a specific project path
func (d *Database) GetSessionsByProject(projectPath string, limit int) ([]*SessionSummary, error) {
	query := `
		SELECT s.id, s.started_at, s.ended_at, s.project_path, s.model_used,
			   COUNT(m.id) as message_count,
			   COALESCE(
				   (SELECT content FROM messages WHERE session_id = s.id ORDER BY timestamp DESC LIMIT 1),
				   ''
			   ) as last_message
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		WHERE s.project_path = ?
		GROUP BY s.id
		ORDER BY s.started_at DESC
		LIMIT ?
	`
	rows, err := d.db.Query(query, projectPath, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by project: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionSummary
	for rows.Next() {
		var summary SessionSummary
		var endedAt sql.NullTime
		err := rows.Scan(
			&summary.ID, &summary.StartedAt, &endedAt, &summary.ProjectPath,
			&summary.ModelUsed, &summary.MessageCount, &summary.LastMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session summary: %w", err)
		}

		if endedAt.Valid {
			summary.EndedAt = &endedAt.Time
		}

		sessions = append(sessions, &summary)
	}

	return sessions, nil
}

// SaveMessage saves a message to the database
func (d *Database) SaveMessage(message *Message) error {
	query := `
		INSERT INTO messages (session_id, timestamp, role, content, tool_calls, tool_results)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	result, err := d.db.Exec(query, message.SessionID, message.Timestamp, message.Role, message.Content, message.ToolCalls, message.ToolResults)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Get the inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}
	message.ID = int(id)

	return nil
}

// GetSessionMessages retrieves all messages for a session
func (d *Database) GetSessionMessages(sessionID string) ([]*Message, error) {
	query := `
		SELECT id, session_id, timestamp, role, content, tool_calls, tool_results
		FROM messages
		WHERE session_id = ?
		ORDER BY timestamp ASC
	`
	rows, err := d.db.Query(query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var message Message
		var toolCalls, toolResults sql.NullString
		err := rows.Scan(
			&message.ID, &message.SessionID, &message.Timestamp,
			&message.Role, &message.Content, &toolCalls, &toolResults,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if toolCalls.Valid {
			message.ToolCalls = &toolCalls.String
		}
		if toolResults.Valid {
			message.ToolResults = &toolResults.String
		}

		messages = append(messages, &message)
	}

	return messages, nil
}

// GetRecentSessions retrieves the most recent sessions across all projects
func (d *Database) GetRecentSessions(limit int) ([]*SessionSummary, error) {
	query := `
		SELECT s.id, s.started_at, s.ended_at, s.project_path, s.model_used,
			   COUNT(m.id) as message_count,
			   COALESCE(
				   (SELECT content FROM messages WHERE session_id = s.id ORDER BY timestamp DESC LIMIT 1),
				   ''
			   ) as last_message
		FROM sessions s
		LEFT JOIN messages m ON s.id = m.session_id
		GROUP BY s.id
		ORDER BY s.started_at DESC
		LIMIT ?
	`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionSummary
	for rows.Next() {
		var summary SessionSummary
		var endedAt sql.NullTime
		err := rows.Scan(
			&summary.ID, &summary.StartedAt, &endedAt, &summary.ProjectPath,
			&summary.ModelUsed, &summary.MessageCount, &summary.LastMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session summary: %w", err)
		}

		if endedAt.Valid {
			summary.EndedAt = &endedAt.Time
		}

		sessions = append(sessions, &summary)
	}

	return sessions, nil
}

// DeleteSession deletes a session and all its messages
func (d *Database) DeleteSession(sessionID string) error {
	tx, err := d.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete messages first
	if _, err := tx.Exec("DELETE FROM messages WHERE session_id = ?", sessionID); err != nil {
		return fmt.Errorf("failed to delete messages: %w", err)
	}

	// Delete session
	if _, err := tx.Exec("DELETE FROM sessions WHERE id = ?", sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
```

これらのクエリ関数により、セッションとメッセージの完全なCRUD操作が可能になります。

#### manager.go - メモリマネージャーの実装

最後に、これらのコンポーネントを統合するメモリマネージャーを実装します。

```go
// memory/manager.go
package memory

import (
	"fmt"
	"os"
	"time"
)

// Manager handles memory operations
type Manager struct {
	db             *Database
	currentSession *Session
}

// NewManager creates a new memory manager
func NewManager(dbPath string) (*Manager, error) {
	db, err := NewDatabase(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return &Manager{
		db: db,
	}, nil
}

// Close closes the memory manager
func (m *Manager) Close() error {
	// End current session if active
	if m.currentSession != nil && m.currentSession.IsActive() {
		if err := m.EndSession(); err != nil {
			return fmt.Errorf("failed to end current session: %w", err)
		}
	}

	return m.db.Close()
}

// StartSession creates a new session or restores an existing one
func (m *Manager) StartSession(projectPath, modelUsed string) (*Session, error) {
	// Generate session ID based on timestamp
	sessionID := fmt.Sprintf("session_%s", time.Now().Format("20060102_150405"))

	session := &Session{
		ID:          sessionID,
		StartedAt:   time.Now(),
		ProjectPath: projectPath,
		ModelUsed:   modelUsed,
	}

	if err := m.db.CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	m.currentSession = session
	return session, nil
}

// RestoreSession restores an existing session
func (m *Manager) RestoreSession(sessionID string) (*Session, error) {
	session, err := m.db.GetSession(sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	m.currentSession = session
	return session, nil
}

// EndSession ends the current session
func (m *Manager) EndSession() error {
	if m.currentSession == nil {
		return nil
	}

	if err := m.db.EndSession(m.currentSession.ID); err != nil {
		return fmt.Errorf("failed to end session: %w", err)
	}

	// Update local session
	now := time.Now()
	m.currentSession.EndedAt = &now
	m.currentSession = nil

	return nil
}

// GetCurrentSession returns the current session
func (m *Manager) GetCurrentSession() *Session {
	return m.currentSession
}

// SaveMessage saves a message to the current session
func (m *Manager) SaveMessage(role, content string, toolCalls, toolResults interface{}) error {
	if m.currentSession == nil {
		return nil
	}

	message := &Message{
		SessionID: m.currentSession.ID,
		Timestamp: time.Now(),
		Role:      role,
		Content:   content,
	}

	// Convert tool calls/results to JSON strings if provided
	if toolCalls != nil {
		if toolCallsJSON, ok := toolCalls.(string); ok {
			message.ToolCalls = &toolCallsJSON
		}
	}
	if toolResults != nil {
		if toolResultsJSON, ok := toolResults.(string); ok {
			message.ToolResults = &toolResultsJSON
		}
	}

	return m.db.SaveMessage(message)
}

// GetSessionsByProject returns sessions for the current project
func (m *Manager) GetSessionsByProject(projectPath string, limit int) ([]*SessionSummary, error) {
	return m.db.GetSessionsByProject(projectPath, limit)
}

// GetCurrentProjectSessions returns sessions for the current working directory
func (m *Manager) GetCurrentProjectSessions(limit int) ([]*SessionSummary, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	return m.GetSessionsByProject(currentDir, limit)
}

// GetSessionMessages returns all messages for a session
func (m *Manager) GetSessionMessages(sessionID string) ([]*Message, error) {
	return m.db.GetSessionMessages(sessionID)
}

// GetRecentSessions returns recent sessions across all projects
func (m *Manager) GetRecentSessions(limit int) ([]*SessionSummary, error) {
	return m.db.GetRecentSessions(limit)
}

// DeleteSession deletes a session and all its messages
func (m *Manager) DeleteSession(sessionID string) error {
	// If deleting current session, clear it
	if m.currentSession != nil && m.currentSession.ID == sessionID {
		m.currentSession = nil
	}

	return m.db.DeleteSession(sessionID)
}
```

これで、メモリパッケージの基本実装が完了しました！

### Step 3: 設定ファイルの拡張

永続記憶機能を設定で管理できるように、`config/config.go`を拡張しましょう。

```go
// config/config.go
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
	Model        string `json:"model"`
	DatabasePath string `json:"database_path"`
	MaxSessions  int    `json:"max_sessions"`
	APIKey       string `json:"-"` // APIキーは設定ファイルに保存しない
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	defaultDBPath := filepath.Join(homeDir, ".nebula", "memory.db")
	
	return &Config{
		Model:        "gpt-4.1-nano", // デフォルトはgpt-4.1-nano
		DatabasePath: defaultDBPath,
		MaxSessions:  100,
	}
}

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

// getConfigPath returns the path to the configuration file
func getConfigPath() string {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// フォールバック: カレントディレクトリに.nebulaフォルダを作成
		return ".nebula/config.json"
	}
	return filepath.Join(homeDir, ".nebula", "config.json")
}

// GetOpenAIModel returns the appropriate OpenAI model identifier
func (c *Config) GetOpenAIModel() string {
	switch c.Model {
	case "gpt-4.1-nano":
		return openai.GPT4Dot1Nano // OpenAIライブラリでの実際の識別子
	case "gpt-4.1-mini":
		return openai.GPT4Dot1Mini // 現在は同じモデルを使用（将来的に変更可能）
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

設定ファイルがシンプルになり、メモリ機能は常時有効になりました。

## 設定ファイルの移行について

Chapter 6では、設定ファイルに新しいデータベース関連フィールドが追加されました。

:::message alert
**重要: 設定ファイルの移行が必要**

既存のChapter 5からアップグレードする場合、記憶機能を有効にするために設定ファイルの移行が必要です。

### config.json構造の変化

**Chapter 5の設定ファイル:**
```json
{
  "model": "gpt-4.1-nano"
}
```

**Chapter 6の設定ファイル:**
```json
{
  "model": "gpt-4.1-nano",
  "database_path": "/home/user/.nebula/memory.db",
  "max_sessions": 100
}
```

### 移行手順

:::message alert
⚠️ **重要：設定ファイルの互換性について**
Chapter 5からChapter 6への移行では設定ファイル形式が変更されています。既存の設定ファイルを削除して新しい形式で再作成する必要があります。モデル設定は再度実行後に`model`コマンドで設定し直してください。
:::


**Linux/macOS の場合:**
```bash
# 既存設定を削除
rm ~/.nebula/config.json

# nebula起動時に新しい形式で自動作成
./nebula
```

**Windows の場合:**
```cmd
# 既存設定を削除
del "%USERPROFILE%\.nebula\config.json"

# nebula起動時に新しい形式で自動作成
nebula.exe
```

**注意**: この操作により、モデル設定は初期値（gpt-4.1-nano）にリセットされますが、`model`コマンドですぐに再設定できます。
:::


### Step 4: main.goの大幅拡張

いよいよ、メイン処理に永続記憶機能とPlanモードを統合します。まず、必要なimportを追加します。

```go
// main.go (importセクション)
import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"nebula/config"
	"nebula/memory"
	"nebula/tools"

	"github.com/sashabaranov/go-openai"
)
```

#### Planモード対応のツール実行関数

ツール実行関数を拡張して、Planモードでの制限を追加します。

```go
// executeToolCall は単一のツールコールを実行する
func executeToolCall(toolCall openai.ToolCall, toolsMap map[string]tools.ToolDefinition, planMode bool) openai.ChatCompletionMessage {
	if tool, exists := toolsMap[toolCall.Function.Name]; exists {
		// planモードでは書き込み系ツールの実行を制限
		if planMode && (toolCall.Function.Name == "writeFile" || toolCall.Function.Name == "editFile") {
			result := fmt.Sprintf(`{"error": "Tool '%s' is not allowed in plan mode. Plan mode is read-only."}`, toolCall.Function.Name)
			fmt.Printf("Plan mode: Blocked execution of '%s'\n", toolCall.Function.Name)
			return openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			}
		}

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
func processToolCalls(toolCalls []openai.ToolCall, toolsMap map[string]tools.ToolDefinition, planMode bool) []openai.ChatCompletionMessage {
	var toolMessages []openai.ChatCompletionMessage

	for _, toolCall := range toolCalls {
		toolMessage := executeToolCall(toolCall, toolsMap, planMode)
		toolMessages = append(toolMessages, toolMessage)
	}

	return toolMessages
}
```

#### 会話履歴の変換関数

メモリから復元したメッセージをOpenAI形式に変換する関数を追加します。

```go
// convertToOpenAIMessages converts memory messages to OpenAI format
func convertToOpenAIMessages(memoryMessages []*memory.Message) []openai.ChatCompletionMessage {
	var messages []openai.ChatCompletionMessage
	
	for _, msg := range memoryMessages {
		// Skip tool messages for now (they are complex to restore properly)
		if msg.Role == "tool" {
			continue
		}
		
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}
	
	return messages
}
```

#### 対話処理関数の拡張

対話処理関数にメモリ管理とPlanモード対応を追加します。

```go
// handleConversation はLLMとの対話セッションを処理する
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

	// メモリに保存
	memoryManager.SaveMessage("user", userInput, nil, nil)

	// 最初のAPI呼び出し
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    cfg.GetOpenAIModel(),
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
			toolMessages := processToolCalls(responseMessage.ToolCalls, toolsMap, planMode)
			messages = append(messages, toolMessages...)

			// 次のAPI呼び出し
			resp, err = client.CreateChatCompletion(
				context.Background(),
				openai.ChatCompletionRequest{
					Model:    cfg.GetOpenAIModel(),
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
			
			// アシスタントメッセージをメモリに保存
			memoryManager.SaveMessage("assistant", responseMessage.Content, nil, nil)
			break
		}
	}

	return messages
}
```

#### モード切り替え関数

動的なモード切り替えを可能にする関数を追加します。

```go
// handleModeSwitch handles interactive mode switching
func handleModeSwitch(planMode *bool) {
	currentMode := "AGENT"
	if *planMode {
		currentMode = "PLAN"
	}
	
	fmt.Printf("Current mode: %s\n", currentMode)
	fmt.Println("Available modes:")
	fmt.Println("1. AGENT (full capabilities)")
	fmt.Println("2. PLAN (read-only, safe exploration)")
	fmt.Print("Select mode (1 or 2): ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())
		
		switch choice {
		case "1":
			*planMode = false
			fmt.Println("Mode switched to: AGENT")
		case "2":
			*planMode = true
			fmt.Println("Mode switched to: PLAN")
		default:
			fmt.Println("Invalid choice. No changes made.")
		}
	}
}
```

#### メイン関数の全面改修

最後に、メイン関数にすべての機能を統合します。

```go
func main() {
	// デフォルトはagentモード
	planMode := false

	// 設定を読み込み
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// APIキーが設定されているかチェック
	if cfg.APIKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable is not set")
		fmt.Println("Please set your OpenAI API key: export OPENAI_API_KEY=your_api_key_here")
		os.Exit(1)
	}

	// メモリマネージャーを初期化
	memoryManager, err := memory.NewManager(cfg.DatabasePath)
	if err != nil {
		fmt.Printf("Error initializing memory: %v\n", err)
		os.Exit(1)
	}
	defer memoryManager.Close()

	// プロジェクトディレクトリを取得
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// セッション管理
	var messages []openai.ChatCompletionMessage
	var currentSession *memory.Session
	
	// 既存セッションを表示
	sessions, err := memoryManager.GetCurrentProjectSessions(5)
	if err != nil {
		fmt.Printf("Error loading sessions: %v\n", err)
	} else if len(sessions) > 0 {
		fmt.Printf("Found %d previous sessions for this project:\n", len(sessions))
		for i, session := range sessions {
			status := "completed"
			if session.EndedAt == nil {
				status = "active"
			}
			lastMsg := session.LastMessage
			if len(lastMsg) > 50 {
				lastMsg = lastMsg[:50] + "..."
			}
			fmt.Printf("%d. %s (%s) - %s\n", i+1, session.ID, status, lastMsg)
		}
		fmt.Print("Start new session or restore (new/1-5): ")
		
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			choice := strings.TrimSpace(scanner.Text())
			
			if choice != "new" && choice != "" {
				// セッション番号をパース
				if sessionIndex, err := strconv.Atoi(choice); err == nil {
					if sessionIndex >= 1 && sessionIndex <= len(sessions) {
						selectedSession := sessions[sessionIndex-1]
						
						// セッションを復元
						restoredSession, err := memoryManager.RestoreSession(selectedSession.ID)
						if err != nil {
							fmt.Printf("Error restoring session: %v\n", err)
						} else {
							currentSession = restoredSession
							fmt.Printf("Restored session: %s\n", restoredSession.ID)
							
							// 過去の会話履歴を読み込み
							memoryMessages, err := memoryManager.GetSessionMessages(selectedSession.ID)
							if err != nil {
								fmt.Printf("Error loading session messages: %v\n", err)
							} else {
								// OpenAI形式に変換
								messages = convertToOpenAIMessages(memoryMessages)
								fmt.Printf("Loaded %d previous messages\n", len(messages))
							}
						}
					} else {
						fmt.Println("Invalid session number. Starting new session.")
					}
				} else {
					fmt.Println("Invalid input. Starting new session.")
				}
			}
		}
	}
	
	// 新しいセッションを開始（復元しなかった場合）
	if currentSession == nil {
		session, err := memoryManager.StartSession(currentDir, cfg.Model)
		if err != nil {
			fmt.Printf("Error starting session: %v\n", err)
		} else {
			currentSession = session
			fmt.Printf("Started new session: %s\n", session.ID)
		}
	}

	// OpenAIクライアントを初期化
	client := openai.NewClient(cfg.APIKey)

	// 利用可能なツールを取得
	toolsMap := tools.GetAvailableTools()

	// ツールのスキーマを配列に変換
	var toolSchemas []openai.Tool
	for _, tool := range toolsMap {
		toolSchemas = append(toolSchemas, tool.Schema)
	}

	fmt.Println("nebula - OpenAI Chat CLI with Function Calling")
	fmt.Printf("Current model: %s\n", cfg.Model)
	fmt.Println("Memory: enabled")
	fmt.Println("Mode: AGENT (full capabilities)")
	fmt.Println("Available tools: readFile, list, searchInDirectory, writeFile, editFile")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  'exit' or 'quit' - End the conversation")
	fmt.Println("  'model' - Switch between models")
	fmt.Println("  'mode' - Interactive mode switching")  
	fmt.Println("  'plan' - Switch to PLAN mode (read-only)")
	fmt.Println("  'agent' - Switch to AGENT mode (full capabilities)")
	fmt.Println("---")

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// 現在のモードを表示
		modeIndicator := "AGENT"
		if planMode {
			modeIndicator = "PLAN"
		}
		fmt.Printf("[%s] You: ", modeIndicator)
		if !scanner.Scan() {
			break
		}

		userInput := strings.TrimSpace(scanner.Text())

		// 終了コマンドをチェック
		if userInput == "exit" || userInput == "quit" {
			fmt.Println("Goodbye!")
			break
		}

		// モデル切り替えコマンドをチェック
		if userInput == "model" {
			handleModelSwitch(cfg)
			continue
		}

		// モード切り替えコマンドをチェック
		if userInput == "mode" {
			handleModeSwitch(&planMode)
			continue
		}

		// 簡単なモード切り替えコマンド
		if userInput == "plan" {
			planMode = true
			fmt.Println("Mode switched to: PLAN (read-only)")
			continue
		}
		if userInput == "agent" {
			planMode = false
			fmt.Println("Mode switched to: AGENT (full capabilities)")
			continue
		}

		if userInput == "" {
			continue
		}

		// 対話セッションを処理
		messages = handleConversation(client, cfg, memoryManager, toolSchemas, toolsMap, userInput, messages, planMode)
	}
}
```

### Step 5: 動作確認

これでこの章の機能が動くようになりました！実際に試してみましょう。

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

#### 基本的な動作確認

1. **新しいセッションの開始**：
   初回起動では、新しいセッションが自動的に作成されます。

2. **メッセージの永続化**：
   何かメッセージを送信し、nebulaを終了後、再起動してください。以前のセッションが表示されるはずです。

3. **セッションの復元**：
   表示されたセッション番号を選択すると、前回の会話履歴が復元されます。

4. **Planモードのテスト**：
   ```
   [AGENT] You: plan
   Mode switched to: PLAN (read-only)
   
   [PLAN] You: sample.txtというファイルを作成してください
   # writeFileがブロックされることを確認
   
   [PLAN] You: agent
   Mode switched to: AGENT (full capabilities)
   
   [AGENT] You: sample.txtというファイルを作成してください
   # 実際にファイルが作成される
   ```

#### 実際の開発ワークフローで試してみる

1. **計画フェーズ**：
   ```
   [AGENT] You: plan
   [PLAN] You: クイズを出題してくれるCLIをGoで作りたいのですが、仕様を考えてください。
   ```

2. **実行フェーズ**：
   ```
   [PLAN] You: agent
   [AGENT] You: それでは仕様を元にクイズCLIを作ってください。作成場所は現在のディレクトリ直下に quizフォルダを作ってください。
   ```

3. **セッション継続**：
   途中でnebulaを終了し、再起動してセッションを復元して作業を継続

## この章のまとめと次のステップ

おめでとうございます！この章では、nebulaに以下の強力な機能を追加しました。

### 達成したこと

1. **永続記憶機能**: SQLiteを使った会話履歴の永続保存
2. **プロジェクト固有記憶**: ディレクトリごとの独立したセッション管理
3. **セッション復元**: 過去の会話を完全に復元する機能
4. **Planモード**: 安全な計画確認のための読み取り専用モード
5. **動的モード切り替え**: 実行中のリアルタイムなモード変更
6. **改善されたUI**: モード表示と直感的なコマンド体系

### おわりに

ここまで長い間お疲れさまでした。
本章にてnebulaコーディングエージェント開発は終了です。
コーディングエージェントと聞くとこれってどんな風に動いているんだろうと思っていたのですが、
骨子はシステムプロンプトと、FunctionCallingでその二つさえ押さえておけ一応は形になりそうかも、と思い自作コーディングエージェントを作り始めることにしました。

ただ、システムプロンプトには手を焼かされて、全然上手く動作しなくて開発が頓挫しそうになりました...
(nebula開発前にgemini-cliが出てくれなかったらどうなっていたことか)


さて、nebulaですがまだまだ課題はあります。
- editツールの最適化
- write,editのdiff表示
- コンテキストとなるファイルやフォルダの指定
- 複雑なタスク自動分割機能
- RAGの導入
- OpenAI以外のProvider対応
- ローカルLLM対応
- CLIのリッチ化(ロゴ表示させたり等々)

これらはいずれおまけという形でやるかもしれませんが、あなた自身でやってみても面白いかもしれません。

再三になりますが、やはりこれらの課題、特にeditツール最適化はgemini-cliが参考になりますのでリンクを貼っておきます。

それではこれにてnebula開発は終わりとします。
本書があなたのコーディングエージェントの理解の一助になってくれれば幸いです。


:::message
**参考リンク**
- [Gemini CLIソースコード](https://github.com/google-gemini/gemini-cli)
:::


