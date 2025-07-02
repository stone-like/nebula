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

// SearchInDirectory は指定されたディレクトリ配下を再帰的に検索し、キーワードを含むファイルを見つける
func SearchInDirectory(args string) (string, error) {
	var searchArgs SearchInDirectoryArgs
	if err := json.Unmarshal([]byte(args), &searchArgs); err != nil {
		return "", fmt.Errorf("引数の解析に失敗しました: %v", err)
	}

	var matchingFiles []string

	err := filepath.Walk(searchArgs.Directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// ディレクトリはスキップ
		if info.IsDir() {
			return nil
		}

		// ファイルの内容を読み込み
		file, err := os.Open(path)
		if err != nil {
			// ファイルが開けない場合はスキップ
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), searchArgs.Keyword) {
				matchingFiles = append(matchingFiles, path)
				break // ファイル内で見つかったら次のファイルへ
			}
		}

		return nil
	})

	if err != nil {
		result := SearchInDirectoryResult{
			Files: []string{},
			Error: fmt.Sprintf("検索に失敗しました: %v", err),
		}
		resultJSON, _ := json.Marshal(result)
		return string(resultJSON), nil
	}

	result := SearchInDirectoryResult{
		Files: matchingFiles,
		Error: "",
	}
	resultJSON, _ := json.Marshal(result)
	return string(resultJSON), nil
}

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