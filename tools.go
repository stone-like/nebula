package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// ReadFileArgs はreadFileツールの引数を表す構造体
type ReadFileArgs struct {
	Path string `json:"path" description:"読み込むファイルのパス"`
}

// ReadFileResult はreadFileツールの結果を表す構造体
type ReadFileResult struct {
	Content string `json:"content"`
	Error   string `json:"error,omitempty"`
}

// ToolDefinition はLLMが呼び出せるツールを表す構造体
type ToolDefinition struct {
	Schema   openai.Tool
	Function func(args string) (string, error)
}

// ReadFile は指定されたパスのファイル内容を読み込む
func ReadFile(args string) (string, error) {
	var readFileArgs ReadFileArgs
	if err := json.Unmarshal([]byte(args), &readFileArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	file, err := os.Open(readFileArgs.Path)
	if err != nil {
		result := ReadFileResult{
			Content: "",
			Error:   fmt.Sprintf("ファイルを開けませんでした: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		result := ReadFileResult{
			Content: "",
			Error:   fmt.Sprintf("ファイルの読み込みに失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	result := ReadFileResult{
		Content: string(content),
		Error:   "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

// GetReadFileTool はreadFileツールの定義を返す
func GetReadFileTool() ToolDefinition {
	return ToolDefinition{
		Schema: openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "readFile",
				Description: "指定されたファイルの内容全体を読み込みます。",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "読み込むファイルのパス",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		Function: ReadFile,
	}
}

// GetAvailableTools は利用可能な全てのツールを返す
func GetAvailableTools() map[string]ToolDefinition {
	return map[string]ToolDefinition{
		"readFile": GetReadFileTool(),
	}
}