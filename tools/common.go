package tools

import (
	"github.com/sashabaranov/go-openai"
)

// ToolDefinition はLLMが呼び出せるツールを表す構造体
type ToolDefinition struct {
	Schema   openai.Tool
	Function func(args string) (string, error)
}