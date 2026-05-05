package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/lib/pq"

	"github.com/n1ckerr0r/shortener/internal/link"
)

const schema = `
CREATE TABLE IF NOT EXISTS links (
	short_code TEXT PRIMARY KEY,
	original_url TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL,
	expires_at TIMESTAMPTZ,
	blocked BOOLEAN NOT NULL DEFAULT FALSE
);
`

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, dsn string) (*Repository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	repo := NewRepository(db)
	if err = repo.Ping(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err = repo.Init(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *Repository) Init(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) Save(ctx context.Context, shortLink *link.ShortLink) error {
	var expiresAt sql.NullTime
	if value := shortLink.ExpiresAt(); value != nil {
		expiresAt = sql.NullTime{Time: *value, Valid: true}
	}

	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO links (short_code, original_url, created_at, expires_at, blocked)
		 VALUES ($1, $2, $3, $4, $5)`,
		shortLink.ShortCode().Value(),
		shortLink.OriginalURL().Value(),
		shortLink.CreatedAt(),
		expiresAt,
		shortLink.IsBlocked(),
	)
	return err
}

func (r *Repository) Find(ctx context.Context, code link.ShortCode) (*link.ShortLink, error) {
	var (
		shortCode   string
		originalURL string
		createdAt   time.Time
		expiresAt   sql.NullTime
		blocked     bool
	)

	err := r.db.QueryRowContext(
		ctx,
		`SELECT short_code, original_url, created_at, expires_at, blocked
		 FROM links
		 WHERE short_code = $1`,
		code.Value(),
	).Scan(&shortCode, &originalURL, &createdAt, &expiresAt, &blocked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, link.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	restoredCode, err := link.NewShortCode(shortCode)
	if err != nil {
		return nil, err
	}

	restoredURL, err := link.NewOriginalURL(originalURL)
	if err != nil {
		return nil, err
	}

	var expiresAtPtr *time.Time
	if expiresAt.Valid {
		expiresAtPtr = &expiresAt.Time
	}

	shortLink, err := link.NewShortLink(
		restoredCode,
		restoredURL,
		createdAt,
		link.NewExpiration(expiresAtPtr),
	)
	if err != nil {
		return nil, err
	}
	if blocked {
		shortLink.Block()
	}

	return shortLink, nil
}

func (r *Repository) Exists(ctx context.Context, code link.ShortCode) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM links WHERE short_code = $1)`,
		code.Value(),
	).Scan(&exists)
	return exists, err
}
