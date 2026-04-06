package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"keda-kind/sample-app/internal/dequeue"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, connectionString string) (*Store, error) {
	pool, err := pgxpool.New(ctx, connectionString)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Init(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
CREATE TABLE IF NOT EXISTS queue_messages (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(5) NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL,
    stored_at TIMESTAMPTZ NOT NULL,
    queue_name TEXT NOT NULL,
    raw_body JSONB NOT NULL
)
`)
	if err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *Store) Save(ctx context.Context, msg dequeue.StoredMessage) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO queue_messages (code, sent_at, stored_at, queue_name, raw_body)
VALUES ($1, $2, $3, $4, $5::jsonb)
`, msg.Code, msg.SentAt, msg.StoredAt, msg.QueueName, msg.RawBody)
	if err != nil {
		return fmt.Errorf("insert message: %w", err)
	}
	return nil
}
