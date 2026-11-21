// Package apperr defines the kinds of errors our app can return. Handlers
// return these; httpx.WriteErr turns each kind into the right HTTP status.
package apperr

import (
	"errors"
	"sort"
	"strings"
)

// Sentinel errors: standard stamps. Wrap them with %w to add details,
// like fmt.Errorf("task %d %w", id, apperr.ErrNotFound) -> "task 7 not found".
var (
	ErrNotFound = errors.New("not found")      // -> 404
	ErrConflict = errors.New("already exists") // -> 409
)

// BadRequestError means the request itself is broken: bad JSON, a missing
// header, an id that isn't a number. -> 400
type BadRequestError struct {
	Msg string
}

func (e *BadRequestError) Error() string { return e.Msg }

// ValidationError means the request was well-formed, but some fields break
// our rules. Fields maps each field name to its problem. -> 422
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	names := make([]string, 0, len(e.Fields))
	for name := range e.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, len(names))
	for i, name := range names {
		parts[i] = name + ": " + e.Fields[name]
	}
	return "validation failed: " + strings.Join(parts, "; ")
}
