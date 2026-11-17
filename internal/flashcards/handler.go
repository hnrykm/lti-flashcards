package flashcards

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/hnrykm/lti-flashcards/internal/auth"
)

type Handler struct {
	store *Store
}

func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// Register adds the CRUD routes. The router must already run auth
// middleware: every handler reads the course from the session.
// Reads are open to any course member; writes need an instructor.
func (h *Handler) Register(r fiber.Router) {
	r.Get("/decks", h.listDecks)
	r.Post("/decks", auth.RequireInstructor, h.createDeck)
	r.Get("/decks/:id", h.getDeck)
	r.Put("/decks/:id", auth.RequireInstructor, h.updateDeck)
	r.Delete("/decks/:id", auth.RequireInstructor, h.deleteDeck)

	r.Get("/decks/:id/cards", h.listCards)
	r.Post("/decks/:id/cards", auth.RequireInstructor, h.createCard)
	r.Put("/cards/:id", auth.RequireInstructor, h.updateCard)
	r.Delete("/cards/:id", auth.RequireInstructor, h.deleteCard)
}

func parseID(c fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	return id, nil
}

// bindValid decodes the JSON body into req and runs its Validate method.
func bindValid[T any, PT interface {
	*T
	Validate() error
}](c fiber.Ctx) (T, error) {
	var req T
	if err := c.Bind().JSON(&req); err != nil {
		return req, fiber.NewError(fiber.StatusBadRequest, "invalid JSON body")
	}
	if err := PT(&req).Validate(); err != nil {
		return req, fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	return req, nil
}

func storeError(err error) error {
	if errors.Is(err, ErrNotFound) {
		return fiber.ErrNotFound // 404
	}
	return err // anything else → 500
}

func courseID(c fiber.Ctx) int64 {
	return auth.FromCtx(c).CourseID
}

// --- decks ---

func (h *Handler) listDecks(c fiber.Ctx) error {
	decks, err := h.store.ListDecks(c.Context(), courseID(c))
	if err != nil {
		return storeError(err)
	}
	return c.JSON(decks)
}

func (h *Handler) getDeck(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	d, err := h.store.GetDeck(c.Context(), courseID(c), id)
	if err != nil {
		return storeError(err)
	}
	return c.JSON(d)
}

func (h *Handler) createDeck(c fiber.Ctx) error {
	req, err := bindValid[DeckRequest](c)
	if err != nil {
		return err
	}
	d, err := h.store.CreateDeck(c.Context(), courseID(c), req)
	if err != nil {
		return storeError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(d) // 201
}

func (h *Handler) updateDeck(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	req, err := bindValid[DeckRequest](c)
	if err != nil {
		return err
	}
	d, err := h.store.UpdateDeck(c.Context(), courseID(c), id, req)
	if err != nil {
		return storeError(err)
	}
	return c.JSON(d)
}

func (h *Handler) deleteDeck(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.store.DeleteDeck(c.Context(), courseID(c), id); err != nil {
		return storeError(err)
	}
	return c.SendStatus(fiber.StatusNoContent) // 204
}

// --- cards ---

func (h *Handler) listCards(c fiber.Ctx) error {
	deckID, err := parseID(c)
	if err != nil {
		return err
	}
	cards, err := h.store.ListCards(c.Context(), courseID(c), deckID)
	if err != nil {
		return storeError(err)
	}
	return c.JSON(cards)
}

func (h *Handler) createCard(c fiber.Ctx) error {
	deckID, err := parseID(c)
	if err != nil {
		return err
	}
	req, err := bindValid[CardRequest](c)
	if err != nil {
		return err
	}
	card, err := h.store.CreateCard(c.Context(), courseID(c), deckID, req)
	if err != nil {
		return storeError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(card)
}

func (h *Handler) updateCard(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	req, err := bindValid[CardRequest](c)
	if err != nil {
		return err
	}
	card, err := h.store.UpdateCard(c.Context(), courseID(c), id, req)
	if err != nil {
		return storeError(err)
	}
	return c.JSON(card)
}

func (h *Handler) deleteCard(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := h.store.DeleteCard(c.Context(), courseID(c), id); err != nil {
		return storeError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
