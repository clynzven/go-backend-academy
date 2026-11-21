package apperr_test

import (
	"errors"
	"fmt"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// go test checks the printed lines against the Output comment below.
func Example_wrappedSentinel() {
	err := fmt.Errorf("task %d %w", 7, apperr.ErrNotFound)

	fmt.Println(err)
	fmt.Println(err == apperr.ErrNotFound)          // compares the outer envelope only
	fmt.Println(errors.Is(err, apperr.ErrNotFound)) // opens every envelope
	// Output:
	// task 7 not found
	// false
	// true
}
