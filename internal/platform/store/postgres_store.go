package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	_ "github.com/jackc/pgx/v5/stdlib"

	"unifiedsubscriptionproxy/internal/platform/domain"
)

type PostgresStore struct {
	db    *sql.DB
	table string
	mu    sync.Mutex
}

func NewPostgresStore(ctx context.Context, dsn string) (*PostgresStore, error) {
	db, err := sql.Open("pgx", strings.TrimSpace(dsn))
	if err != nil {
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	store := &PostgresStore{db: db, table: "platform_state"}
	if err := store.ensureSchema(ctx); err != nil {
		return nil, err
	}
	if err := store.ensureBootstrap(ctx); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *PostgresStore) Load() (domain.PlatformData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadUnlocked(context.Background())
}

func (s *PostgresStore) Save(data domain.PlatformData) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveUnlocked(context.Background(), data)
}

func (s *PostgresStore) Mutate(fn func(*domain.PlatformData) error) (domain.PlatformData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := s.loadUnlocked(context.Background())
	if err != nil {
		return domain.PlatformData{}, err
	}
	if err := fn(&data); err != nil {
		return domain.PlatformData{}, err
	}
	if err := s.saveUnlocked(context.Background(), data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (s *PostgresStore) ensureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id TEXT PRIMARY KEY,
			payload JSONB NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`, s.table))
	return err
}

func (s *PostgresStore) ensureBootstrap(ctx context.Context) error {
	var exists bool
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE id = $1)", s.table), "default").Scan(&exists); err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.saveUnlocked(ctx, BootstrapData())
}

func (s *PostgresStore) loadUnlocked(ctx context.Context) (domain.PlatformData, error) {
	var raw []byte
	err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT payload FROM %s WHERE id = $1", s.table), "default").Scan(&raw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			data := BootstrapData()
			if err := s.saveUnlocked(ctx, data); err != nil {
				return domain.PlatformData{}, err
			}
			return data, nil
		}
		return domain.PlatformData{}, err
	}
	var data domain.PlatformData
	if err := json.Unmarshal(raw, &data); err != nil {
		return domain.PlatformData{}, err
	}
	return data, nil
}

func (s *PostgresStore) saveUnlocked(ctx context.Context, data domain.PlatformData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (id, payload, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (id) DO UPDATE SET payload = EXCLUDED.payload, updated_at = NOW()
	`, s.table), "default", raw)
	return err
}
