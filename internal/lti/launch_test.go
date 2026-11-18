package lti

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jws"

	"github.com/hnrykm/lti-flashcards/internal/auth"
)

// fakeStore is an in-memory Store with one registered platform.
type fakeStore struct {
	platform Platform
	states   map[string]*LoginState
	courses  map[string]int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		platform: Platform{
			ID: 1, Issuer: "https://lms.example", ClientID: "client-1", DeploymentID: "deploy-1",
			AuthLoginURL: "https://lms.example/auth?tenant=a", JWKSURL: "https://lms.example/jwks",
		},
		states:  map[string]*LoginState{},
		courses: map[string]int64{},
	}
}

func (f *fakeStore) FindPlatform(_ context.Context, iss, clientID string) (Platform, error) {
	if iss != f.platform.Issuer || (clientID != "" && clientID != f.platform.ClientID) {
		return Platform{}, ErrNotFound
	}
	return f.platform, nil
}

func (f *fakeStore) GetPlatform(context.Context, int64) (Platform, error) { return f.platform, nil }

func (f *fakeStore) SaveState(_ context.Context, state, nonce string, pid int64, _ time.Time) error {
	f.states[state] = &LoginState{Nonce: nonce, PlatformID: pid}
	return nil
}

func (f *fakeStore) LoadState(_ context.Context, state string) (LoginState, error) {
	st, ok := f.states[state]
	if !ok {
		return LoginState{}, ErrNotFound
	}
	return *st, nil
}

func (f *fakeStore) MarkStateUsed(_ context.Context, state string) error {
	st, ok := f.states[state]
	if !ok || st.Used {
		return ErrNotFound
	}
	st.Used = true
	return nil
}

func (f *fakeStore) UpsertCourse(_ context.Context, _ int64, contextID, _ string) (int64, error) {
	if id, ok := f.courses[contextID]; ok {
		return id, nil
	}
	id := int64(len(f.courses) + 1)
	f.courses[contextID] = id
	return id, nil
}

// staticKeys serves one fixed JWKS.
type staticKeys struct{ set jwk.Set }

func (k staticKeys) Get(context.Context, string) (jwk.Set, error)     { return k.set, nil }
func (k staticKeys) Refresh(context.Context, string) (jwk.Set, error) { return k.set, nil }

// platformKey makes an RSA signing key and the public JWKS for it.
func platformKey(t *testing.T, kid string) (jwk.Key, jwk.Set) {
	t.Helper()
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	priv, err := jwk.Import(raw)
	if err != nil {
		t.Fatal(err)
	}
	_ = priv.Set(jwk.KeyIDKey, kid)
	_ = priv.Set(jwk.AlgorithmKey, jwa.RS256())
	pub, err := priv.PublicKey()
	if err != nil {
		t.Fatal(err)
	}
	set := jwk.NewSet()
	_ = set.AddKey(pub)
	return priv, set
}

type launchTest struct {
	t      *testing.T
	app    *fiber.App
	store  *fakeStore
	key    jwk.Key
	signer *auth.Signer
}

func newLaunchTest(t *testing.T) *launchTest {
	key, set := platformKey(t, "key-1")
	signer, err := auth.NewSigner([]byte(strings.Repeat("s", 32)))
	if err != nil {
		t.Fatal(err)
	}
	store := newFakeStore()
	h := NewHandler(store, staticKeys{set}, signer)
	h.now = func() time.Time { return testNow }
	app := fiber.New()
	h.Register(app)
	return &launchTest{t: t, app: app, store: store, key: key, signer: signer}
}

func (lt *launchTest) do(req *http.Request) *http.Response {
	lt.t.Helper()
	resp, err := lt.app.Test(req)
	if err != nil {
		lt.t.Fatal(err)
	}
	return resp
}

// login runs login initiation and returns the state and nonce the tool sent
// to the platform's auth URL.
func (lt *launchTest) login() (state, nonce string) {
	lt.t.Helper()
	form := url.Values{"iss": {"https://lms.example"}, "login_hint": {"u1"}, "target_link_uri": {"http://tool/"}}
	req := httptest.NewRequest("POST", "/lti/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := lt.do(req)
	if resp.StatusCode != http.StatusFound {
		lt.t.Fatalf("login: status %d", resp.StatusCode)
	}
	loc, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		lt.t.Fatal(err)
	}
	q := loc.Query()
	if loc.Host != "lms.example" || q.Get("tenant") != "a" || q.Get("client_id") != "client-1" ||
		q.Get("response_mode") != "form_post" || q.Get("login_hint") != "u1" {
		lt.t.Fatalf("login redirect is wrong: %s", loc)
	}
	return q.Get("state"), q.Get("nonce")
}

// idToken signs claims the way the platform would.
func (lt *launchTest) idToken(key jwk.Key, c Claims) string {
	lt.t.Helper()
	payload, err := json.Marshal(c)
	if err != nil {
		lt.t.Fatal(err)
	}
	signed, err := jws.Sign(payload, jws.WithKey(jwa.RS256(), key))
	if err != nil {
		lt.t.Fatal(err)
	}
	return string(signed)
}

func (lt *launchTest) launch(idToken, state string) *http.Response {
	lt.t.Helper()
	form := url.Values{"id_token": {idToken}, "state": {state}}
	req := httptest.NewRequest("POST", "/lti/launch", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return lt.do(req)
}

func instructorClaims(nonce string) Claims {
	c := validClaims()
	c.Nonce = nonce
	c.Roles = []string{"http://purl.imsglobal.org/vocab/lis/v2/membership#Instructor"}
	return c
}

func TestLaunchIssuesSessionForCourse(t *testing.T) {
	lt := newLaunchTest(t)
	state, nonce := lt.login()

	resp := lt.launch(lt.idToken(lt.key, instructorClaims(nonce)), state)
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("launch: status %d", resp.StatusCode)
	}
	token, ok := strings.CutPrefix(resp.Header.Get("Location"), "/#token=")
	if !ok {
		t.Fatalf("launch redirect is wrong: %s", resp.Header.Get("Location"))
	}
	sess, err := lt.signer.Verify(token, testNow)
	if err != nil {
		t.Fatal(err)
	}
	if sess.CourseID != lt.store.courses["course-1"] || sess.UserID != "user-1" || !sess.Instructor {
		t.Fatalf("unexpected session %+v", sess)
	}
}

func TestLaunchRejects(t *testing.T) {
	t.Run("replayed id_token", func(t *testing.T) {
		lt := newLaunchTest(t)
		state, nonce := lt.login()
		tok := lt.idToken(lt.key, instructorClaims(nonce))
		if resp := lt.launch(tok, state); resp.StatusCode != http.StatusFound {
			t.Fatalf("first launch: status %d", resp.StatusCode)
		}
		if resp := lt.launch(tok, state); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("replay: status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("signed by another key", func(t *testing.T) {
		lt := newLaunchTest(t)
		state, nonce := lt.login()
		forged, _ := platformKey(t, "key-1") // same kid, different key
		if resp := lt.launch(lt.idToken(forged, instructorClaims(nonce)), state); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("unknown state", func(t *testing.T) {
		lt := newLaunchTest(t)
		_, nonce := lt.login()
		if resp := lt.launch(lt.idToken(lt.key, instructorClaims(nonce)), "made-up"); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", resp.StatusCode)
		}
	})

	t.Run("nonce from another login", func(t *testing.T) {
		lt := newLaunchTest(t)
		state, _ := lt.login()
		_, otherNonce := lt.login()
		if resp := lt.launch(lt.idToken(lt.key, instructorClaims(otherNonce)), state); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", resp.StatusCode)
		}
	})
}

func TestLoginUnknownIssuer(t *testing.T) {
	lt := newLaunchTest(t)
	req := httptest.NewRequest("GET", "/lti/login?iss=https://unknown.example&login_hint=u1", nil)
	if resp := lt.do(req); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d, want 400", resp.StatusCode)
	}
}
