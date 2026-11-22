-- 001: tasks, each owned by one tenant.
CREATE TABLE tasks (
    id         BIGSERIAL PRIMARY KEY,
    tenant_id  TEXT        NOT NULL,
    title      TEXT        NOT NULL,
    done       BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Every query filters by tenant first.
CREATE INDEX tasks_tenant_idx ON tasks (tenant_id, id);

-- The lesson 4 rule, now enforced by the database itself:
-- one tenant can't have two OPEN tasks with the same title (any case).
CREATE UNIQUE INDEX tasks_open_title_uniq ON tasks (tenant_id, lower(title)) WHERE NOT done;
