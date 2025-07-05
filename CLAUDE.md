# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `nebula`, a learning project for building an autonomous coding agent written in Go. The agent is designed to:

1. **Primary Goal**: Add new features to layered architecture projects (like Clean Architecture) with minimal instructions
2. **Secondary Goal**: Build small projects (CLI tools, web APIs) from scratch through interactive dialogue

The project follows a chapter-based development approach, currently completed through Chapter 5 (Gemini CLI-inspired system prompt with strict execution protocols and safe JSON processing).

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

# Model switching commands (interactive)
./nebula
# Type "model" to switch between:
# 1. gpt-4.1-nano (default, faster)
# 2. gpt-4.1-mini (complex tasks)
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

### Current Implementation (Through Chapter 5)

The project implements a complete OpenAI chat CLI with Function Calling support and strict execution protocols:
- **main.go**: Core chat loop with OpenAI GPT-4.1-nano integration, tool orchestration with loop-based tool calling, and Gemini CLI-inspired system prompt
- **tools/** package: Complete modular tool system with all basic file operations, safety features, and safe JSON processing
- **Environment-based API key management**
- **Conversation history handling**
- **User permission system** for destructive operations (writeFile and editFile)
- **Read-Modify-Write pattern** enforcement for safe file editing through detailed tool descriptions
- **Strict execution protocol** preventing assumptions and enforcing information gathering before implementation

### Implemented Tool System

The agent has complete file system operation capabilities with enhanced safety:
- `readFile`: Complete file content reading with UTF-8 validation ✓
- `list`: Directory listing with recursive option ✓
- `searchInDirectory`: Recursive keyword search through file contents ✓
- `writeFile`: New file creation with auto-directory creation, user permission, and UTF-8 validation ✓
- `editFile`: Complete file overwrite with Read-Modify-Write pattern, user permission, and UTF-8 validation ✓
- **Safe JSON processing**: All tools use SafeJSONMarshal/SafeJSONUnmarshal to prevent character encoding issues

### System Prompt Architecture (Chapter 5)

The agent operates under a strict Gemini CLI-inspired system prompt that enforces:

**Critical Rules (Non-Negotiable):**
1. Never assume or guess file contents - must read files to understand them
2. Implementation without prior file analysis is technically impossible and prohibited
3. Before using writeFile or editFile, must have used readFile on reference files

**Execution Protocol:**
- **Phase 1: Information Gathering (REQUIRED)** - Use list, readFile, and searchInDirectory to understand structure
- **Phase 2: Implementation (Only after Phase 1)** - Use writeFile/editFile to make changes
- **Self-Verification Checklist** - Ensures all required information has been gathered

### Current Development Status (Updated)
- **Chapter 5**: Complete with model selection and configuration management
- **Chapter 6 (Planned)**: Final features including `plan` mode and persistent memory

## Code Structure and Patterns

### Modular Tool Architecture
The tools are organized in a separate `tools/` package:
- **`tools/common.go`**: Shared `ToolDefinition` struct and common types
- **`tools/registry.go`**: Central tool registration via `GetAvailableTools()`
- **`tools/json_helpers.go`**: Safe JSON processing utilities with UTF-8 validation and control character filtering
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
2. **Safe JSON processing**: Use `SafeJSONUnmarshal` for input and `SafeJSONMarshal` for output
3. **UTF-8 validation**: Validate all file content for proper encoding
4. **Business logic**: Core functionality implementation
5. **Result struct**: JSON result formatting (e.g., `ReadFileResult`)
6. **Schema definition**: OpenAI Function schema with jsonschema validation
7. **Tool getter**: Function like `GetReadFileTool()` returning `ToolDefinition`

### User Safety Features
- **Permission prompts**: Both `writeFile` and `editFile` require explicit user confirmation (`y/N`)
- **File existence checks**: `writeFile` prevents overwrites, `editFile` requires existing files
- **Auto-directory creation**: `writeFile` creates parent directories automatically
- **UTF-8 validation**: All file content is validated for proper encoding before processing
- **Safe JSON processing**: Control characters are automatically filtered from JSON input/output
- **Graceful error handling**: Tools return JSON error responses rather than crashing

### Tool Calling Loop Implementation
The core innovation in Chapter 4 is the **continuous tool execution loop** in `main.go:handleConversation`:
- **Problem solved**: Originally, `readFile` would execute but subsequent `editFile` calls would not execute
- **Solution**: Loop-based API calling that continues until LLM provides final text response
- **Key mechanism**: Each tool execution result triggers another API call, enabling complex multi-tool workflows

### File Editing Pattern (Implemented)
All file edits MUST follow the "Read-Modify-Write" pattern enforced by detailed tool descriptions:
1. Use `readFile` to get complete current content
2. Construct complete new file version mentally  
3. Use `editFile` with entire new content (never partial edits)
4. Tool Description explicitly instructs LLM to follow this workflow

## Project Structure

- `main.go`: CLI application entry point with OpenAI integration, tool orchestration, and system prompt
- `config/`: Configuration management package
  - `config.go`: Model selection and settings file management
- `tools/`: Modular tool package with individual tool implementations
  - `common.go`: Shared types and definitions
  - `registry.go`: Tool registration and management  
  - `json_helpers.go`: Safe JSON processing utilities with UTF-8 validation
  - `readfile.go`, `list.go`, `search.go`, `writefile.go`, `editfile.go`: Individual tool implementations
- `test/`: Test projects for validating agent capabilities
  - `todo-app/`: Clean Architecture example project for testing multi-file editing
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

### EditFile Tool Testing
```
# Read-Modify-Write pattern testing
"sample.txtの内容を 'Hello, Nebula!' に変更してください"

# Complex editing with content preservation
"sample.txtに新しい行を追加して、元の内容も残してください"
```

### Multi-File Architecture Testing (todo-app)
```bash
# Setup test environment
cp -r test/todo-app ./practice-todo-app
cd practice-todo-app
git init && git add . && git commit -m "Initial state"

# Test complex multi-file editing with natural language
"Goで書かれている本プロジェクトのTODOアプリに優先度機能を追加してください。具体的には次のように機能追加をお願いします。Todoエンティティに priority フィールド を追加し、domain層のtodo.go、usecase層のtodo_usecase.go、handler層のtodo_handler.go すべてに適切な変更を行ってください。"

# Reset for next test
git reset --hard HEAD~1 && git clean -fd
```

## Key Implementation Notes

- Uses `github.com/sashabaranov/go-openai` for OpenAI API integration
- Conversation history maintained in memory during session
- All tools use consistent JSON input/output format for Function Calling with safe processing
- **Gemini CLI-inspired system prompt** enforces strict execution protocols and prevents assumptions
- **Safe JSON processing** via `SafeJSONMarshal`/`SafeJSONUnmarshal` prevents character encoding issues
- **UTF-8 validation** ensures all file content is properly encoded
- Modular architecture allows easy addition of new tools
- User safety built-in for destructive operations (`writeFile` and `editFile` permission prompts)
- Error boundaries prevent tool failures from crashing the main program
- **Critical Feature**: Loop-based tool calling enables complex multi-step workflows like Read-Modify-Write
- **Tool Description Guidance**: Detailed descriptions in tool schemas guide LLM behavior for safe file operations
- **Phase-based execution**: Information gathering phase must complete before implementation phase

## Chapter Progress and Learning Path

### Completed (Chapter 1-5)
- ✅ OpenAI API integration and basic chat functionality
- ✅ Function Calling implementation with complete tool set
- ✅ Tool Calling loops for continuous multi-tool execution
- ✅ Read-Modify-Write pattern for safe file editing
- ✅ User permission system for destructive operations
- ✅ **Chapter 5**: Complete with model selection (gpt-4.1-nano/mini) and configuration file management (~/.nebula/config.json)

### Resolved Issues (Chapter 5)
- ✅ **editFile character encoding problem**: RESOLVED - Added UTF-8 validation and safe JSON processing to prevent control characters and invalid Unicode
- ✅ **JSON escaping issues**: RESOLVED - Implemented SafeJSONMarshal/SafeJSONUnmarshal with proper encoding validation and control character filtering
- ✅ **LLM assumption behavior**: RESOLVED - Gemini CLI-inspired system prompt prevents guessing and enforces information gathering
- ✅ **Inconsistent execution patterns**: RESOLVED - Strict phase-based execution protocol ensures consistent behavior

### Planned Features (Chapter 6)
- **Model switching**: GPT-4.1-nano (default) and GPT-4.1-mini (complex tasks)
- **Plan mode**: Read-only mode for safe planning before execution
- **Persistent memory**: Session storage for conversation history and context
- **Configuration management**: File-based settings beyond environment variables

### Chapter 5 Implementation Highlights

**System Prompt Implementation:**
- **Execution Protocol**: Strict phase-based approach (Information Gathering → Implementation)
- **Critical Rules**: Non-negotiable rules preventing assumptions and guessing
- **Self-Verification**: Built-in checklist system ensures thorough preparation
- **Forbidden Patterns**: Explicit prohibition of common LLM mistakes

**Technical Safety Enhancements:**
- **UTF-8 Validation**: `ValidateUTF8String` function prevents invalid character encoding
- **Safe JSON Processing**: `SafeJSONMarshal`/`SafeJSONUnmarshal` with control character filtering
- **Consistent Error Handling**: Enhanced error reporting across all tools
- **Tool Architecture**: All tools follow consistent safety patterns

**Testing and Validation:**
- **Multi-file editing**: Successfully tested with Clean Architecture todo-app
- **Complex feature addition**: Priority system implementation across domain/usecase/handler layers
- **System prompt effectiveness**: Verified proper exploration → implementation flow
- **Architecture understanding**: Demonstrated ability to identify and modify all relevant files

## Development Approach

This codebase follows a **simplified, focused approach** for Chapter 6 completion:

### Current Priority (Chapter 5 → 6)
1. **Model switching functionality** for handling complex tasks with GPT-4.1-mini
2. **Basic configuration management** beyond environment variables
3. **Plan mode** for safe pre-execution planning
4. **Persistent memory** for session continuity

### Removed Complexity
- Removed: Complex `init` functionality and automatic `NEBULA.md` generation
- Removed: Extensive error handling and logging systems
- Focused: Simple, practical implementation for real-world use

### Test Projects
The `test/` directory contains validation projects:
- **`todo-app/`**: Clean Architecture validation with multi-layer editing
- **`task-cli/`**: Simple CLI for basic feature addition testing
- **`ecommerce-platform/`**: Microservices architecture for complex integration testing

This codebase serves as a learning project demonstrating the evolution from basic LLM interaction to sophisticated autonomous coding capabilities with practical, focused implementation.