package main

import (
	"fmt"
	"log"
	"net/http"
	"todo-app/handler"
	"todo-app/repository"
	"todo-app/usecase"

	"github.com/gorilla/mux"
)

func main() {
	// Initialize repository
	repo := repository.NewMemoryTodoRepository()
	
	// Initialize usecase
	todoUsecase := usecase.NewTodoUsecase(repo)
	
	// Initialize handler
	todoHandler := handler.NewTodoHandler(todoUsecase)
	
	// Setup routes
	router := mux.NewRouter()
	
	// Todo routes
	router.HandleFunc("/todos", todoHandler.CreateTodo).Methods("POST")
	router.HandleFunc("/todos", todoHandler.GetTodos).Methods("GET")
	router.HandleFunc("/todos/{id}", todoHandler.GetTodo).Methods("GET")
	router.HandleFunc("/todos/{id}", todoHandler.UpdateTodo).Methods("PUT")
	router.HandleFunc("/todos/{id}/complete", todoHandler.CompleteTodo).Methods("PATCH")
	router.HandleFunc("/todos/{id}", todoHandler.DeleteTodo).Methods("DELETE")
	
	// Health check
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET")
	
	fmt.Println("Server starting on :8080")
	fmt.Println("Available endpoints:")
	fmt.Println("  GET    /health")
	fmt.Println("  POST   /todos")
	fmt.Println("  GET    /todos")
	fmt.Println("  GET    /todos/{id}")
	fmt.Println("  PUT    /todos/{id}")
	fmt.Println("  PATCH  /todos/{id}/complete")
	fmt.Println("  DELETE /todos/{id}")
	
	log.Fatal(http.ListenAndServe(":8080", router))
}