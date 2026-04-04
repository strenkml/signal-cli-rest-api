package storage

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	s := &Store{pool: pool}
	if err := s.createSchema(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	log.Info("Storage: PostgreSQL connection established")
	return s, nil
}

func (s *Store) Close() {
	s.pool.Close()
}
