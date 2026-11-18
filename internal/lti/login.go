package lti

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/hnrykm/lti-flashcards/internal/auth"
)

// stateTTL is how long a user has between login initiation and launch.
const stateTTL = 10 * time.Minute

// Handler serves the two LTI endpoints.
type Handler struct {
	store  Store
	keys   KeySets
	signer *auth.Signer
	now    func() time.Time // replaceable in tests
}

func NewHandler(store Store, keys KeySets, signer *auth.Signer) *Handler {
	return &Handler{store: store, keys: keys, signer: signer, now: time.Now}
}

func (h *Handler) Register(app fiber.Router) {
	app.Get("/lti/login", h.login)
	app.Post("/lti/login", h.login)
	app.Post("/lti/launch", h.launch)
}

// param reads a value from the query string (GET) or the form body (POST);
// platforms may use either for login initiation.
func param(c fiber.Ctx, name string) string {
	if v := c.Query(name); v != "" {
		return v
	}
	return c.FormValue(name)
}

// randomToken returns 32 random bytes, base64url-encoded.
func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// login is step 1 of the launch (OIDC login initiation). The platform sends
// iss and login_hint; we save a fresh state and nonce, then redirect the
// browser to the platform's auth URL to ask for a signed id_token.
func (h *Handler) login(c fiber.Ctx) error {
	iss, loginHint := param(c, "iss"), param(c, "login_hint")
	if iss == "" || loginHint == "" {
		return fiber.NewError(fiber.StatusBadRequest, "iss and login_hint are required")
	}

	p, err := h.store.FindPlatform(c.Context(), iss, param(c, "client_id"))
	if errors.Is(err, ErrNotFound) {
		return fiber.NewError(fiber.StatusBadRequest, "unknown issuer")
	}
	if err != nil {
		return err
	}

	state, err := randomToken()
	if err != nil {
		return err
	}
	nonce, err := randomToken()
	if err != nil {
		return err
	}
	if err := h.store.SaveState(c.Context(), state, nonce, p.ID, h.now().Add(stateTTL)); err != nil {
		return err
	}

	q := url.Values{
		"scope":         {"openid"},
		"response_type": {"id_token"},
		"response_mode": {"form_post"},
		"prompt":        {"none"},
		"client_id":     {p.ClientID},
		"redirect_uri":  {c.BaseURL() + "/lti/launch"},
		"login_hint":    {loginHint},
		"state":         {state},
		"nonce":         {nonce},
	}
	if hint := param(c, "lti_message_hint"); hint != "" {
		q.Set("lti_message_hint", hint)
	}

	authURL, err := url.Parse(p.AuthLoginURL)
	if err != nil {
		return err
	}
	// Keep any query string already on the platform's auth URL.
	existing := authURL.Query()
	for k, v := range q {
		existing[k] = v
	}
	authURL.RawQuery = existing.Encode()

	return c.Redirect().Status(fiber.StatusFound).To(authURL.String())
}
