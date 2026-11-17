// Package flashcards holds decks and cards: models, SQL, and HTTP handlers.
package flashcards

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("not found")

type Deck struct {
	ID        int64     `json:"id" db:"id"`
	CourseID  int64     `json:"course_id" db:"course_id"`
	Name      string    `json:"name" db:"name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Card struct {
	ID        int64     `json:"id" db:"id"`
	DeckID    int64     `json:"deck_id" db:"deck_id"`
	Front     string    `json:"front" db:"front"`
	Back      string    `json:"back" db:"back"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type DeckRequest struct {
	Name string `json:"name" db:"name"`
}

func (r *DeckRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	return checkLength("name", r.Name, 100)
}

type CardRequest struct {
	Front string `json:"front" db:"front"`
	Back  string `json:"back" db:"back"`
}

func (r *CardRequest) Validate() error {
	r.Front = strings.TrimSpace(r.Front)
	r.Back = strings.TrimSpace(r.Back)
	if err := checkLength("front", r.Front, 1000); err != nil {
		return err
	}
	return checkLength("back", r.Back, 1000)
}

func checkLength(field, value string, max int) error {
	n := utf8.RuneCountInString(value) // characters, like Postgres char_length

	if n == 0 {
		return errors.New(field + " is required")
	}
	if n > max {
		return errors.New(field + " is too long")
	}
	return nil
}
