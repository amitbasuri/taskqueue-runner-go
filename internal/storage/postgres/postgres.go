package postgres

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store implements the storage.Store interface using PostgreSQL
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new PostgreSQL store
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{
		pool: pool,
	}
}

// GetPool returns the underlying connection pool (for testing)
func (s *Store) GetPool() *pgxpool.Pool {
	return s.pool
}
