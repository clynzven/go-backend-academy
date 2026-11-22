// Package task is everything about tasks: the data, the store and the HTTP handlers.
package task

import "time"

// Task is one to-do item, owned by one tenant.
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest is what a client may send to create a task.
type CreateRequest struct {
	Title string `json:"title"`
}

// UpdateRequest is what a client may send to change a task. The fields are
// pointers so we can tell "not sent" (nil) from "sent as empty/false".
type UpdateRequest struct {
	Title *string `json:"title"`
	Done  *bool   `json:"done"`
}
