package storage

import "context"

const createTableSQL = `
CREATE TABLE IF NOT EXISTS signal_messages (
    id              BIGSERIAL PRIMARY KEY,
    account         TEXT        NOT NULL,
    sender          TEXT,
    sender_name     TEXT,
    source_device   INT,
    timestamp       BIGINT      NOT NULL,
    message_text    TEXT,
    group_id        TEXT,
    envelope_type   TEXT        NOT NULL,
    raw_json        JSONB       NOT NULL,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_signal_messages_account
    ON signal_messages (account);
CREATE INDEX IF NOT EXISTS idx_signal_messages_sender
    ON signal_messages (sender);
CREATE INDEX IF NOT EXISTS idx_signal_messages_group_id
    ON signal_messages (group_id);
CREATE INDEX IF NOT EXISTS idx_signal_messages_timestamp
    ON signal_messages (timestamp);
CREATE INDEX IF NOT EXISTS idx_signal_messages_account_timestamp
    ON signal_messages (account, timestamp);
CREATE INDEX IF NOT EXISTS idx_signal_messages_raw_json
    ON signal_messages USING GIN (raw_json);
`

func (s *Store) createSchema(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, createTableSQL)
	return err
}
