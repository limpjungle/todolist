package event

import "todolist/internal/model"

type TodoEventType string

const (
	TodoCreated TodoEventType = "todo_created"
	TodoUpdated TodoEventType = "todo_updated"
	TodoDeleted TodoEventType = "todo_deleted"
)

type TodoEvent struct {
	Type      TodoEventType `json:"type"`
	TodoID    int           `json:"todo_id"`
	Todo      *model.Todo   `json:"todo,omitempty"`
	Timestamp int64         `json:"timestamp"`
}
