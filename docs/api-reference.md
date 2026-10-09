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
| 401 | `Unauthenticated.` | Authentication is missing or invalid. |
| 403 | `This action is forbidden.` | The caller is authenticated but not allowed. |
| 404 | `Resource not found.` | The route or the record does not exist, or the record belongs to someone else. |
| 409 | depends on the error | The request conflicts with the current state. |
| 415 | `Unsupported Media Type.` | The body is not sent as `application/json`. |
| 422 | `The given data was invalid.` | Validation failed; see `errors`. |
| 429 | `Too many requests.` | A rate limit was hit. |
| 500 | `Internal server error.` | An unexpected error occurred. |
| 503 | `Service unavailable.` | A dependency is down. |

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
