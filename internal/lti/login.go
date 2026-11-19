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
	debug  bool             // see WithDebug
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
	h.debugf("<- LOGIN from LMS: %s %s%s", c.Method(), c.Path(), formatParams(url.Values{
		"iss":               {iss},
		"login_hint":        {loginHint},
		"client_id":         {param(c, "client_id")},
		"lti_deployment_id": {param(c, "lti_deployment_id")},
		"target_link_uri":   {param(c, "target_link_uri")},
		"lti_message_hint":  {param(c, "lti_message_hint")},
	}))
	if iss == "" || loginHint == "" {
		h.debugf("   rejected: iss and login_hint are required")
		return fiber.NewError(fiber.StatusBadRequest, "iss and login_hint are required")
	}

	p, err := h.store.FindPlatform(c.Context(), iss, param(c, "client_id"))
	if errors.Is(err, ErrNotFound) {
		h.debugf("   rejected: no platforms row for iss %q", iss)
		return fiber.NewError(fiber.StatusBadRequest, "unknown issuer")
	}
	if err != nil {
		return err
	}
	h.debugf("   matched platform #%d (client_id=%s, deployment_id=%s)", p.ID, p.ClientID, p.DeploymentID)

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
	h.debugf("   saved state=%s nonce=%s (valid for %s)", short(state), short(nonce), stateTTL)

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

	h.debugf("-> redirecting browser to LMS auth URL %s://%s%s%s",
		authURL.Scheme, authURL.Host, authURL.Path, formatParams(existing))
	return c.Redirect().Status(fiber.StatusFound).To(authURL.String())
}
