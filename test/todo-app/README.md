# TODO App - Clean Architecture Example

This is a simple TODO application built with Clean Architecture principles in Go.

## Architecture Layers

### Domain Layer (`domain/`)
- **todo.go**: Core Todo entity with business rules
- **repository.go**: Repository interface defining data operations

### Use Case Layer (`usecase/`)
- **todo_usecase.go**: Business logic and application rules

### Handler Layer (`handler/`)
- **todo_handler.go**: HTTP handlers for REST API endpoints

### Repository Layer (`repository/`)
- **memory_repo.go**: In-memory implementation of TodoRepository

## Project Structure
```
todo-app/
├── domain/
│   ├── todo.go          # Todo entity
│   └── repository.go    # Repository interface
├── usecase/
│   └── todo_usecase.go  # Business logic
├── handler/
│   └── todo_handler.go  # HTTP handlers
├── repository/
│   └── memory_repo.go   # In-memory implementation
├── main.go              # Entry point
├── go.mod               # Module definition
└── README.md            # This file
```

## API Endpoints

- `GET /health` - Health check
- `POST /todos` - Create a new todo
- `GET /todos` - Get all todos
- `GET /todos/{id}` - Get a specific todo
- `PUT /todos/{id}` - Update a todo
- `PATCH /todos/{id}/complete` - Mark todo as completed
- `DELETE /todos/{id}` - Delete a todo

## How to Run

```bash
cd test/todo-app
go mod tidy
go run main.go
```

The server will start on port 8080.

## Example Usage

```bash
# Create a todo
curl -X POST http://localhost:8080/todos \
  -H "Content-Type: application/json" \
  -d '{"title":"Learn Go","description":"Study Go programming language"}'

# Get all todos
curl http://localhost:8080/todos

# Complete a todo
curl -X PATCH http://localhost:8080/todos/1/complete
```

## Test Scenarios for nebula Agent

This project is designed to test the nebula coding agent with various feature addition scenarios:

### Easy Level
- Add a "priority" field to todos (High/Medium/Low)
- Add created_by field for todo ownership

### Medium Level  
- Add todo categories/tags functionality
- Add due date functionality with validation

### Hard Level
- Add user authentication and user-specific todos
- Add todo sharing between users
- Add notification system for due dates