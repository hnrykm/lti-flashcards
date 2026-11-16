CREATE TABLE platforms (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  issuer         TEXT NOT NULL,
  client_id      TEXT NOT NULL,
  deployment_id  TEXT NOT NULL,
  auth_login_url TEXT NOT NULL,
  jwks_url       TEXT NOT NULL,
  UNIQUE (issuer, client_id)
);

CREATE TABLE oidc_states (
  state          TEXT PRIMARY KEY,
  nonce          TEXT NOT NULL UNIQUE,
  platform_id    BIGINT NOT NULL REFERENCES platforms(id),
  used           BOOLEAN NOT NULL DEFAULT false,
  expires_at     TIMESTAMPTZ NOT NULL
);

CREATE TABLE courses (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  platform_id    BIGINT NOT NULL REFERENCES platforms(id),
  context_id     TEXT NOT NULL, -- LMS course id from the launch
  title          TEXT NOT NULL DEFAULT '',
  UNIQUE (platform_id, context_id)
);

CREATE TABLE decks (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  course_id      BIGINT NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
  name           TEXT NOT NULL CHECK (char_length(name) BETWEEN 1 and 100),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_decks_course ON decks (course_id);

CREATE TABLE cards(
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  deck_id        BIGINT NOT NULL REFERENCES decks(id) ON DELETE CASCADE,
  front          TEXT NOT NULL CHECK (char_length(front) BETWEEN 1 AND 1000),
  back           TEXT NOT NULL CHECK (char_length(back) BETWEEN 1 and 1000),
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_cards_deck ON cards (deck_id);