// Package lti implements the LTI 1.3 launch: OIDC login initiation, id_token
// verification, and the claim checks the spec requires.
package lti

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// Claim names from the LTI 1.3 spec.
const (
	MessageTypeLaunch  = "LtiResourceLinkRequest"
	SupportedVersion   = "1.3.0"
	clockSkewAllowance = time.Minute
)

// ErrInvalidLaunch wraps every claim failure, so the handler can return 401
// with errors.Is while logs keep the specific reason.
var ErrInvalidLaunch = errors.New("invalid launch")

// Claims is the part of the id_token payload this tool uses.
type Claims struct {
	Issuer          string   `json:"iss"`
	Subject         string   `json:"sub"`
	Audience        Audience `json:"aud"`
	AuthorizedParty string   `json:"azp"`
	ExpiresAt       int64    `json:"exp"`
	IssuedAt        int64    `json:"iat"`
	Nonce           string   `json:"nonce"`
	Name            string   `json:"name"`

	DeploymentID string   `json:"https://purl.imsglobal.org/spec/lti/claim/deployment_id"`
	MessageType  string   `json:"https://purl.imsglobal.org/spec/lti/claim/message_type"`
	Version      string   `json:"https://purl.imsglobal.org/spec/lti/claim/version"`
	Roles        []string `json:"https://purl.imsglobal.org/spec/lti/claim/roles"`
	Context      Context  `json:"https://purl.imsglobal.org/spec/lti/claim/context"`
}

// Context is the course the launch came from.
type Context struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Title string `json:"title"`
}

// Audience is the aud claim. JWTs allow either one string or an array.
type Audience []string

func (a *Audience) UnmarshalJSON(b []byte) error {
	var one string
	if err := json.Unmarshal(b, &one); err == nil {
		*a = Audience{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("aud must be a string or an array of strings")
	}
	*a = many
	return nil
}

// Expected holds what the launch must match: the registered platform, the
// nonce saved at login, and whether that nonce was already used.
type Expected struct {
	Issuer       string
	ClientID     string
	DeploymentID string
	Nonce        string
	NonceUsed    bool
	Now          time.Time
}

// Validate checks every claim the LTI 1.3 security framework requires. The
// signature must already be verified; this only looks at the payload, so it
// is a pure function and easy to table-test.
func (c Claims) Validate(want Expected) error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrInvalidLaunch, fmt.Sprintf(format, args...))
	}
	now := want.Now.Unix()
	skew := int64(clockSkewAllowance.Seconds())

	switch {
	case c.Issuer != want.Issuer:
		return fail("iss %q is not the registered platform", c.Issuer)
	case !slices.Contains(c.Audience, want.ClientID):
		return fail("aud does not include our client_id")
	case len(c.Audience) > 1 && c.AuthorizedParty != want.ClientID:
		return fail("azp must be our client_id when aud has several values")
	case c.ExpiresAt == 0 || now > c.ExpiresAt+skew:
		return fail("token expired")
	case c.IssuedAt == 0 || c.IssuedAt > now+skew:
		return fail("iat is missing or in the future")
	case c.Nonce == "" || c.Nonce != want.Nonce:
		return fail("nonce does not match the login request")
	case want.NonceUsed:
		return fail("nonce was already used")
	case c.DeploymentID != want.DeploymentID:
		return fail("deployment_id %q is not registered", c.DeploymentID)
	case c.MessageType != MessageTypeLaunch:
		return fail("message_type %q is not supported", c.MessageType)
	case c.Version != SupportedVersion:
		return fail("LTI version %q is not supported", c.Version)
	case c.Subject == "":
		return fail("sub is required")
	case c.Context.ID == "":
		return fail("context claim is required: launch from inside a course")
	}
	return nil
}

// instructorRoles are the LIS roles allowed to edit decks.
var instructorRoles = []string{
	"http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor",
	"http://purl.imsglobal.org/vocab/lis/v2/membership#ContentDeveloper",
	"http://purl.imsglobal.org/vocab/lis/v2/membership#Administrator",
	"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Administrator",
	"http://purl.imsglobal.org/vocab/lis/v2/system/person#Administrator",
}

// IsInstructor reports whether the user may edit. Teaching assistants
// (membership/Instructor#TeachingAssistant) count as instructors.
func (c Claims) IsInstructor() bool {
	for _, r := range c.Roles {
		if slices.Contains(instructorRoles, r) ||
			strings.HasPrefix(r, "http://purl.imsglobal.org/vocab/lis/v2/membership/Instructor#") {
			return true
		}
	}
	return false
}
