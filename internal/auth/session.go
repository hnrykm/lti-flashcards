// Package auth issues and checks the app's own session tokens.
//
// After an LTI launch is validated, the tool stops trusting the LMS's id_token
// and issues its own short-lived HS256 JWT. Angular sends it as
// "Authorization: Bearer <token>" on every API call.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jws"
)

// SessionTTL is how long a session token stays valid after a launch.
const SessionTTL = time.Hour

var ErrInvalidToken = errors.New("invalid or expired session token")

// Session is what the tool knows about the user after a launch. It is the
// payload of the session JWT.
type Session struct {
	CourseID   int64  `json:"course_id"`
	UserID     string `json:"sub"`  // the LMS user id (the launch's sub claim)
	Name       string `json:"name"` // display name, if the LMS sent one
	Instructor bool   `json:"instructor"`
	IssuedAt   int64  `json:"iat"`
	ExpiresAt  int64  `json:"exp"`
}

// Signer signs and verifies session tokens with one shared secret.
type Signer struct {
	secret []byte
}

// NewSigner needs at least 32 bytes of secret: HS256 is only as strong as its key.
func NewSigner(secret []byte) (*Signer, error) {
	if len(secret) < 32 {
		return nil, errors.New("SESSION_SECRET must be at least 32 bytes")
	}
	return &Signer{secret: secret}, nil
}

// Issue sets the issued/expiry times and returns a signed token.
func (s *Signer) Issue(sess Session, now time.Time) (string, error) {
	sess.IssuedAt = now.Unix()
	sess.ExpiresAt = now.Add(SessionTTL).Unix()
	payload, err := json.Marshal(sess)
	if err != nil {
		return "", err
	}
	signed, err := jws.Sign(payload, jws.WithKey(jwa.HS256(), s.secret))
	if err != nil {
		return "", fmt.Errorf("sign session: %w", err)
	}
	return string(signed), nil
}

// Verify checks the signature (HS256 only, so a token can't pick a weaker
// algorithm) and the expiry, then returns the session.
func (s *Signer) Verify(token string, now time.Time) (Session, error) {
	payload, err := jws.Verify([]byte(token), jws.WithKey(jwa.HS256(), s.secret))
	if err != nil {
		return Session{}, ErrInvalidToken
	}
	var sess Session
	if err := json.Unmarshal(payload, &sess); err != nil {
		return Session{}, ErrInvalidToken
	}
	if sess.CourseID <= 0 || now.Unix() >= sess.ExpiresAt {
		return Session{}, ErrInvalidToken
	}
	return sess, nil
}
