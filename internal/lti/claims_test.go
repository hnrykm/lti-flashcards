package lti

import (
	"encoding/json"
	"errors"
	"testing"
	"time"
)

var testNow = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func validClaims() Claims {
	return Claims{
		Issuer:       "https://lms.example",
		Subject:      "user-1",
		Audience:     Audience{"client-1"},
		ExpiresAt:    testNow.Add(5 * time.Minute).Unix(),
		IssuedAt:     testNow.Unix(),
		Nonce:        "nonce-1",
		DeploymentID: "deploy-1",
		MessageType:  MessageTypeLaunch,
		Version:      SupportedVersion,
		Context:      Context{ID: "course-1", Title: "Biology 101"},
	}
}

func expected() Expected {
	return Expected{
		Issuer:       "https://lms.example",
		ClientID:     "client-1",
		DeploymentID: "deploy-1",
		Nonce:        "nonce-1",
		Now:          testNow,
	}
}

func TestClaimsValidate(t *testing.T) {
	tests := []struct {
		name   string
		claims func(*Claims)
		want   func(*Expected)
		ok     bool
	}{
		{name: "good token", ok: true},
		{name: "wrong aud", claims: func(c *Claims) { c.Audience = Audience{"someone-else"} }},
		{name: "expired", claims: func(c *Claims) { c.ExpiresAt = testNow.Add(-2 * time.Minute).Unix() }},
		{name: "reused nonce", want: func(e *Expected) { e.NonceUsed = true }},
		{
			name:   "expired within clock skew is accepted",
			claims: func(c *Claims) { c.ExpiresAt = testNow.Add(-30 * time.Second).Unix() },
			ok:     true,
		},
		{name: "wrong issuer", claims: func(c *Claims) { c.Issuer = "https://evil.example" }},
		{name: "nonce mismatch", claims: func(c *Claims) { c.Nonce = "other-nonce" }},
		{name: "missing nonce", claims: func(c *Claims) { c.Nonce = "" }},
		{name: "iat in the future", claims: func(c *Claims) { c.IssuedAt = testNow.Add(10 * time.Minute).Unix() }},
		{name: "wrong deployment", claims: func(c *Claims) { c.DeploymentID = "deploy-2" }},
		{name: "deep linking message", claims: func(c *Claims) { c.MessageType = "LtiDeepLinkingRequest" }},
		{name: "LTI 1.1 version", claims: func(c *Claims) { c.Version = "1.1" }},
		{name: "no context", claims: func(c *Claims) { c.Context = Context{} }},
		{
			name:   "several audiences need azp",
			claims: func(c *Claims) { c.Audience = Audience{"client-1", "client-2"} },
		},
		{
			name: "several audiences with azp",
			claims: func(c *Claims) {
				c.Audience = Audience{"client-1", "client-2"}
				c.AuthorizedParty = "client-1"
			},
			ok: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, want := validClaims(), expected()
			if tt.claims != nil {
				tt.claims(&c)
			}
			if tt.want != nil {
				tt.want(&want)
			}

			err := c.Validate(want)
			if tt.ok && err != nil {
				t.Fatalf("expected valid, got %v", err)
			}
			if !tt.ok && !errors.Is(err, ErrInvalidLaunch) {
				t.Fatalf("expected ErrInvalidLaunch, got %v", err)
			}
		})
	}
}

func TestAudienceAcceptsStringOrArray(t *testing.T) {
	for _, raw := range []string{`{"aud":"client-1"}`, `{"aud":["client-1","x"]}`} {
		var c Claims
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			t.Fatalf("%s: %v", raw, err)
		}
		if c.Audience[0] != "client-1" {
			t.Fatalf("%s: got %v", raw, c.Audience)
		}
	}
}

func TestIsInstructor(t *testing.T) {
	tests := map[string]bool{
		"http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor":                   true,
		"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Administrator":        true,
		"http://purl.imsglobal.org/vocab/lis/v2/membership/Instructor#TeachingAssistant": true,
		"http://purl.imsglobal.org/vocab/lis/v2/membership#Learner":                      false,
		"http://purl.imsglobal.org/vocab/lis/v2/institution/person#Student":              false,
	}
	for role, want := range tests {
		if got := (Claims{Roles: []string{role}}).IsInstructor(); got != want {
			t.Errorf("%s: got %v, want %v", role, got, want)
		}
	}
}
