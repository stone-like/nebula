package usecase

import (
	"errors"
	"todo-app/domain"
)

var (
	ErrTodoNotFound = errors.New("todo not found")
	ErrInvalidInput = errors.New("invalid input")
)

// TodoUsecase handles business logic for todo operations
type TodoUsecase struct {
	repo domain.TodoRepository
}

// NewTodoUsecase creates a new TodoUsecase
func NewTodoUsecase(repo domain.TodoRepository) *TodoUsecase {
	return &TodoUsecase{
		repo: repo,
	}
}

// CreateTodo creates a new todo item
func (u *TodoUsecase) CreateTodo(title, description string) (*domain.Todo, error) {
	if title == "" {
		return nil, ErrInvalidInput
	}

	// Generate new ID (simple counter-based approach)
	todos, _ := u.repo.GetAll()
	newID := len(todos) + 1

	todo := domain.NewTodo(newID, title, description)
	return u.repo.Create(todo)
}

// GetTodo retrieves a todo by ID
func (u *TodoUsecase) GetTodo(id int) (*domain.Todo, error) {
	todo, err := u.repo.GetByID(id)
	if err != nil {
		return nil, ErrTodoNotFound
	}
	return todo, nil
}

// GetAllTodos retrieves all todos
func (u *TodoUsecase) GetAllTodos() ([]*domain.Todo, error) {
	return u.repo.GetAll()
}

// UpdateTodo updates a todo's information
func (u *TodoUsecase) UpdateTodo(id int, title, description string) (*domain.Todo, error) {
	if title == "" {
		return nil, ErrInvalidInput
	}

	todo, err := u.repo.GetByID(id)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	todo.Update(title, description)
	if err := u.repo.Update(todo); err != nil {
		return nil, err
	}

	return todo, nil
}

// CompleteTodo marks a todo as completed
func (u *TodoUsecase) CompleteTodo(id int) (*domain.Todo, error) {
	todo, err := u.repo.GetByID(id)
	if err != nil {
		return nil, ErrTodoNotFound
	}

	todo.MarkCompleted()
	if err := u.repo.Update(todo); err != nil {
		return nil, err
	}

	return todo, nil
}

// DeleteTodo removes a todo
func (u *TodoUsecase) DeleteTodo(id int) error {
	_, err := u.repo.GetByID(id)
	if err != nil {
		return ErrTodoNotFound
	}

	return u.repo.Delete(id)
}

// GetCompletedTodos retrieves all completed todos
func (u *TodoUsecase) GetCompletedTodos() ([]*domain.Todo, error) {
	return u.repo.GetByCompleted(true)
}

// GetPendingTodos retrieves all pending todos
func (u *TodoUsecase) GetPendingTodos() ([]*domain.Todo, error) {
	return u.repo.GetByCompleted(false)
}