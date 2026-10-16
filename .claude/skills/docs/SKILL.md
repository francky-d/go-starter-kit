---
name: docs
description: Keep the docs/ folder an accurate description of the current state of the starter kit. Use this skill when the user asks to write, update or check the documentation, and whenever a change touches an endpoint, a payload, a status code, an authentication rule, a configuration variable, a command, a package boundary, a table or column, a test helper, or the deployment setup. Always invoke it before every commit (the commit skill calls it), so the documentation update ships in the same commit as the code.
---

# Docs

`docs/` explains how to use and extend the starter kit as it exists **now**.
It is the entry point for a developer who has not read the code yet.

## Principles

- **Describe the observable state only.** Confirm every statement against
  the code, the migrations and the tests. A plan, a TODO or an unmerged
  branch is not a capability.
- **The code wins.** When the code and the documentation disagree, the code
  is the source of truth: fix the documentation in the same change.
- **Present tense, no history.** Write "the refresh endpoint rotates the
  token", not "we changed the refresh endpoint to rotate the token". The
  history lives in git.
- **Show, then explain.** Prefer a short `curl` call, a code excerpt or a
  table over a paragraph.
- **French, plain words.** `docs/` and `README.md` are written in French
  (see the language rule in `AGENTS.md`). Short sentences, one idea each.
- **What stays in English.** Identifiers, commands, file paths, API
  messages, and code copied from the repository stay verbatim, comments
  included. Comments and labels in snippets written only for the
  documentation (a `# start PostgreSQL` after a shell command, the boxes
  of a text diagram) are in French, and so are the comments of the code
  samples in the guides (`docs/guides/`). Guide code that moves into the
  kit gets English comments again.
- **Same vocabulary everywhere.** Access token is « jeton d'accès »,
  refresh token is « jeton de rafraîchissement », rate limit is
  « limitation de débit »; endpoint, middleware and hash stay in English.
  « repository » names the data-access layer only; the git repository is
  « dépôt ».

## Start here

1. Read `docs/README.md` to see which documents exist.
2. Read the diff of the change (`git diff --staged` and `git diff`).
3. Answer the five questions below.
4. Update every destination the answers point to.

## Five questions to ask of every change

1. Does it change what an API client sees: a route, a payload, a status
   code, an error message, a header?
2. Does it change a guarantee or an access condition: authentication, token
   lifetime, rate limit, validation rule?
3. Does it change a dependency, a package boundary, the request lifecycle
   or a coding convention?
4. Does it change a table, a column, an index, a relation or a model?
5. Does it change a configuration variable, a command, a prerequisite or an
   operational process?

## Routing table

| Nature of the change | Destination |
| --- | --- |
| Route, payload, status code, response envelope | `docs/api-reference.md` |
| Authentication flow, token lifetime, security guarantee | `docs/authentication.md` |
| Configuration variable | `docs/configuration.md` and `.env.example` |
| Package boundary, dependency, request lifecycle | `docs/architecture.md` |
| Table, column, index, migration, model | `docs/database.md` |
| Command, prerequisite, setup step | `docs/getting-started.md` and the Commands section of `AGENTS.md` |
| Test helper, way to run tests, CI workflow | `docs/testing.md` |
| Coding convention | `docs/conventions.md` and the `conventions` skill |
| Container image, Compose services, deployment | `docs/deployment.md` |
| User-visible feature of the kit | the feature list in `README.md` |

If a destination file does not exist yet, create it and add it to the
table in `docs/README.md`. Remove or rename an entry there when a document
is removed or renamed.

## Fixes

Document the corrected behaviour as a rule of the present, not as a log of
the fix. Add or adapt a regression test that would have caught the bug. If
no test is added, say so explicitly in the output: the intent is then
unprotected.

## Expected output

- List every documentation file touched, with one line on what changed in
  each.
- If nothing changes, explain precisely why the observable contract is
  identical (for example: "internal refactor, same routes, payloads and
  configuration").
- Check that every relative link in the touched files resolves.
- The documentation update belongs to the same atomic commit as the code it
  describes. Never defer it to a later commit.
