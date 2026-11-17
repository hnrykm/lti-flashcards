package flashcards

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store runs the SQL for decks and cards. Every method takes the course ID
// from the session and filters by it, so one course can never read or change
// another course's data. A row in another course looks exactly like a missing
// row (ErrNotFound), so IDs don't leak between courses.
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const deckColumns = `id, course_id, name, created_at`
const cardColumns = `id, deck_id, front, back, created_at`

// one runs a query that returns a single row and maps "no rows" to ErrNotFound.
func one[T any](rows pgx.Rows, err error) (T, error) {
	if err != nil {
		var zero T
		return zero, err
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return v, ErrNotFound
	}
	return v, err
}

// many runs a query and collects every row; an empty result is an empty slice, not nil,
// so it encodes as [] in JSON.
func many[T any](rows pgx.Rows, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}

// --- decks ---

func (s *Store) ListDecks(ctx context.Context, courseID int64) ([]Deck, error) {
	return many[Deck](s.pool.Query(ctx,
		`SELECT `+deckColumns+` FROM decks WHERE course_id = $1 ORDER BY created_at, id`,
		courseID))
}

func (s *Store) GetDeck(ctx context.Context, courseID, id int64) (Deck, error) {
	return one[Deck](s.pool.Query(ctx,
		`SELECT `+deckColumns+` FROM decks WHERE id = $1 AND course_id = $2`,
		id, courseID))
}

func (s *Store) CreateDeck(ctx context.Context, courseID int64, req DeckRequest) (Deck, error) {
	return one[Deck](s.pool.Query(ctx,
		`INSERT INTO decks (course_id, name) VALUES ($1, $2) RETURNING `+deckColumns,
		courseID, req.Name))
}

func (s *Store) UpdateDeck(ctx context.Context, courseID, id int64, req DeckRequest) (Deck, error) {
	return one[Deck](s.pool.Query(ctx,
		`UPDATE decks SET name = $3 WHERE id = $1 AND course_id = $2 RETURNING `+deckColumns,
		id, courseID, req.Name))
}

// DeleteDeck deletes the deck; ON DELETE CASCADE removes its cards.
func (s *Store) DeleteDeck(ctx context.Context, courseID, id int64) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM decks WHERE id = $1 AND course_id = $2`, id, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// --- cards ---

// ListCards returns ErrNotFound when the deck isn't in this course, and an
// empty slice when the deck exists but has no cards.
func (s *Store) ListCards(ctx context.Context, courseID, deckID int64) ([]Card, error) {
	if _, err := s.GetDeck(ctx, courseID, deckID); err != nil {
		return nil, err
	}
	return many[Card](s.pool.Query(ctx,
		`SELECT `+cardColumns+` FROM cards WHERE deck_id = $1 ORDER BY created_at, id`,
		deckID))
}

// CreateCard inserts only if the deck belongs to the course: the SELECT
// returns no row otherwise, so nothing is inserted and we report ErrNotFound.
func (s *Store) CreateCard(ctx context.Context, courseID, deckID int64, req CardRequest) (Card, error) {
	return one[Card](s.pool.Query(ctx,
		`INSERT INTO cards (deck_id, front, back)
		 SELECT id, $3, $4 FROM decks WHERE id = $1 AND course_id = $2
		 RETURNING `+cardColumns,
		deckID, courseID, req.Front, req.Back))
}

func (s *Store) UpdateCard(ctx context.Context, courseID, id int64, req CardRequest) (Card, error) {
	return one[Card](s.pool.Query(ctx,
		`UPDATE cards c SET front = $3, back = $4
		 FROM decks d
		 WHERE c.id = $1 AND c.deck_id = d.id AND d.course_id = $2
		 RETURNING c.id, c.deck_id, c.front, c.back, c.created_at`,
		id, courseID, req.Front, req.Back))
}

func (s *Store) DeleteCard(ctx context.Context, courseID, id int64) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM cards c USING decks d
		 WHERE c.id = $1 AND c.deck_id = d.id AND d.course_id = $2`,
		id, courseID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
