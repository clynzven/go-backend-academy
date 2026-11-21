package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/clynzven/go-backend-academy/internal/apperr"
)

// PostgresStore keeps tasks in PostgreSQL. Every query has the tenant in its
// WHERE clause, and every value travels as a $1, $2 parameter, never glued
// into the SQL text.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore returns a store that uses the connection pool db.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

const taskColumns = `id, title, done, created_at`

// scanTask reads one row of taskColumns.
func scanTask(row interface{ Scan(...any) error }) (Task, error) {
	var t Task
	err := row.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt)
	t.CreatedAt = t.CreatedAt.UTC()
	return t, err
}

// Add inserts a task. The unique index on open titles makes Postgres refuse a
// duplicate; we turn that refusal into apperr.ErrConflict.
func (s *PostgresStore) Add(ctx context.Context, tenant, title string) (Task, error) {
	row := s.db.QueryRowContext(ctx,
		`INSERT INTO tasks (tenant_id, title) VALUES ($1, $2) RETURNING `+taskColumns,
		tenant, title)
	t, err := scanTask(row)
	if isUniqueViolation(err) {
		return Task{}, fmt.Errorf("task %q %w", title, apperr.ErrConflict)
	}
	if err != nil {
		return Task{}, fmt.Errorf("add task: %w", err)
	}
	return t, nil
}

// List returns one tenant's tasks, oldest first.
func (s *PostgresStore) List(ctx context.Context, tenant string) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE tenant_id = $1 ORDER BY id`, tenant)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close() // gives the connection back to the pool

	tasks := []Task{} // an empty list is [] in JSON, not null
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("list tasks: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err() // errors that happened while iterating
}

// Get returns one task, but only if it belongs to this tenant.
func (s *PostgresStore) Get(ctx context.Context, tenant string, id int64) (Task, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+taskColumns+` FROM tasks WHERE tenant_id = $1 AND id = $2`, tenant, id)
	t, err := scanTask(row)
	// TODO 2: no row is not a server error. If errors.Is(err, sql.ErrNoRows),
	// return fmt.Errorf("task %d %w", id, apperr.ErrNotFound) so the client gets a 404.
	if err != nil {
		return Task{}, fmt.Errorf("get task: %w", err)
	}
	return t, nil
}

// Update changes only the fields that were sent: COALESCE keeps the old value
// when the parameter is NULL (a nil pointer in Go).
func (s *PostgresStore) Update(ctx context.Context, tenant string, id int64, title *string, done *bool) (Task, error) {
	row := s.db.QueryRowContext(ctx,
		`UPDATE tasks SET title = COALESCE($3, title), done = COALESCE($4, done)
		 WHERE tenant_id = $1 AND id = $2
		 RETURNING `+taskColumns,
		tenant, id, title, done)
	t, err := scanTask(row)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Task{}, fmt.Errorf("task %d %w", id, apperr.ErrNotFound)
	case isUniqueViolation(err):
		return Task{}, fmt.Errorf("an open task with that title %w", apperr.ErrConflict)
	case err != nil:
		return Task{}, fmt.Errorf("update task: %w", err)
	}
	return t, nil
}

// isUniqueViolation reports whether Postgres refused a duplicate (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	// TODO 3: use errors.As to find a *pgconn.PgError in err, and check that its
	// Code is "23505" (unique_violation).
	var pgErr *pgconn.PgError
	_ = pgErr
	return false
}
