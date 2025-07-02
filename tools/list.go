package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

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

// List は指定されたパス内のファイルとディレクトリをリストする
func List(args string) (string, error) {
	var listArgs ListArgs
	if err := json.Unmarshal([]byte(args), &listArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	var files []string

	if listArgs.Recursive {
		err := filepath.Walk(listArgs.Path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			files = append(files, path)
			return nil
		})
		if err != nil {
			result := ListResult{
				Files: []string{},
				Error: fmt.Sprintf("ディレクトリの読み込みに失敗しました: %v", err),
			}
			resultJSON, _ := json.Marshal(result)
			return string(resultJSON), nil
		}
	} else {
		entries, err := os.ReadDir(listArgs.Path)
		if err != nil {
			result := ListResult{
				Files: []string{},
				Error: fmt.Sprintf("ディレクトリの読み込みに失敗しました: %v", err),
			}
			resultJSON, _ := json.Marshal(result)
			return string(resultJSON), nil
		}

		for _, entry := range entries {
			files = append(files, filepath.Join(listArgs.Path, entry.Name()))
		}
	}

	result := ListResult{
		Files: files,
		Error: "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

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