package domain

// TodoRepository defines the interface for todo data operations
type TodoRepository interface {
	// Create saves a new todo and returns it with assigned ID
	Create(todo *Todo) (*Todo, error)
	
	// GetByID retrieves a todo by its ID
	GetByID(id int) (*Todo, error)
	
	// GetAll retrieves all todos
	GetAll() ([]*Todo, error)
	
	// Update updates an existing todo
	Update(todo *Todo) error
	
	// Delete removes a todo by ID
	Delete(id int) error
	
	// GetByCompleted retrieves todos by completion status
	GetByCompleted(completed bool) ([]*Todo, error)
}