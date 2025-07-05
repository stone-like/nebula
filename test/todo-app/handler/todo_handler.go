package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"todo-app/usecase"

	"github.com/gorilla/mux"
)

// TodoHandler handles HTTP requests for todo operations
type TodoHandler struct {
	usecase *usecase.TodoUsecase
}

// NewTodoHandler creates a new TodoHandler
func NewTodoHandler(uc *usecase.TodoUsecase) *TodoHandler {
	return &TodoHandler{
		usecase: uc,
	}
}

// CreateTodoRequest represents the request body for creating a todo
type CreateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// UpdateTodoRequest represents the request body for updating a todo
type UpdateTodoRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// CreateTodo handles POST /todos
func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	todo, err := h.usecase.CreateTodo(req.Title, req.Description)
	if err != nil {
		if err == usecase.ErrInvalidInput {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(todo)
}

// GetTodos handles GET /todos
func (h *TodoHandler) GetTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := h.usecase.GetAllTodos()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todos)
}

// GetTodo handles GET /todos/{id}
func (h *TodoHandler) GetTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	todo, err := h.usecase.GetTodo(id)
	if err != nil {
		if err == usecase.ErrTodoNotFound {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

// UpdateTodo handles PUT /todos/{id}
func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	todo, err := h.usecase.UpdateTodo(id, req.Title, req.Description)
	if err != nil {
		if err == usecase.ErrTodoNotFound {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		if err == usecase.ErrInvalidInput {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

// CompleteTodo handles PATCH /todos/{id}/complete
func (h *TodoHandler) CompleteTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	todo, err := h.usecase.CompleteTodo(id)
	if err != nil {
		if err == usecase.ErrTodoNotFound {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(todo)
}

// DeleteTodo handles DELETE /todos/{id}
func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	err = h.usecase.DeleteTodo(id)
	if err != nil {
		if err == usecase.ErrTodoNotFound {
			http.Error(w, "Todo not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}