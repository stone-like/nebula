# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `nebula`, a learning project for building an autonomous coding agent written in Go. The agent is designed to:

1. **Primary Goal**: Add new features to layered architecture projects (like Clean Architecture) with minimal instructions
2. **Secondary Goal**: Build small projects (CLI tools, web APIs) from scratch through interactive dialogue

The project follows a chapter-based development approach, currently completed through Chapter 3 (file system tools implementation).

## Common Development Commands

### Building and Running
```bash
# Build the project
go build -o nebula .           # Linux/macOS
go build -o nebula.exe .       # Windows

# Run the CLI
./nebula                       # Linux/macOS
nebula.exe                     # Windows

# Run with Go directly
go run .
```

### Environment Setup
```bash
# Required environment variable
export OPENAI_API_KEY=your_api_key_here

# Verify Go version (requires Go 1.23.1+)
go version
```

### Testing and Dependencies
```bash
# Install/update dependencies
go mod tidy

# View current dependencies
go list -m all

# Clean module cache if needed
go clean -modcache
```

## Architecture

### Current Implementation (Through Chapter 3)

The project implements a complete OpenAI chat CLI with Function Calling support:
- **main.go**: Core chat loop with OpenAI GPT-4.1-nano integration and tool orchestration
- **tools/** package: Modular tool system with complete file system operations
- **Environment-based API key management**
- **Conversation history handling**
- **User permission system** for destructive operations

### Implemented Tool System

The agent has full file system operation capabilities:
- `readFile`: Complete file content reading ✓
- `list`: Directory listing with recursive option ✓
- `searchInDirectory`: Recursive keyword search through file contents ✓
- `writeFile`: New file creation with auto-directory creation and user permission ✓

### Future Chapters
- `editFile`: Complete file overwrite (follows Read-Modify-Write pattern)
- Operating modes: `plan` mode (read-only) and `agent` mode (full operations)

## Code Structure and Patterns

### Modular Tool Architecture
The tools are organized in a separate `tools/` package:
- **`tools/common.go`**: Shared `ToolDefinition` struct and common types
- **`tools/registry.go`**: Central tool registration via `GetAvailableTools()`
- **Individual tool files**: Each tool in its own file (e.g., `readfile.go`, `list.go`)

### Tool Definition Pattern
```go
type ToolDefinition struct {
    Schema   openai.Tool
    Function func(args string) (string, error)
}
```

All tools follow this consistent pattern:
1. **Args struct**: JSON argument parsing with specific structs (e.g., `ReadFileArgs`)
2. **Business logic**: Core functionality implementation
3. **Result struct**: JSON result formatting (e.g., `ReadFileResult`)
4. **Schema definition**: OpenAI Function schema with jsonschema validation
5. **Tool getter**: Function like `GetReadFileTool()` returning `ToolDefinition`

### User Safety Features
- **Permission prompts**: `writeFile` requires explicit user confirmation (`y/N`)
- **File existence checks**: Prevents accidental overwrites
- **Auto-directory creation**: `writeFile` creates parent directories automatically
- **Graceful error handling**: Tools return JSON error responses rather than crashing

### File Editing Pattern (Future Implementation)
All file edits MUST follow the "Read-Modify-Write" pattern:
1. Use `readFile` to get complete current content
2. Construct complete new file version mentally  
3. Use `editFile` with entire new content (never partial edits)

## Project Structure

- `main.go`: CLI application entry point with OpenAI integration and tool orchestration
- `tools/`: Modular tool package with individual tool implementations
  - `common.go`: Shared types and definitions
  - `registry.go`: Tool registration and management
  - `readfile.go`, `list.go`, `search.go`, `writefile.go`: Individual tool implementations
- `spec.md`: Complete technical specification in Japanese
- `tasks.md`: Chapter-based learning curriculum with progress tracking
- `book/`: Chapter-by-chapter tutorial content (Zenn book format)
- `go.mod`/`go.sum`: Go module definition and dependencies

## Tool Testing Commands

Since the CLI requires OpenAI API key, test tools using natural language commands:

### List Tool Testing
```
# Basic directory listing
"現在のディレクトリのファイル一覧を表示してください"

# Recursive listing  
"toolsディレクトリを再帰的にリストしてください"
```

### Search Tool Testing
```
# Keyword search
"プロジェクト内で 'OpenAI' という文字列を含むファイルを探してください"

# Function search
"'GetAvailableTools' という関数が定義されているファイルを見つけてください"
```

### WriteFile Tool Testing
```
# Simple file creation (with user prompt)
"'hello.txt' というファイルを作成して、内容は 'Hello, World!' にしてください"

# Directory creation
"'test/example.go' というファイルを作成して、シンプルなHello Worldプログラムを書いてください"
```

## Key Implementation Notes

- Uses `github.com/sashabaranov/go-openai` for OpenAI API integration
- Conversation history maintained in memory during session
- All tools use consistent JSON input/output format for Function Calling
- Modular architecture allows easy addition of new tools
- User safety built-in for destructive operations (`writeFile` permission prompts)
- Error boundaries prevent tool failures from crashing the main program