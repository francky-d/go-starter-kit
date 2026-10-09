# Authentication

This page describes how the kit protects user credentials, how it issues
and checks access tokens and refresh tokens, and how sessions start, renew
and end.

## Passwords

Passwords are hashed with bcrypt before they are stored; the plain
password is never stored or logged.

- **Work factor:** `BCRYPT_COST` (default `12`). Each step doubles the
  time to hash and to verify, which slows down an attacker who obtains the
  hashes. See [configuration.md](configuration.md#authentication). After a
  change, each existing hash is replaced at the next successful login of
  its user.
- **Salt:** every hash has its own random salt, so two users with the same
  password get different hashes.
- **Length:** bcrypt only reads the first 72 bytes of a password. Longer
  passwords are rejected rather than silently truncated: registration
  refuses them, and login treats them as a wrong password. Characters
  outside ASCII take several bytes, so 72 bytes can be fewer than 72
  characters.
- **Unknown accounts:** when a login names an email that does not exist,
  the hasher still runs a full bcrypt comparison against a dummy hash, so
  the response time does not reveal which emails have an account. Until
  every hash has been upgraded after a change of `BCRYPT_COST`, old
  hashes verify at their old cost, which a precise timing could tell
  apart.

## Access tokens

An access token is a JSON Web Token signed with HMAC-SHA256 (`HS256`) and
the `JWT_SECRET` key. It carries:

| Claim | Content |
| --- | --- |
| `sub` | The id of the user. |
| `iss` | `JWT_ISSUER`. |
| `iat` | When the token was issued. |
| `exp` | When the token expires: `ACCESS_TOKEN_LIFETIME` after `iat`. |
| `jti` | A unique id for the token. |

A token is accepted only when all of these hold:

- it is signed with `HS256` and the configured secret. Tokens signed with
  another algorithm, including `none`, are refused before their signature
  is checked;
- its issuer is `JWT_ISSUER`;
- it has an expiry, and the expiry has not passed;
- it was not issued in the future;
- its subject is a user id.

Clocks may differ by up to 30 seconds between the server that issued a
token and the one that checks it.

Every refused token gives the same result, whatever the reason, so a
client learns nothing about why its token was refused.

An access token cannot be revoked: it stays valid until it expires. Keep
`ACCESS_TOKEN_LIFETIME` short. Changing `JWT_SECRET` invalidates every
access token at once.

`HS256` uses the same secret to sign and to verify, which suits an API
that both issues and checks its tokens. As soon as another service must
verify them, sharing the secret would let that service forge tokens too:
switch to an asymmetric algorithm such as `EdDSA` or `RS256`, keep the
private key in this API and publish only the public key.

## Refresh tokens

A refresh token lets a client obtain a new access token without sending
the password again. It is an opaque string, not a JWT: 32 random bytes
encoded as 43 URL-safe base64 characters.

- **Storage:** only the SHA-256 hash of the token is stored, in the
  `refresh_tokens` table. A copy of the table gives no usable token. A
  plain hash is enough here, unlike passwords: 256 random bits cannot be
  guessed from their hash.
- **Lifetime:** `REFRESH_TOKEN_LIFETIME` (30 days by default).
- **Revocation:** a refresh token can be revoked at any time. Revoked
  tokens are kept, so that a stolen token can be recognised when it is
  presented again (see [Refreshing a session](#refreshing-a-session)).
- **Secret rotation:** refresh tokens do not depend on `JWT_SECRET`, so
  changing the secret does not revoke them.

## Sessions

A session is a pair of tokens: a short-lived access token and the refresh
token that renews it. Every user can have several sessions at once, one
per device.

### Registering

Registration creates the user and starts a session. The email is stored
trimmed and in lowercase, so `Ada@Example.com` and `ada@example.com` are
the same account.

An email that already has an account is refused with
`Unable to register with the provided details.`. This message does reveal
that the email is taken: hiding it would need an email verification flow,
which the kit does not have.

### Logging in

Login starts a new session. An unknown email, a wrong password and a
password longer than 72 bytes all get the same `Invalid credentials.`
error, after the same bcrypt work.

### Refreshing a session

Exchanging a refresh token returns a new session and revokes the old
token, which records the new token as its replacement. Each refresh token
can therefore be used once.

A refresh token presented after it was already exchanged means that two
parties hold it: the client and someone who stole it. The API cannot tell
which one is legitimate, so it revokes **every** refresh token of the
user, and every device must sign in again. The event is logged as a
warning with the user id.

The same happens when a client sends the same refresh token twice: two
browser tabs refreshing at the same time, or a refresh whose response was
lost in the network and that the client retries with the old token. Only
one exchange can succeed; the other counts as a reuse. Clients should
serialise their refresh calls, and sign in again when a refresh gets no
response, instead of retrying with the same token: the retry would also
sign out every other device.

An expired token, a token revoked by a logout and an unknown token are
refused without revoking anything else.

### Logging out

Logout revokes the refresh token it receives; the other sessions of the
user stay open. Holding the refresh token is enough: no access token is
needed, so a client can sign out after its access token has expired. An
unknown or already revoked token is accepted silently, so logging out
twice is not an error.

The access token of the session stays valid until it expires, at most
`ACCESS_TOKEN_LIFETIME`. Clients should delete both tokens when they log
out.
