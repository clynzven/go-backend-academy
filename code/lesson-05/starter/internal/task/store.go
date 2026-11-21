package task

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// Store is everything the handlers need from storage. Two types satisfy it:
// PostgresStore (the real one) and MemoryStore (fast, for unit tests).
// Go interfaces are satisfied implicitly: no "implements" keyword.
type Store interface {
	Add(ctx context.Context, tenant, title string) (Task, error)
	List(ctx context.Context, tenant string) ([]Task, error)
	Get(ctx context.Context, tenant string, id int64) (Task, error)
	Update(ctx context.Context, tenant string, id int64, title *string, done *bool) (Task, error)
}

// MemoryStore keeps tasks in memory, separately for each tenant.
// Every request runs on its own goroutine, so the mutex makes sure only one
// of them touches the maps at a time.
type MemoryStore struct {
	mu     sync.Mutex
	nextID int64
	tasks  map[string]map[int64]Task // tenant -> task ID -> task
}

// NewMemoryStore returns an empty in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{nextID: 1, tasks: make(map[string]map[int64]Task)}
}

// Add creates a task for one tenant. A tenant can't have two open tasks with
// the same title: that returns an error wrapping apperr.ErrConflict.
func (s *MemoryStore) Add(_ context.Context, tenant, title string) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.tasks[tenant] {
		if !t.Done && strings.EqualFold(t.Title, title) {
			return Task{}, fmt.Errorf("task %q %w", title, apperr.ErrConflict)
		}
	}
	if s.tasks[tenant] == nil {
		s.tasks[tenant] = make(map[int64]Task)
	}
	t := Task{ID: s.nextID, Title: title, CreatedAt: time.Now().UTC().Truncate(time.Second)}
	s.tasks[tenant][t.ID] = t
	s.nextID++
	return t, nil
}

// List returns one tenant's tasks, oldest first.
func (s *MemoryStore) List(_ context.Context, tenant string) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]Task, 0, len(s.tasks[tenant]))
	for id := int64(1); id < s.nextID; id++ {
		if t, ok := s.tasks[tenant][id]; ok {
			out = append(out, t)
		}
	}
	return out, nil
}

// Get returns one task, but only if it belongs to this tenant.
func (s *MemoryStore) Get(_ context.Context, tenant string, id int64) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[tenant][id]
	if !ok {
		return Task{}, fmt.Errorf("task %d %w", id, apperr.ErrNotFound)
	}
	return t, nil
}

// Update applies the fields that were sent. Unknown IDs (for this tenant)
// return an error wrapping apperr.ErrNotFound.
func (s *MemoryStore) Update(_ context.Context, tenant string, id int64, title *string, done *bool) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tasks[tenant][id]
	if !ok {
		return Task{}, fmt.Errorf("task %d %w", id, apperr.ErrNotFound)
	}
	if title != nil {
		t.Title = *title
	}
	if done != nil {
		t.Done = *done
	}
	s.tasks[tenant][id] = t
	return t, nil
}
