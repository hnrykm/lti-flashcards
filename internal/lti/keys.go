package lti

import (
	"context"
	"fmt"

	"github.com/lestrrat-go/httprc/v3"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"
)

// KeySets returns a platform's public signing keys (its JWKS).
type KeySets interface {
	Get(ctx context.Context, jwksURL string) (jwk.Set, error)
	Refresh(ctx context.Context, jwksURL string) (jwk.Set, error)
}

// JWKSCache fetches each platform's JWKS once and refreshes it in the
// background, so a launch doesn't wait on an HTTP call to the LMS.
type JWKSCache struct {
	cache *jwk.Cache
}

func NewJWKSCache(ctx context.Context) (*JWKSCache, error) {
	c, err := jwk.NewCache(ctx, httprc.NewClient())
	if err != nil {
		return nil, err
	}
	return &JWKSCache{cache: c}, nil
}

// Get registers the URL on first use, then serves the cached set.
func (k *JWKSCache) Get(ctx context.Context, u string) (jwk.Set, error) {
	if !k.cache.IsRegistered(ctx, u) {
		// Another request may register the same URL at the same moment; that
		// error is harmless as long as the URL ends up registered.
		if err := k.cache.Register(ctx, u); err != nil && !k.cache.IsRegistered(ctx, u) {
			return nil, fmt.Errorf("fetch JWKS %s: %w", u, err)
		}
	}
	return k.cache.Lookup(ctx, u)
}

// Refresh refetches now. Used when a token's kid isn't in the cached set,
// which is what happens right after a platform rotates its keys.
func (k *JWKSCache) Refresh(ctx context.Context, u string) (jwk.Set, error) {
	return k.cache.Refresh(ctx, u)
}

// verifySignature checks the id_token against the platform's keys and returns
// the verified payload. If the token names a key ID we haven't cached, the
// platform probably rotated keys, so we refetch once. We don't refetch for
// other failures, or anyone could make us hammer the LMS with bad tokens.
// debugf logs each step when launch debugging is on.
func verifySignature(ctx context.Context, keys KeySets, jwksURL string, token []byte, debugf func(string, ...any)) ([]byte, error) {
	set, err := keys.Get(ctx, jwksURL)
	if err != nil {
		return nil, err
	}
	debugf("   platform keys from %s: kids %v", jwksURL, keyIDs(set))
	if kid := tokenKeyID(token); kid != "" {
		if _, ok := set.LookupKeyID(kid); !ok {
			debugf("   kid %q not in cached keys (key rotation?), refetching JWKS", kid)
			if fresh, err := keys.Refresh(ctx, jwksURL); err == nil {
				set = fresh
				debugf("   refetched keys: kids %v", keyIDs(set))
			}
		}
	}
	// WithKeySet requires the token's kid and alg to match a key in the set,
	// so a token can't choose a key or algorithm (like "none" or HS256).
	return jws.Verify(token, jws.WithKeySet(set))
}

// tokenKeyID reads the kid from the JWS header without verifying anything.
func tokenKeyID(token []byte) string {
	msg, err := jws.Parse(token)
	if err != nil || len(msg.Signatures()) == 0 {
		return ""
	}
	kid, _ := msg.Signatures()[0].ProtectedHeaders().KeyID()
	return kid
}
