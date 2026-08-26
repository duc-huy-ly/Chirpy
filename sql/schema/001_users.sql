-- Connection string : protocol://username:password@host:port/database
-- postgres://postgres:postgres@localhost:5432/chirpy

-- +goose Up
CREATE TABLE users(
  id UUID PRIMARY KEY,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  email TEXT UNIQUE NOT NULL

);

-- +goose Down
DROP TABLE users ;
