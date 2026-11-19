package lti

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"slices"
	"strings"

	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
)

// WithDebug turns on step-by-step logging of what the LMS sends and what the
// tool sends back during login and launch. It logs names, emails and course
// details from the LMS, so leave it off in production. Secrets (state, nonce,
// the raw id_token and the session token) are shortened, never logged whole.
func (h *Handler) WithDebug(on bool) *Handler {
	h.debug = on
	return h
}

func (h *Handler) debugf(format string, args ...any) {
	if h.debug {
		log.Printf("lti debug: "+format, args...)
	}
}

// short keeps the first few characters of a secret so log lines can be
// matched up without the full value being usable.
func short(s string) string {
	if len(s) <= 8 {
		return s
	}
	return fmt.Sprintf("%s…(%d chars)", s[:8], len(s))
}

// formatParams lists query or form values one per line, sorted, with the
// one-time codes shortened.
func formatParams(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var b strings.Builder
	for _, k := range keys {
		val := v.Get(k)
		if k == "state" || k == "nonce" {
			val = short(val)
		}
		fmt.Fprintf(&b, "\n    %s = %s", k, val)
	}
	return b.String()
}

// prettyJSON indents a JSON payload for the log, with the nonce shortened.
func prettyJSON(b []byte) string {
	var claims map[string]any
	if err := json.Unmarshal(b, &claims); err == nil {
		if n, ok := claims["nonce"].(string); ok {
			claims["nonce"] = short(n)
		}
		if redacted, err := json.Marshal(claims); err == nil {
			b = redacted
		}
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "    ", "  "); err != nil {
		return string(b)
	}
	return "    " + out.String()
}

// tokenHeader returns the id_token's protected header (alg, kid, typ) as JSON,
// read without verifying anything.
func tokenHeader(token []byte) string {
	msg, err := jws.Parse(token)
	if err != nil || len(msg.Signatures()) == 0 {
		return "(unreadable)"
	}
	b, err := json.Marshal(msg.Signatures()[0].ProtectedHeaders())
	if err != nil {
		return "(unreadable)"
	}
	return string(b)
}

// keyIDs lists the kid of every key in a JWKS.
func keyIDs(set jwk.Set) []string {
	ids := make([]string, 0, set.Len())
	for i := 0; i < set.Len(); i++ {
		if k, ok := set.Key(i); ok {
			kid, _ := k.KeyID()
			ids = append(ids, kid)
		}
	}
	return ids
}
