-- 007: rate-limit buckets shared by every copy of the API.
CREATE TABLE rate_limits (
    key        TEXT PRIMARY KEY,
    tokens     DOUBLE PRECISION NOT NULL,
    allowed    BOOLEAN          NOT NULL,
    updated_at TIMESTAMPTZ      NOT NULL
);
