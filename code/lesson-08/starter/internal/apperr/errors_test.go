package apperr

import (
	"errors"
	"fmt"
	"testing"
)

func TestWrappedSentinelIsStillFound(t *testing.T) {
	err := fmt.Errorf("task %d %w", 7, ErrNotFound)

	if err.Error() != "task 7 not found" {
		t.Fatalf("message = %q", err.Error())
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatal("errors.Is must look inside the wrapper")
	}
	if err == ErrNotFound {
		t.Fatal("== compares the outer error only; it must be false here")
	}
}

func TestValidationErrorFoundThroughWrapping(t *testing.T) {
	err := fmt.Errorf("create task: %w", &ValidationError{Fields: map[string]string{"title": "required"}})

	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("errors.As must find the ValidationError inside")
	}
	if ve.Fields["title"] != "required" {
		t.Fatalf("fields = %v", ve.Fields)
	}
}
