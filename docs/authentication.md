# Authentication

This page describes how the kit protects user credentials.

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
