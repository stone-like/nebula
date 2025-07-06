# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `nebula`, a learning project for building an autonomous coding agent written in Go. The agent is designed to:

1. **Primary Goal**: Add new features to layered architecture projects (like Clean Architecture) with minimal instructions
2. **Secondary Goal**: Build small projects (CLI tools, web APIs) from scratch through interactive dialogue

The project follows a chapter-based development approach, **now fully completed through Chapter 6** with persistent memory, dynamic mode switching, and session restoration capabilities.

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

# Interactive commands
./nebula
# Type "model" to switch between:
# 1. gpt-4.1-nano (default, faster)
# 2. gpt-4.1-mini (complex tasks)
#
# Type "plan" to switch to read-only planning mode
# Type "agent" to switch to full execution mode
# Type "mode" for interactive mode switching
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

### Current Implementation (Completed through Chapter 6)

The project implements a complete autonomous coding agent with persistent memory, dynamic mode switching, and session restoration:
- **main.go**: Core chat loop with OpenAI GPT-4.1-nano integration, tool orchestration with loop-based tool calling, and Gemini CLI-inspired system prompt
- **tools/** package: Complete modular tool system with all basic file operations, safety features, and safe JSON processing
- **memory/** package: SQLite-based persistent memory system with session management and conversation history
- **config/** package: Configuration management with model selection and settings persistence
- **Dynamic mode switching**: Real-time switching between PLAN (read-only) and AGENT (full capabilities) modes
- **Session restoration**: Complete conversation history restoration across sessions
- **Project-specific memory**: Independent session management per project directory
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

### Current Development Status (Completed)
- **Chapter 5**: Complete with model selection and configuration management
- **Chapter 6**: ✅ **COMPLETED** - Persistent memory, dynamic mode switching, and session restoration

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

- `main.go`: CLI application entry point with OpenAI integration, tool orchestration, system prompt, and session management
- `config/`: Configuration management package
  - `config.go`: Model selection, database path, and settings file management
- `memory/`: Persistent memory package (Chapter 6)
  - `manager.go`: Memory manager with session lifecycle management
  - `models.go`: Session and Message data structures
  - `database.go`: SQLite database initialization and connection management
  - `queries.go`: SQL operations for sessions and messages
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
- `go.mod`/`go.sum`: Go module definition and dependencies (includes modernc.org/sqlite)

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

### Completed (Chapter 1-6) ✅
- ✅ OpenAI API integration and basic chat functionality
- ✅ Function Calling implementation with complete tool set
- ✅ Tool Calling loops for continuous multi-tool execution
- ✅ Read-Modify-Write pattern for safe file editing
- ✅ User permission system for destructive operations
- ✅ **Chapter 5**: Model selection (gpt-4.1-nano/mini) and configuration file management (~/.nebula/config.json)
- ✅ **Chapter 6**: Persistent memory, dynamic mode switching, and session restoration

### Resolved Issues (Chapter 5)
- ✅ **editFile character encoding problem**: RESOLVED - Added UTF-8 validation and safe JSON processing to prevent control characters and invalid Unicode
- ✅ **JSON escaping issues**: RESOLVED - Implemented SafeJSONMarshal/SafeJSONUnmarshal with proper encoding validation and control character filtering
- ✅ **LLM assumption behavior**: RESOLVED - Gemini CLI-inspired system prompt prevents guessing and enforces information gathering
- ✅ **Inconsistent execution patterns**: RESOLVED - Strict phase-based execution protocol ensures consistent behavior

### Chapter 6 Implementation Highlights (Completed) ✅

**Persistent Memory System:**
- **SQLite Database**: `modernc.org/sqlite` for cross-platform compatibility without CGO dependencies
- **Session Management**: Project-specific session storage with automatic session lifecycle management
- **Message Persistence**: Complete conversation history with user, assistant, and tool messages
- **Session Restoration**: Full conversation context restoration across sessions

**Dynamic Mode Switching:**
- **PLAN Mode**: Read-only mode that blocks writeFile/editFile for safe exploration and planning
- **AGENT Mode**: Full capabilities mode with all tool access
- **Real-time Switching**: `plan`/`agent` commands for instant mode changes during conversation
- **Interactive Mode Selection**: `mode` command for guided mode switching

**Enhanced User Experience:**
- **Session Selection**: Interactive session restoration with preview of recent conversations
- **Mode Indicators**: Clear visual indicators showing current mode in prompts `[AGENT]` / `[PLAN]`
- **Memory Status**: Always-on memory with simplified configuration (no enable/disable complexity)
- **Conversation Continuity**: Seamless continuation of interrupted work sessions

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

This codebase follows a **simplified, focused approach** and has completed all planned features:

### Completed Features (Chapter 6) ✅
1. ✅ **Model switching functionality** for handling complex tasks with GPT-4.1-mini
2. ✅ **Configuration management** with simplified settings structure
3. ✅ **Plan mode** for safe pre-execution planning with dynamic switching
4. ✅ **Persistent memory** for session continuity with SQLite backend

### Removed Complexity
- Removed: Complex `init` functionality and automatic `NEBULA.md` generation
- Removed: Extensive error handling and logging systems
- Focused: Simple, practical implementation for real-world use

### Test Projects
The `test/` directory contains validation projects:
- **`todo-app/`**: Clean Architecture validation with multi-layer editing
- **`task-cli/`**: Simple CLI for basic feature addition testing
- **`ecommerce-platform/`**: Microservices architecture for complex integration testing

## Final Status

**nebula** is now a **complete autonomous coding agent** that has successfully evolved from basic LLM interaction to sophisticated coding capabilities. The project demonstrates:

### Key Achievements ✅
- **Full Tool Integration**: Complete file system operations with safety features
- **Persistent Memory**: SQLite-based session management across project directories
- **Dynamic Workflows**: Real-time switching between planning and execution modes
- **Production Ready**: Practical implementation suitable for real development workflows
- **Learning Foundation**: Complete chapter-based progression showing agent development principles

### Ideal Use Cases
1. **Adding features to existing projects**: Clean Architecture, microservices, or any structured codebase
2. **Iterative development**: Plan with PLAN mode, execute with AGENT mode
3. **Learning and experimentation**: Safe exploration of codebases before making changes
4. **Continuation of interrupted work**: Session restoration maintains full context across sessions

**nebula** represents a successful implementation of an autonomous coding agent that balances power with safety, providing both educational value and practical utility for software development tasks.