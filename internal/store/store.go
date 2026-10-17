// Package store is the PostgreSQL read model (spec §37).
package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"

	"github.com/agnaldom/crosswatc/db"
)

var (
	ErrNotFound = errors.New("store: not found")
	// ErrConflict signals an optimistic-lock or uniqueness violation.
	ErrConflict = errors.New("store: conflict")
)

type Store struct {
	pool *pgxpool.Pool
}

// Open connects to PostgreSQL.
func Open(ctx context.Context, url string) (*Store, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

// Ping checks connectivity (readiness probe).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

// Migrate applies all embedded migrations.
func Migrate(ctx context.Context, url string) error {
	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	p, err := goose.NewProvider(goose.DialectPostgres, sqlDB, mustSub(db.Migrations, "migrations"))
	if err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("store: migrate: %w", err)
	}
	return nil
}

// Page is cursor pagination input (spec §39).
type Page struct {
	Limit  int
	Cursor string
}

const (
	defaultLimit = 50
	maxLimit     = 200
)

func (p Page) limit() int {
	switch {
	case p.Limit <= 0:
		return defaultLimit
	case p.Limit > maxLimit:
		return maxLimit
	default:
		return p.Limit
	}
}

type cursor struct {
	Key string `json:"k"`
	ID  string `json:"id"`
}

func encodeCursor(key, id string) string {
	b, _ := json.Marshal(cursor{key, id})
	return base64.RawURLEncoding.EncodeToString(b)
}

// ErrBadCursor is returned for malformed cursors.
var ErrBadCursor = errors.New("store: invalid cursor")

func decodeCursor(s string) (*cursor, error) {
	if s == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, ErrBadCursor
	}
	var c cursor
	if err := json.Unmarshal(b, &c); err != nil || c.ID == "" {
		return nil, ErrBadCursor
	}
	return &c, nil
}

func mapErr(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return fmt.Errorf("%w: %s", ErrConflict, pgErr.ConstraintName)
	}
	return err
}
