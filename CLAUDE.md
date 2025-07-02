# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is `nebula`, a learning project for building an autonomous coding agent written in Go. The agent is designed to:

1. **Primary Goal**: Add new features to layered architecture projects (like Clean Architecture) with minimal instructions
2. **Secondary Goal**: Build small projects (CLI tools, web APIs) from scratch through interactive dialogue

The project follows a chapter-based development approach, currently on Chapter 1 (basic OpenAI integration).

## Architecture

### Core Components

- **Language**: Go (agent implementation)
- **LLM Integration**: OpenAI GPT series with Tool Calling support
- **Operating Modes**:
  - `plan` mode: Read-only analysis and planning
  - `agent` mode: Full file system operations

### Tool System

The agent uses Go-implemented tools for file system operations:

- `list`: Directory listing (recursive option)
- `readFile`: Complete file content reading
- `writeFile`: New file creation with auto-directory creation
- `editFile`: Complete file overwrite (follows Read-Modify-Write pattern)
- `searchInDirectory`: Recursive keyword search

### Context Management

- **Existing Projects**: Uses `NEBULA.md` for project context or auto-generates via internal `init`
- **New Projects**: Starts with empty context, builds through user dialogue
- **Memory**: Session-based conversation history

## Development Workflow

### File Editing Pattern

All file edits MUST follow the "Read-Modify-Write" pattern:
1. Use `readFile` to get complete current content
2. Construct complete new file version mentally
3. Use `editFile` with entire new content (never partial edits)

### Execution Flow

1. **Thought**: Analyze context and define current task
2. **Plan**: Create step-by-step approach
3. **Exploration Phase**: Gather all needed information (NO file operations)
4. **Implementation Phase**: Apply changes using tools
5. **Report**: Communicate results to user

## Project Structure

- `spec.md`: Complete technical specification in Japanese
- `tasks.md`: Chapter-based learning curriculum
- Currently implementing Chapter 1: Basic OpenAI CLI integration

## Branch Strategy

- `master`: Main development branch
- `chapter-1`: Current working branch for OpenAI integration chapter

## Key Implementation Notes

- Agent follows layered prompt architecture (system prompt + dynamic context)
- Context initialization varies based on project state (new vs existing)
- Tool schemas are defined in Go structs for OpenAI Function Calling
- Emphasis on safe, complete file operations over partial edits