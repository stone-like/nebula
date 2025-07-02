package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

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

// WriteFile は指定されたパスに新しいファイルを作成する（ユーザー許可が必要）
func WriteFile(args string) (string, error) {
	var writeArgs WriteFileArgs
	if err := json.Unmarshal([]byte(args), &writeArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	// ファイルが既に存在するかチェック
	if _, err := os.Stat(writeArgs.Path); err == nil {
		result := WriteFileResult{
			Success: false,
			Error:   "ファイルが既に存在します。既存ファイルの編集にはeditFileを使用してください。",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// ユーザーに許可を求める
	fmt.Printf("\n新しいファイルを作成します: %s\n", writeArgs.Path)
	fmt.Print("実行してもよろしいですか？ (y/N): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		result := WriteFileResult{
			Success: false,
			Error:   "ユーザー入力の読み取りに失敗しました",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	userResponse := strings.TrimSpace(scanner.Text())
	if userResponse != "y" && userResponse != "Y" {
		result := WriteFileResult{
			Success: false,
			Error:   "ユーザーによってキャンセルされました",
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// 親ディレクトリを作成
	dir := filepath.Dir(writeArgs.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		result := WriteFileResult{
			Success: false,
			Error:   fmt.Sprintf("ディレクトリの作成に失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	// ファイルを作成
	file, err := os.Create(writeArgs.Path)
	if err != nil {
		result := WriteFileResult{
			Success: false,
			Error:   fmt.Sprintf("ファイルの作成に失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}
	defer file.Close()

	// 内容を書き込み
	if _, err := file.WriteString(writeArgs.Content); err != nil {
		result := WriteFileResult{
			Success: false,
			Error:   fmt.Sprintf("ファイルの書き込みに失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	result := WriteFileResult{
		Success: true,
		Error:   "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

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