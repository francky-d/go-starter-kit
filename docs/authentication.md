# Authentication

This page describes how the kit protects user credentials and how it
issues and checks access tokens.

## Passwords

Passwords are hashed with bcrypt before they are stored; the plain
password is never stored or logged.

- **Work factor:** `BCRYPT_COST` (default `12`). Each step doubles the
  time to hash and to verify, which slows down an attacker who obtains the
  hashes. See [configuration.md](configuration.md#authentication).
- **Salt:** every hash has its own random salt, so two users with the same
  password get different hashes.
- **Length:** bcrypt only reads the first 72 bytes of a password. Longer
  passwords are rejected rather than silently truncated: registration
  refuses them, and login treats them as a wrong password. Characters
  outside ASCII take several bytes, so 72 bytes can be fewer than 72
  characters.
- **Unknown accounts:** when a login names an email that does not exist,
  the hasher still runs a full bcrypt comparison against a dummy hash, so
  the response time does not reveal which emails have an account.

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
