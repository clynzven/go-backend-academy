package httpx

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

func TestWriteErrMapsEveryKind(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
		wantBody string
	}{
		{"bad request", &apperr.BadRequestError{Msg: "body must not be empty"}, 400, "body must not be empty"},
		{"validation", &apperr.ValidationError{Fields: map[string]string{"title": "required"}}, 422, `"title": "required"`},
		{"not found", fmt.Errorf("task 7 %w", apperr.ErrNotFound), 404, "task 7 not found"},
		{"conflict", fmt.Errorf("task %q %w", "Buy milk", apperr.ErrConflict), 409, `already exists`},
		{"anything else", errors.New("db: password=hunter2 refused"), 500, "internal server error"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			WriteErr(rec, httptest.NewRequest("GET", "/tasks/7", nil), tc.err)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if !strings.Contains(rec.Body.String(), tc.wantBody) {
				t.Fatalf("body = %s, want it to contain %s", rec.Body, tc.wantBody)
			}
		})
	}
}

// A 500 must never leak the real cause to the client, but must log it for us.
func TestInternalErrorIsLoggedNotLeaked(t *testing.T) {
	var logs bytes.Buffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)

	rec := httptest.NewRecorder()
	secret := errors.New("db: password=hunter2 refused")
	WriteErr(rec, httptest.NewRequest("POST", "/tasks", nil), secret)

	if strings.Contains(rec.Body.String(), "hunter2") {
		t.Fatalf("client saw the secret: %s", rec.Body)
	}
	if !strings.Contains(logs.String(), "hunter2") {
		t.Fatalf("log is missing the real cause: %q", logs.String())
	}
}
