package lti

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

// Platform is one registered LMS (one row in platforms).
type Platform struct {
	ID           int64  `db:"id"`
	Issuer       string `db:"issuer"`
	ClientID     string `db:"client_id"`
	DeploymentID string `db:"deployment_id"`
	AuthLoginURL string `db:"auth_login_url"`
	JWKSURL      string `db:"jwks_url"`
}

// LoginState is what /lti/login saved for /lti/launch to check.
type LoginState struct {
	Nonce      string `db:"nonce"`
	PlatformID int64  `db:"platform_id"`
	Used       bool   `db:"used"`
}

// Store is what the LTI handlers need from the database. It's an interface
// so the launch test can use an in-memory fake.
type Store interface {
	FindPlatform(ctx context.Context, issuer, clientID string) (Platform, error)
	GetPlatform(ctx context.Context, id int64) (Platform, error)
	SaveState(ctx context.Context, state, nonce string, platformID int64, expiresAt time.Time) error
	LoadState(ctx context.Context, state string) (LoginState, error)
	MarkStateUsed(ctx context.Context, state string) error
	UpsertCourse(ctx context.Context, platformID int64, contextID, title string) (int64, error)
}

// PGStore is the Postgres implementation of Store.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

const platformColumns = `id, issuer, client_id, deployment_id, auth_login_url, jwks_url`

func onePlatform(rows pgx.Rows, err error) (Platform, error) {
	if err != nil {
		return Platform{}, err
	}
	p, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Platform])
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

// FindPlatform looks a platform up by issuer, and by client_id when the
// login request included one (platforms may send it; it's optional).
func (s *PGStore) FindPlatform(ctx context.Context, issuer, clientID string) (Platform, error) {
	return onePlatform(s.pool.Query(ctx,
		`SELECT `+platformColumns+` FROM platforms
		 WHERE issuer = $1 AND ($2 = '' OR client_id = $2)
		 ORDER BY id LIMIT 1`,
		issuer, clientID))
}

func (s *PGStore) GetPlatform(ctx context.Context, id int64) (Platform, error) {
	return onePlatform(s.pool.Query(ctx,
		`SELECT `+platformColumns+` FROM platforms WHERE id = $1`, id))
}

func (s *PGStore) SaveState(ctx context.Context, state, nonce string, platformID int64, expiresAt time.Time) error {
	// Old rows are useless after they expire; clear them out as we go.
	if _, err := s.pool.Exec(ctx, `DELETE FROM oidc_states WHERE expires_at < now()`); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO oidc_states (state, nonce, platform_id, expires_at) VALUES ($1, $2, $3, $4)`,
		state, nonce, platformID, expiresAt)
	return err
}

// LoadState returns ErrNotFound for unknown or expired states.
func (s *PGStore) LoadState(ctx context.Context, state string) (LoginState, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT nonce, platform_id, used FROM oidc_states
		 WHERE state = $1 AND expires_at > now()`, state)
	if err != nil {
		return LoginState{}, err
	}
	st, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[LoginState])
	if errors.Is(err, pgx.ErrNoRows) {
		return st, ErrNotFound
	}
	return st, err
}

// MarkStateUsed flips used only if it is still false, so two requests racing
// with the same state can't both succeed.
func (s *PGStore) MarkStateUsed(ctx context.Context, state string) error {
	tag, err := s.pool.Exec(ctx,
		`UPDATE oidc_states SET used = true WHERE state = $1 AND NOT used`, state)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpsertCourse creates the course on its first launch and keeps the title
// current on later ones. It returns the tool's course ID.
func (s *PGStore) UpsertCourse(ctx context.Context, platformID int64, contextID, title string) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx,
		`INSERT INTO courses (platform_id, context_id, title) VALUES ($1, $2, $3)
		 ON CONFLICT (platform_id, context_id) DO UPDATE SET title = EXCLUDED.title
		 RETURNING id`,
		platformID, contextID, title).Scan(&id)
	return id, err
}
