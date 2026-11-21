package main

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// Task is one to-do item. The json tags are the labels on the form:
// they decide the field names clients see.
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
}

// createTaskRequest is what a client may send. It has no ID or Done field,
// so nobody can choose their own ID or create a task that is already done.
type createTaskRequest struct {
	Title string `json:"title"`
}

// taskStore keeps tasks in memory (Postgres arrives in lesson 5).
// Every request runs on its own goroutine, so the mutex makes sure only one
// of them touches the map at a time.
type taskStore struct {
	mu     sync.Mutex
	nextID int64
	tasks  map[int64]Task
}

func newTaskStore() *taskStore {
	return &taskStore{nextID: 1, tasks: make(map[int64]Task)}
}

// add uses a pointer receiver (*taskStore): it changes the real store,
// not a copy of it.
func (s *taskStore) add(title string) Task {
	// TODO 3: lock the mutex (and defer the unlock), build a Task with
	// ID s.nextID, the title and CreatedAt time.Now().UTC().Truncate(time.Second),
	// store it in s.tasks, increment s.nextID, and return the task.
	return Task{}
}

// list returns every task, oldest first.
func (s *taskStore) list() []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Task, 0, len(s.tasks))
	for id := int64(1); id < s.nextID; id++ {
		if t, ok := s.tasks[id]; ok {
			out = append(out, t)
		}
	}
	return out
}

// tasksHandler serves /tasks. Without method routing (lesson 3) we have to
// switch on r.Method ourselves.
func (s *taskStore) tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.list())

	case http.MethodPost:
		// TODO 4: decode the body into a createTaskRequest with decodeJSON
		// (400 on error), trim the title and reject "" or more than 200
		// characters (400), then reply 201 with the new task from s.add.
		_ = strings.TrimSpace // you will need this in TODO 4; delete this line when you're done
		writeError(w, http.StatusNotImplemented, "TODO 4")

	default:
		w.Header().Set("Allow", "GET, POST")
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}
