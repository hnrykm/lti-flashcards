# LTI Flashcards

An LTI 1.3 tool that a course launches into. Instructors create decks and flashcards for their
course; students study them.

- **API:** Go 1.27, Fiber v3, PostgreSQL 16 (pgx v5), lestrrat-go/jwx v3
- **UI:** Angular 22 (standalone components, signals, zoneless) and Bootstrap 5.3
- **Packaging:** one Docker image holds the Go API and the built Angular files; `docker compose`
  adds Postgres and a local Moodle to launch from

## Run it

```sh
echo "127.0.0.1 moodle" | sudo tee -a /etc/hosts   # once; see "Why moodle:8080" below
docker compose up -d --build
./scripts/moodle-setup.sh                          # registers the tool in Moodle
```

Then log in to **http://moodle:8080** as `admin` / `Admin123!` (instructor) or `student` /
`Student123!` and open the **Flashcards** activity in *Biology 101*. The setup script prints the
direct link.

| URL | What |
|---|---|
| http://localhost:3000 | The app (needs a launch) |
| http://localhost:3000/api/docs | Swagger UI for the API |
| http://moodle:8080 | Local Moodle 5.3 (first start takes about a minute) |

`SESSION_SECRET` must be at least 32 bytes. Compose uses a dev default unless you set it in `.env`.

### Without Docker

```sh
docker compose up -d db
export DATABASE_URL=postgres://flashcards:flashcards@localhost:5432/flashcards
export SESSION_SECRET=$(openssl rand -hex 32)
go run ./cmd/api                     # API on :3000
cd web && npm install && npm start   # Angular on :4200, proxies /api and /lti to :3000
```

Angular 22 needs Node 22.22.3+ or 24.15+.

### Try the API with curl

`cmd/devtoken` prints a session token for the seeded dev course, so you can skip the launch:

```sh
TOKEN=$(go run ./cmd/devtoken -instructor)   # same SESSION_SECRET as the API
curl -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
     -d '{"name":"Chapter 3"}' localhost:3000/api/decks
```

### Tests

```sh
go test ./...                       # claim validation table + full launch against a fake platform
cd web && npx ng test --watch=false
```

## How a launch works

```mermaid
sequenceDiagram
    participant B as Browser
    participant L as LMS (Moodle)
    participant T as Tool (Go)
    participant DB as Postgres

    B->>L: Open the Flashcards activity
    L->>B: Auto-submit form
    B->>T: POST /lti/login (iss, login_hint, lti_message_hint)
    T->>DB: Save random state + nonce (10 min)
    T->>B: 302 to LMS auth URL (state, nonce, client_id, redirect_uri)
    B->>L: GET /mod/lti/auth.php
    L->>B: Auto-submit form with a signed id_token
    B->>T: POST /lti/launch (id_token, state)
    T->>DB: Load state → nonce, platform
    T->>L: GET JWKS (cached, refetched on unknown kid)
    T->>T: Verify signature, check iss, aud/azp, exp, iat, nonce,<br/>deployment_id, message_type, version
    T->>DB: Mark nonce used; upsert course from the context claim
    T->>B: 302 /#token=<1-hour session JWT>
    B->>T: Angular calls /api/* with Authorization: Bearer
```

## Endpoints

| Method | Path | Who | Success | Errors |
|---|---|---|---|---|
| GET, POST | `/lti/login` | Platform | 302 to platform | 400 unknown issuer |
| POST | `/lti/launch` | Platform | 302 to `/#token=...` | 401 invalid token or state |
| GET | `/api/health` | Anyone | 200 | 503 if DB is down |
| GET | `/api/me` | Course members | 200 session | 401 |
| GET | `/api/decks` | Course members | 200 decks for the course | 401 |
| POST | `/api/decks` | Instructors | 201 deck | 400, 401, 403 |
| GET | `/api/decks/:id` | Course members | 200 deck | 401, 404 |
| PUT | `/api/decks/:id` | Instructors | 200 deck | 400, 401, 403, 404 |
| DELETE | `/api/decks/:id` | Instructors | 204 | 401, 403, 404 |
| GET | `/api/decks/:id/cards` | Course members | 200 cards | 401, 404 |
| POST | `/api/decks/:id/cards` | Instructors | 201 card | 400, 401, 403, 404 |
| PUT | `/api/cards/:id` | Instructors | 200 card | 400, 401, 403, 404 |
| DELETE | `/api/cards/:id` | Instructors | 204 | 401, 403, 404 |

Every error is JSON: `{"error": "..."}`. The full spec is at `/api/openapi.yaml`.

## Design decisions

- **Validate every claim.** A valid signature only proves the LMS signed *some* token. `aud` stops
  a token meant for another tool from being replayed here, `nonce` plus the `used` flag stops replays
  of this one, and `deployment_id` stops another deployment on the same LMS from reaching our data.
  The checks live in `internal/lti/claims.go` as a pure function, so they're table-tested without keys.
- **Scope every query by course.** The course ID comes only from the signed session token, never the
  request. A deck or card in another course returns 404, not 403, so IDs don't leak.
- **Role from the launch.** Instructor, content developer, admin and TA roles can edit; everyone
  else reads. The API enforces it with middleware; the UI just hides the buttons.
- **Own session token after the launch.** The LMS's id_token is checked once; after that the tool
  issues a 1-hour HS256 JWT so API calls don't depend on the LMS.
- **One container.** Fiber serves the Angular build, so there's one image, one origin and no CORS.

## Shortcuts, and what I'd add next

- **Token in the URL fragment.** Simple for a demo, and fragments never reach server logs. In
  production I'd redirect with a one-time code and exchange it for the token, because LMS iframes
  often block third-party cookies and URLs end up in browser history.
- **One platform row instead of Dynamic Registration.** `scripts/moodle-setup.sh` stands in for
  LTI Dynamic Registration, which would let an admin install the tool from a URL.
- **AGS** (grade passback) for study scores, **NRPS** (roster sync) for progress per student, and
  **Deep Linking** so instructors can place a single deck in a module.
- **Platform-specific quirks** isolated behind one adapter per LMS vendor.
- Migrations with a tool such as goose instead of Postgres init scripts.

## Why moodle:8080

The browser and the app container must both reach Moodle at the address Moodle calls itself (its
`wwwroot`, which is also the token issuer). Moodle redirects any other host name, so `localhost:8080`
fails from inside the app container. `moodle` resolves to Moodle on the Docker network and, with the
hosts entry, to 127.0.0.1 in your browser.

To launch from [saltire.lti.app](https://saltire.lti.app) instead, register the tool there with login
URL `http://localhost:3000/lti/login` and redirect URL `http://localhost:3000/lti/launch`, then add a
row to `platforms` with the issuer, client ID, deployment ID, auth URL and JWKS URL it shows.

## Layout

```
cmd/api/main.go             config, pool, routes, serves the Angular build
cmd/devtoken/               session token for curl testing
internal/db/                pgxpool
internal/lti/               login, launch, claim checks (tested), JWKS cache, SQL
internal/auth/              session JWT and bearer middleware
internal/flashcards/        models, SQL and CRUD handlers
internal/apidocs/           OpenAPI spec and Swagger UI
db/migrations/              schema and dev seed (run by Postgres on first start)
scripts/moodle-setup.*      registers the tool in the local Moodle
web/                        Angular app
```
