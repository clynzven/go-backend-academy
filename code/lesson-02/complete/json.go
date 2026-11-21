package main

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
)

// decodeJSON reads exactly one JSON object from the request body into dst.
// It is strict on purpose: unknown fields, wrong types, an empty body and
// extra data after the object are all rejected.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // {"title":"x","is_admin":true} -> error

	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("request body must not be empty")
		}
		return err
	}

	// A second Decode must hit the end of the body. Anything else means the
	// client sent more than one JSON value, like {"title":"a"}{"title":"b"}.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

// writeJSON sends v as indented JSON with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		// The status line is already sent, so all we can do is log it.
		log.Printf("writeJSON: %v", err)
	}
}

// writeError sends {"error": "..."} with the given status code.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
