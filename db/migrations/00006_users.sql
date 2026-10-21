-- +goose Up
CREATE TABLE IF NOT EXISTS users (
  id BIGSERIAL PRIMARY KEY,
  username TEXT NOT NULL UNIQUE,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL DEFAULT '',
  role TEXT NOT NULL DEFAULT 'user',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS users_username ON users(username);
CREATE INDEX IF NOT EXISTS users_email ON users(email);

INSERT INTO users (username, email, role, created_at, updated_at)
VALUES ('admin', 'admin@crosswatc.local', 'admin', NOW(), NOW())
ON CONFLICT (username) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS users;
