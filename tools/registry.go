package tools

// GetAvailableTools は利用可能な全てのツールを返す
func GetAvailableTools() map[string]ToolDefinition {
	return map[string]ToolDefinition{
		"readFile":           GetReadFileTool(),
		"list":               GetListTool(),
		"searchInDirectory":  GetSearchInDirectoryTool(),
		"writeFile":          GetWriteFileTool(),
		"editFile":           GetEditFileTool(),
	}
}