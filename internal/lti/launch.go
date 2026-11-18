package lti

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/gofiber/fiber/v3"

	"github.com/hnrykm/lti-flashcards/internal/auth"
)

// launch is the end of the flow. The platform's auth page form-posts
// id_token and state here. We check the state, verify the signature with the
// platform's JWKS, check every claim, mark the nonce used, upsert the course,
// and hand Angular our own session token in the URL fragment.
func (h *Handler) launch(c fiber.Ctx) error {
	ctx := c.Context()
	idToken, state := c.FormValue("id_token"), c.FormValue("state")
	if idToken == "" || state == "" {
		return unauthorized("id_token and state are required")
	}

	// 1. The state must be one we issued at login, not expired.
	st, err := h.store.LoadState(ctx, state)
	if errors.Is(err, ErrNotFound) {
		return unauthorized("unknown or expired state")
	}
	if err != nil {
		return err
	}
	p, err := h.store.GetPlatform(ctx, st.PlatformID)
	if err != nil {
		return err
	}

	// 2. The signature must verify against the platform's published keys.
	payload, err := verifySignature(ctx, h.keys, p.JWKSURL, []byte(idToken))
	if err != nil {
		log.Printf("lti launch: signature: %v", err)
		return unauthorized("id_token signature is invalid")
	}

	// 3. Every claim must match what we registered and what we sent at login.
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return unauthorized("id_token payload is not valid JSON claims")
	}
	err = claims.Validate(Expected{
		Issuer:       p.Issuer,
		ClientID:     p.ClientID,
		DeploymentID: p.DeploymentID,
		Nonce:        st.Nonce,
		NonceUsed:    st.Used,
		Now:          h.now(),
	})
	if err != nil {
		log.Printf("lti launch: %v", err)
		return unauthorized(err.Error())
	}

	// 4. Use the nonce up. This only succeeds once, even if the same id_token
	// is posted twice at the same moment.
	if err := h.store.MarkStateUsed(ctx, state); errors.Is(err, ErrNotFound) {
		return unauthorized("nonce was already used")
	} else if err != nil {
		return err
	}

	// 5. Map the LMS course to ours and issue a session for it.
	courseID, err := h.store.UpsertCourse(ctx, p.ID, claims.Context.ID, claims.Context.Title)
	if err != nil {
		return err
	}
	token, err := h.signer.Issue(auth.Session{
		CourseID:   courseID,
		UserID:     claims.Subject,
		Name:       claims.Name,
		Instructor: claims.IsInstructor(),
	}, h.now())
	if err != nil {
		return err
	}

	// The fragment (#...) never reaches the server or its logs. Fine for a
	// demo; production would use a one-time code exchanged for a token.
	return c.Redirect().Status(fiber.StatusFound).To("/#token=" + token)
}

func unauthorized(msg string) error {
	return fiber.NewError(fiber.StatusUnauthorized, msg)
}
