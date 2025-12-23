package worker

import (
	"fmt"
	"strings"

	"github.com/amitbasuri/taskqueue-go/internal/models"
)

// HandlerRegistry manages the registration and lookup of task handlers
type HandlerRegistry struct {
	handlers map[models.TaskType]models.TaskHandler
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[models.TaskType]models.TaskHandler),
	}
}

// Register adds a handler to the registry
// Normalizes handler type to lowercase for consistent lookups
func (r *HandlerRegistry) Register(handler models.TaskHandler) {
	normalizedType := models.TaskType(strings.ToLower(string(handler.Type())))
	r.handlers[normalizedType] = handler
}

// Get retrieves a handler by task type (case-insensitive)
func (r *HandlerRegistry) Get(taskType string) (models.TaskHandler, error) {
	// Normalize to lowercase for case-insensitive lookup
	normalizedType := models.TaskType(strings.ToLower(taskType))

	handler, ok := r.handlers[normalizedType]
	if !ok {
		return nil, fmt.Errorf("handler not found for type: %s", taskType)
	}
	return handler, nil
}

// Has checks if a handler exists for the given task type (case-insensitive)
func (r *HandlerRegistry) Has(taskType string) bool {
	normalizedType := models.TaskType(strings.ToLower(taskType))
	_, ok := r.handlers[normalizedType]
	return ok
}

// List returns all registered task types
func (r *HandlerRegistry) List() []string {
	types := make([]string, 0, len(r.handlers))
	for taskType := range r.handlers {
		types = append(types, string(taskType))
	}
	return types
}
