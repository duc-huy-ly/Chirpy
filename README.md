# Chirpy

Chirpy is a small social microblogging service written in Go. Users can create
accounts, authenticate with a password, publish short messages called chirps,
list chirps, and delete their own chirps. The service also supports rotating
access tokens with refresh tokens and a webhook that upgrades users to Chirpy
Red members.

The repository includes a small static frontend at `/app/`, but the primary
interface is the JSON API.

## Features

- User registration and profile updates
- Argon2id password hashing
- JWT access tokens with one-hour expiry
- Persistent refresh tokens that can be revoked
- Chirps limited to 139 characters
- Automatic filtering of selected words in chirp bodies
- Chirp filtering by author and sorting by creation time
- Owner-only chirp deletion
- Chirpy Red upgrades through a protected webhook
- Development-only database and metrics reset endpoint

## Technologies

- Go 1.27
- Go's standard `net/http` package
- PostgreSQL
- `sqlc`-generated database access code
- Goose SQL migrations
- `github.com/lib/pq` PostgreSQL driver
- JWTs using `github.com/golang-jwt/jwt/v5`
- Argon2id password hashing using `github.com/alexedwards/argon2id`
- UUIDs using `github.com/google/uuid`
- `.env` loading using `github.com/joho/godotenv`

## Prerequisites

- Go 1.27 or newer
- PostgreSQL
- The `goose` command-line tool for migrations
- Optional: `sqlc` if you change the SQL schema or queries

## Setup

1. Create the PostgreSQL database expected by the project:

	 ```bash
	 createdb chirpy
	 ```

	 The included scripts assume the local PostgreSQL user is `postgres` with
	 password `postgres`.

2. Apply the migrations:

	 ```bash
	 ./sql/schema/up.sh
	 ```

	 The migration scripts use:
	 `postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable`.

3. Create a `.env` file in the repository root:

	 ```dotenv
	 DB_URL=postgres://postgres:postgres@localhost:5432/chirpy?sslmode=disable
	 PLATFORM=dev
	 SECRET=replace-with-a-long-random-secret
	 POLKA_KEY=replace-with-your-polka-api-key
	 ```

	 `DB_URL`, `SECRET`, and `POLKA_KEY` are read by the application at startup.
	 `PLATFORM=dev` enables `POST /admin/reset`; use another value outside local
	 development to disable that endpoint.

4. Download Go dependencies and start the server:

	 ```bash
	 go mod download
	 go run .
	 ```

	 The server listens on `http://localhost:8080`.

## Usage

### Health check and static frontend

```bash
curl http://localhost:8080/api/healthz
curl http://localhost:8080/app/
```

### Create a user and log in

```bash
curl -X POST http://localhost:8080/api/users \
	-H 'Content-Type: application/json' \
	-d '{"email":"alice@example.com","password":"secret"}'

curl -X POST http://localhost:8080/api/login \
	-H 'Content-Type: application/json' \
	-d '{"email":"alice@example.com","password":"secret"}'
```

The login response contains a one-hour JWT in `token` and a persistent refresh
token in `refresh_token`. Store both values for authenticated requests.

### Create and read chirps

```bash
export ACCESS_TOKEN='paste-the-login-token-here'

curl -X POST http://localhost:8080/api/chirps \
	-H "Authorization: Bearer $ACCESS_TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"body":"Hello from Chirpy!"}'

curl http://localhost:8080/api/chirps
curl 'http://localhost:8080/api/chirps?sort=desc'
curl 'http://localhost:8080/api/chirps?author_id=USER_UUID'
curl http://localhost:8080/api/chirps/CHIRP_UUID
```

Chirp bodies must be shorter than 140 characters. The words `kerfuffle`,
`sharbert`, and `fornax` are replaced with `****`.

### Refresh and revoke access

```bash
export REFRESH_TOKEN='paste-the-refresh-token-here'

curl -X POST http://localhost:8080/api/refresh \
	-H "Authorization: Bearer $REFRESH_TOKEN"

curl -X POST http://localhost:8080/api/revoke \
	-H "Authorization: Bearer $REFRESH_TOKEN"
```

### Update a user or delete a chirp

```bash
curl -X PUT http://localhost:8080/api/users \
	-H "Authorization: Bearer $ACCESS_TOKEN" \
	-H 'Content-Type: application/json' \
	-d '{"email":"new@example.com","password":"new-secret"}'

curl -X DELETE http://localhost:8080/api/chirps/CHIRP_UUID \
	-H "Authorization: Bearer $ACCESS_TOKEN"
```

Only the user who created a chirp can delete it.

### Chirpy Red webhook

The webhook expects an `ApiKey` authorization header. A `user.upgraded` event
sets the specified user's `is_chirpy_red` flag to `true`:

```bash
curl -X POST http://localhost:8080/api/polka/webhooks \
	-H "Authorization: ApiKey $POLKA_KEY" \
	-H 'Content-Type: application/json' \
	-d '{"event":"user.upgraded","data":{"user_id":"USER_UUID"}}'
```

### Development administration

```bash
curl http://localhost:8080/admin/metrics
curl -X POST http://localhost:8080/admin/reset
```

The reset endpoint is available only when `PLATFORM=dev` and deletes all
users, their chirps, and their refresh tokens.

## API overview

| Method | Endpoint | Authentication | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/healthz` | None | Health check |
| `POST` | `/api/users` | None | Create a user |
| `POST` | `/api/login` | None | Log in and issue tokens |
| `GET` | `/api/chirps` | None | List chirps |
| `GET` | `/api/chirps/{chirpID}` | None | Get one chirp |
| `POST` | `/api/chirps` | Bearer JWT | Create a chirp |
| `DELETE` | `/api/chirps/{chirpID}` | Bearer JWT | Delete an owned chirp |
| `PUT` | `/api/users` | Bearer JWT | Update the current user |
| `POST` | `/api/refresh` | Bearer refresh token | Issue a new JWT |
| `POST` | `/api/revoke` | Bearer refresh token | Revoke a refresh token |
| `POST` | `/api/polka/webhooks` | ApiKey | Process a Chirpy Red upgrade |
| `GET` | `/admin/metrics` | None | View static-file hit count |
| `POST` | `/admin/reset` | None, dev only | Clear users and reset metrics |

Errors are returned as JSON objects with an `error` field.

## Development

Run the test suite with:

```bash
go test ./...
```

If SQL schemas or queries change, regenerate the database package with:

```bash
sqlc generate
```

To roll back the latest migration locally:

```bash
./sql/schema/down.sh
```
