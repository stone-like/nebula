package repository

import (
	"errors"
	"sync"
	"todo-app/domain"
)

var (
	ErrTodoNotFound = errors.New("todo not found")
)

// MemoryTodoRepository implements TodoRepository using in-memory storage
type MemoryTodoRepository struct {
	todos map[int]*domain.Todo
	mutex sync.RWMutex
}

// NewMemoryTodoRepository creates a new in-memory todo repository
func NewMemoryTodoRepository() *MemoryTodoRepository {
	return &MemoryTodoRepository{
		todos: make(map[int]*domain.Todo),
		mutex: sync.RWMutex{},
	}
}

// Create saves a new todo and returns it
func (r *MemoryTodoRepository) Create(todo *domain.Todo) (*domain.Todo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	r.todos[todo.ID] = todo
	return todo, nil
}

// GetByID retrieves a todo by its ID
func (r *MemoryTodoRepository) GetByID(id int) (*domain.Todo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	todo, exists := r.todos[id]
	if !exists {
		return nil, ErrTodoNotFound
	}
	
	return todo, nil
}

// GetAll retrieves all todos
func (r *MemoryTodoRepository) GetAll() ([]*domain.Todo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	todos := make([]*domain.Todo, 0, len(r.todos))
	for _, todo := range r.todos {
		todos = append(todos, todo)
	}
	
	return todos, nil
}

// Update updates an existing todo
func (r *MemoryTodoRepository) Update(todo *domain.Todo) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.todos[todo.ID]; !exists {
		return ErrTodoNotFound
	}
	
	r.todos[todo.ID] = todo
	return nil
}

// Delete removes a todo by ID
func (r *MemoryTodoRepository) Delete(id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	
	if _, exists := r.todos[id]; !exists {
		return ErrTodoNotFound
	}
	
	delete(r.todos, id)
	return nil
}

// GetByCompleted retrieves todos by completion status
func (r *MemoryTodoRepository) GetByCompleted(completed bool) ([]*domain.Todo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	
	var todos []*domain.Todo
	for _, todo := range r.todos {
		if todo.Completed == completed {
			todos = append(todos, todo)
		}
	}
	
	return todos, nil
}