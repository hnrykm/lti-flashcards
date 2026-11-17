# CLAUDE.md

This is a flashcards tool that runs inside an LMS such as Moodle, using LTI 1.3. Teachers make decks of cards for their course, and students study them. The backend is Go with Fiber, the frontend is Angular, and the database is Postgres. Everything runs from one Docker container.

## Running it

```sh
docker compose up -d --build     # starts the app, the database and Moodle
./scripts/moodle-setup.sh        # connects the app to Moodle (safe to run again)
```

Then open http://moodle:8080 and log in as `admin` / `Admin123!` (teacher) or `student` / `Student123!`.

To work without Docker, start only the database with `docker compose up -d db`, run `go run ./cmd/api`, and run `npm start` in `web/`. To call the API without going through Moodle, get a token with `go run ./cmd/devtoken -instructor`.

## Checking your work

```sh
go build ./... && go vet ./... && go test ./...
cd web && npx ng build && npx ng test --watch=false
```

## Where things are

- `cmd/api/main.go` starts the server and connects all the parts.
- `internal/lti/` handles the Moodle launch and checks that it's genuine.
- `internal/auth/` creates and checks the app's own login token.
- `internal/flashcards/` holds the decks and cards: data types, database queries and API routes.
- `web/src/app/` is the Angular app. Each file is one screen or service.
- `db/migrations/` is the database schema. It only runs on a fresh database. To apply changes, run `docker compose down -v`, which deletes all data.

## Rules to follow

- **Keep courses separate.** Every database query must filter by the course from the user's token, never by a course ID sent in the request. If something belongs to another course, return "not found".
- **Only teachers can make changes.** Protect every create, edit and delete route with `auth.RequireInstructor`. Hiding buttons in the UI isn't enough.
- **Check input in two places.** Check it in Go to give a clear error message, and keep the database limits as a backup. Deck names can be up to 100 characters and card sides up to 1000.
- **Don't weaken the launch checks** in `internal/lti/claims.go`. If you add a check, add a test for it.
- **Keep error messages safe.** Expected problems use `fiber.NewError` and the user sees the message. Unexpected errors are logged, and the user only sees "internal server error".
- **Keep the API docs current.** Update `internal/apidocs/openapi.yaml` when you change an endpoint.
- **In Angular,** store state in signals and use `inject()`. Turn errors into plain sentences with `errorMessage()`.

## Accessibility

The app must work for keyboard and screen reader users. When you change the UI:

- Give every input a visible label.
- If several buttons share the same text, give each a unique name ("Delete deck Biology").
- Announce loading and errors to screen readers.
- Confirm deletes on the page, not in a popup.
- Give every page a title.
- Respect the reduced-motion setting.

Two known problems to fix rather than copy: keyboard focus gets lost when a button is replaced (Delete becomes Confirm), and moving to a new page isn't announced.

## Common problems

- Moodle must be at `http://moodle:8080`, not `localhost`. Add `127.0.0.1 moodle` to `/etc/hosts`.
- `SESSION_SECRET` must be at least 32 characters, or the app won't start.
- The URLs registered in Moodle must exactly match the ones the app uses.

## Git
- Use one branch per feature and merge it into `main`. Don't push unless asked.
