package task

import (
	"sync"
	"time"
)

// Store keeps tasks in memory, separately for each tenant (Postgres arrives in lesson 5).
// Every request runs on its own goroutine, so the mutex makes sure only one
// of them touches the maps at a time.
type Store struct {
	mu     sync.Mutex
	nextID int64
	tasks  map[string]map[int64]Task // tenant -> task ID -> task
}

// NewStore returns an empty store, ready to share between handlers.
func NewStore() *Store {
	return &Store{nextID: 1, tasks: make(map[string]map[int64]Task)}
}

// Add creates a task for one tenant.
func (s *Store) Add(tenant, title string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.tasks[tenant] == nil {
		s.tasks[tenant] = make(map[int64]Task)
	}
	t := Task{ID: s.nextID, Title: title, CreatedAt: time.Now().UTC().Truncate(time.Second)}
	s.tasks[tenant][t.ID] = t
	s.nextID++
	return t
}

// List returns one tenant's tasks, oldest first.
func (s *Store) List(tenant string) []Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Task, 0, len(s.tasks[tenant]))
	for id := int64(1); id < s.nextID; id++ {
		if t, ok := s.tasks[tenant][id]; ok {
			out = append(out, t)
		}
	}
	return out
}

// Get returns one task, but only if it belongs to this tenant.
func (s *Store) Get(tenant string, id int64) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[tenant][id]
	return t, ok
}

// Update applies the fields that were sent. It reports false if this tenant
// has no task with that ID.
func (s *Store) Update(tenant string, id int64, title *string, done *bool) (Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[tenant][id]
	if !ok {
		return Task{}, false
	}
	if title != nil {
		t.Title = *title
	}
	if done != nil {
		t.Done = *done
	}
	s.tasks[tenant][id] = t
	return t, true
}
