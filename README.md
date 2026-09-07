# Task Management API

REST-API for multi-user task management using go standard library, JWT Auth, idempoten task creation, transactional assignment, and structured logging.

## Run

```bash
docker-composer build
```

Start database, migration, API at `localhost:8000`

```
curl localhost:8000/health
{"status":"ok"}
```

## Endpoints

| Method | Path                 | Auth    |
| ------ | -------------------- | ------- |
| POST   | `/auth/register`     | -       |
| POST   | `/auth/login`        | -       |
| POST   | `/tasks`             | &check; |
| GET    | `/tasks`             | &check; |
| GET    | `/tasks/{id}`        | &check; |
| PUT    | `/tasks/{id}`        | &check; |
| DELETE | `/tasks/{id}`        | &check; |
| POST   | `/tasks/{id}/assign` | &check; |
| POST   | `/health`            | &check; |
| POST   | `/ready`             | &check; |

### Example

```
curl -s -X POST localhost:8000/auth/register -H 'content-type: application/json' -d '{"email":"febrian@Email.com","name":"febrian","password":"password123","team_name":"team"}'

TOKEN=$(curl -s -X POST localhost:8000/auth/login -H 'content-type: application/json' -d '{"email":"febrian@Email.com","password":"password123"}' | jq -r .data.token)

curl -s -X POST localhost:8000/tasks -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $(uuidgen)" -H 'content-type: application/json' -d '{"title": "Testing Task"}'
```

## Architecture

**Handler -> Service -> Repository**

```
cmd/
    api/            Start up & wiring
internal/
    apperr/         Error type
    config/         .env config
    domain/         Entity rules
    platform/       Platform tools
        hash/
        jwt/
        logger/
        notifier/
    repository/     SQL
    service/        Business logic
    transport/      handlers, middleware, dtos
        http/
migrations/         SQL Migration
```

**Interface declared by consumers.**
`service/task` declares `repository` function it needs, `repository/postgres` ensures the function is there and never import service.

**Idempotency enforced by database.** `POST /claims` claims its key and create task in 1 transaction.

```
INSERT INTO idempotency_keys ... ON CONFLICT DO NOTHING RETURNING id
    got id -> create task, save response
    got nothing -> different result -> 422 | completed -> replay | running -> 409
```

`UNIQUE (user_id, endpoint, idem_key)` ensures no racing condition, only 1 completed process.

**Middleware**
`RequestID -> Logging -> Recovery -> [Auth] -> handler`.
One JSON log per request; Unknown error goes to `INTERNAL_ERROR` with cause logged, never exposed to consumer. Panic become normal 500

**Stack**
`net/http` (Go 1.22 routing), `log/slog`, PostgreSQL 16 + `pgx/v5`, `golang-jwt/v5` (HS256 algorithm pin), `bcrypt`, `google/uuid`

## Test

```bash
make test-race
```

Run 50 goroutines using one idempotency key simultaneously, no database, no network. Everything in-memory fakes.
