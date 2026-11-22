-- 006: every payment-provider event we have applied, by its id. A replayed
-- event finds its id here and changes nothing.
CREATE TABLE processed_events (
    event_id    TEXT PRIMARY KEY,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
