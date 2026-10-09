# API reference

This page describes the HTTP contract of the API: the response envelopes,
the error format, and every endpoint.

## Response envelope

Every successful JSON response wraps its payload in `data`:

```json
{
  "data": { "id": "0192f1c4-7b7e-7a3e-9a51-3f1f2f1e8c11", "name": "Ada" }
}
```

Collections that are paginated add a `meta` object:

```json
{
  "data": [{ "id": "..." }, { "id": "..." }],
  "meta": { "page": 2, "per_page": 20, "total": 135, "last_page": 7 }
}
```

`204 No Content` responses have no body.

## Headers

Every response carries:

| Header | Value |
| --- | --- |
| `X-Request-Id` | Identifier of the request, also written in every log record about it. A client-provided value is kept when it has 1 to 64 letters, digits, `.`, `_` or `-`; otherwise a new one is generated. |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'` |
| `Referrer-Policy` | `no-referrer` |

Send request bodies as JSON with `Content-Type: application/json`. Bodies
larger than `HTTP_BODY_LIMIT` are rejected.

## Errors

Every error response has the same shape. `errors` is present only for
validation failures and maps each invalid field to its messages:

```json
{
  "message": "The given data was invalid.",
  "errors": {
    "email": ["must be a valid email address"]
  }
}
```

### Status codes

| Status | Message | When |
| --- | --- | --- |
| 400 | `Malformed request body.` | The body is not valid JSON, or a value has the wrong type. |
| 401 | `Unauthenticated.` | Authentication is missing or invalid. Login answers `Invalid credentials.` instead. |
| 403 | `This action is forbidden.` | The caller is authenticated but not allowed. |
| 404 | `Resource not found.` | The route or the record does not exist, or the record belongs to someone else. |
| 409 | depends on the error | The request conflicts with the current state. |
| 413 | `Request Entity Too Large.` | The body is larger than `HTTP_BODY_LIMIT`. |
| 415 | `Unsupported Media Type.` | The body is not sent as `application/json`. |
| 422 | `The given data was invalid.` | Validation failed; see `errors`. |
| 429 | `Too many requests.` | A rate limit was hit. The `Retry-After` header gives the seconds to wait. |
| 500 | `Internal server error.` | An unexpected error occurred. |
| 503 | `Service unavailable.` | A dependency is down, or the request ran longer than `HTTP_REQUEST_TIMEOUT`. |

Other statuses produced by the HTTP layer (for example `405`) use the
standard reason phrase as message, such as `Method Not Allowed.`.

### Validation errors

Each key of `errors` is the JSON name of an invalid field. Nested fields
use their path, such as `address.city` or `items[1].name`. A field can have
several messages.

| Rule | Message |
| --- | --- |
| required | `is required` |
| email | `must be a valid email address` |
| uuid | `must be a valid UUID` |
| url | `must be a valid URL` |
| oneof | `must be one of: <values>` |
| min, gte | `must be at least <n>`, with `characters` for text and `items` for lists |
| max, lte | `must be at most <n>`, with `characters` for text and `items` for lists |
| len | `must be exactly <n>`, with the same units |
| any other rule | `is invalid` |

Messages never repeat the submitted value. Fields the endpoint does not
expect are ignored.

### What an error never contains

Error messages are written to be safe for any client:

- a `404` does not say whether the record exists;
- a `403` names neither the owner nor the missing permission;
- a `400` never quotes the JSON parser's error, which could repeat part of
  the body;
- a `500` never carries an internal error message, a file path or a stack
  trace. The cause is written to the server logs instead.

`HEAD` requests receive the status code without a body.

## Endpoints

| Method | Path | Authentication | Purpose |
| --- | --- | --- | --- |
| `GET` | `/health` | none | Report whether the application and its database can serve requests. |
| `POST` | `/api/v1/auth/register` | none, rate limited | Create an account and start a session. |
| `POST` | `/api/v1/auth/login` | none, rate limited | Start a session with an email and a password. |
| `POST` | `/api/v1/auth/refresh` | none, rate limited | Exchange a refresh token for a new session. |
| `POST` | `/api/v1/auth/logout` | none, rate limited | Revoke a refresh token. |

Register and login together accept `AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE`
requests per minute from each client IP; refresh and logout together
accept `AUTH_REFRESH_RATE_LIMIT_PER_MINUTE`. Beyond that they answer `429`.
See [authentication.md](authentication.md) for how sessions and tokens
work.

### `GET /health`

Answers `200` when the application can reach its database within two
seconds:

```bash
curl -s http://localhost:8080/health
```

```json
{ "data": { "status": "ok", "database": "ok" } }
```

Answers `503` with `{"message":"Service unavailable."}` otherwise; the cause
is written to the server logs.

Use it as a **readiness** probe, so traffic stops while the database is
down. Do not use it as a liveness probe: restarting the API does not fix a
database outage.

### Sessions

Register, login and refresh answer with a session:

```json
{
  "data": {
    "user": {
      "id": "0192f1c4-7b7e-7a3e-9a51-3f1f2f1e8c11",
      "name": "Ada Lovelace",
      "email": "ada@example.com",
      "created_at": "2026-10-09T12:00:00Z"
    },
    "token_type": "Bearer",
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "refresh_token": "kq3V0n0t2uJx3ZbJ0yq6Q3pW8m1sT4cV5bN7aL9dE2f",
    "refresh_token_expires_at": "2026-11-08T12:00:00Z"
  }
}
```

| Field | Content |
| --- | --- |
| `access_token` | Send it as `Authorization: Bearer <access_token>` to authenticated routes. |
| `expires_in` | Seconds until the access token expires. |
| `refresh_token` | Exchange it at `/api/v1/auth/refresh` before `refresh_token_expires_at`. Each refresh token works once. |

Store the refresh token as carefully as a password: whoever holds it can
open a session.

### `POST /api/v1/auth/register`

```bash
curl -s http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"correct horse battery staple"}'
```

| Field | Rules |
| --- | --- |
| `name` | Required, at most 100 characters. |
| `email` | Required, a valid email, at most 254 characters. Stored in lowercase. |
| `password` | Required, 8 to 72 characters, and at most 72 bytes. |

| Status | When |
| --- | --- |
| `201` | The account was created; the body is a session. |
| `409` | The email already has an account: `Unable to register with the provided details.` |
| `422` | A field is invalid. |

### `POST /api/v1/auth/login`

```bash
curl -s http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct horse battery staple"}'
```

| Status | When |
| --- | --- |
| `200` | The body is a new session. |
| `401` | `Invalid credentials.`, whether the email is unknown or the password is wrong. |
| `422` | `email` or `password` is missing, or `email` is longer than 254 characters. |

### `POST /api/v1/auth/refresh`

```bash
curl -s http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh_token>"}'
```

| Status | When |
| --- | --- |
| `200` | The body is a new session; the refresh token sent is now revoked. |
| `401` | `Unauthenticated.`: the token is unknown, expired or revoked. A token that was already exchanged also revokes every session of its user. |
| `422` | `refresh_token` is missing or longer than 128 characters. |

### `POST /api/v1/auth/logout`

```bash
curl -s -i http://localhost:8080/api/v1/auth/logout \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh_token>"}'
```

| Status | When |
| --- | --- |
| `204` | The refresh token is revoked. An unknown or already revoked token gets the same answer. |
| `422` | `refresh_token` is missing or longer than 128 characters. |

The access token stays valid until it expires; delete it on the client.
