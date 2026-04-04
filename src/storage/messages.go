package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type MessageRecord struct {
	Account      string
	Sender       string
	SenderName   string
	SourceDevice int
	Timestamp    int64
	MessageText  string
	GroupID      string
	EnvelopeType string
	RawJSON      json.RawMessage
}

type QueryFilter struct {
	Sender        string
	GroupID       string
	StartTime     *int64
	EndTime       *int64
	EnvelopeTypes []string // defaults to ["dataMessage"] when empty
	Limit         int      // default 100, hard cap 1000
	Offset        int
}

type StoredMessage struct {
	ID           int64           `json:"id"`
	Account      string          `json:"account"`
	Sender       string          `json:"sender"`
	SenderName   string          `json:"sender_name"`
	SourceDevice int             `json:"source_device"`
	Timestamp    int64           `json:"timestamp"`
	MessageText  string          `json:"message_text"`
	GroupID      string          `json:"group_id"`
	EnvelopeType string          `json:"envelope_type"`
	RawJSON      json.RawMessage `json:"raw_json"`
	ReceivedAt   time.Time       `json:"received_at"`
}

func (s *Store) SaveMessage(ctx context.Context, r MessageRecord) error {
	const q = `
		INSERT INTO signal_messages
			(account, sender, sender_name, source_device, timestamp,
			 message_text, group_id, envelope_type, raw_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	_, err := s.pool.Exec(ctx, q,
		r.Account, r.Sender, r.SenderName, r.SourceDevice, r.Timestamp,
		r.MessageText, r.GroupID, r.EnvelopeType, r.RawJSON)
	return err
}

func (s *Store) GetMessages(ctx context.Context, account string, f QueryFilter) ([]StoredMessage, error) {
	envelopeTypes := f.EnvelopeTypes
	if len(envelopeTypes) == 0 {
		envelopeTypes = []string{"dataMessage"}
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 100
	} else if limit > 1000 {
		limit = 1000
	}

	args := []any{account, envelopeTypes}
	paramIdx := 3
	where := "WHERE account = $1 AND envelope_type = ANY($2)"

	if f.Sender != "" {
		where += fmt.Sprintf(" AND sender = $%d", paramIdx)
		args = append(args, f.Sender)
		paramIdx++
	}
	if f.GroupID != "" {
		where += fmt.Sprintf(" AND group_id = $%d", paramIdx)
		args = append(args, f.GroupID)
		paramIdx++
	}
	if f.StartTime != nil {
		where += fmt.Sprintf(" AND timestamp >= $%d", paramIdx)
		args = append(args, *f.StartTime)
		paramIdx++
	}
	if f.EndTime != nil {
		where += fmt.Sprintf(" AND timestamp <= $%d", paramIdx)
		args = append(args, *f.EndTime)
		paramIdx++
	}

	args = append(args, limit, f.Offset)
	q := fmt.Sprintf(`
		SELECT id, account, sender, sender_name, source_device, timestamp,
		       message_text, group_id, envelope_type, raw_json, received_at
		FROM signal_messages
		%s
		ORDER BY timestamp ASC
		LIMIT $%d OFFSET $%d`,
		where, paramIdx, paramIdx+1)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(
			&m.ID, &m.Account, &m.Sender, &m.SenderName, &m.SourceDevice,
			&m.Timestamp, &m.MessageText, &m.GroupID, &m.EnvelopeType,
			&m.RawJSON, &m.ReceivedAt,
		); err != nil {
			return nil, err
		}
		messages = append(messages, m)
	}
	return messages, rows.Err()
}
