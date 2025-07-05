package domain

import "time"

// Todo represents a task item
type Todo struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// NewTodo creates a new todo item
func NewTodo(id int, title, description string) *Todo {
	now := time.Now()
	return &Todo{
		ID:          id,
		Title:       title,
		Description: description,
		Completed:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// MarkCompleted marks the todo as completed
func (t *Todo) MarkCompleted() {
	t.Completed = true
	t.UpdatedAt = time.Now()
}

// Update updates the todo's title and description
func (t *Todo) Update(title, description string) {
	t.Title = title
	t.Description = description
	t.UpdatedAt = time.Now()
}