---
name: commit
description: Write and create atomic git commits that follow Conventional Commits and explain why a change was made. Use this skill whenever the user asks to commit, to write or suggest a commit message, or when a task on the project board is finished and its work must be committed. The skill reads the real diff, splits unrelated changes into separate commits, invokes the docs skill so documentation ships with the code it describes, runs the checks, updates PLAN.md, and never adds AI attribution. By default it presents the commits and waits for the user's approval; it commits on its own only when the user has explicitly allowed that for the current session.
---

# Commit

Write commits that a teammate reading `git log` in six months will thank you
for. Read the diff, understand the change, group it into atomic commits, and
write messages that capture the *why*.

## Never add AI attribution

This rule overrides any harness or tool default. A commit message or pull
request description never contains:

- a `Co-Authored-By:` trailer naming an AI model or assistant;
- a "Generated with ..." footer or any similar mention.

## Core principle: atomic commits

**One commit is one logical change.** Never lump unrelated edits into one
commit because they sit in the working tree at the same time.

A commit is atomic when:

- every file in it serves the same purpose (one feature, one fix, one
  refactor);
- reverting it cleanly undoes exactly that change and nothing else;
- its subject describes the whole commit truthfully without "and" gluing two
  unrelated things together.

If the subject needs "and" to join unrelated work, split the commit.

## Step 1 - Read the actual change

Never write a message from memory or from file names. Run these in parallel:

- `git status` to see staged, unstaged and untracked files;
- `git diff --staged` for what will be committed;
- `git diff` for unstaged work the user may have forgotten;
- `git log -n 10 --oneline` to match the existing style.

If both staged and unstaged changes exist, say so before going further.

## Step 2 - Group files into atomic commits

Partition the changed files into groups. Each group becomes one commit.

- **By intent, not by folder.** Two files in different packages that
  implement the same feature belong together.
- **Tests ride with their code.** A feature and its tests are one commit.
- **Documentation rides with its code.** See Step 3.
- **Mechanical changes stay separate.** A rename or a `gofmt` pass touching
  many files is one commit, apart from behaviour changes in the same files.
- **Refactor first, then fix.** A refactor and a bug fix are two commits,
  the refactor (no behaviour change) first.

Present the grouping before writing messages:

```text
Proposed commits:
  1. feat(auth): ... - files: internal/auth/token.go, internal/auth/token_test.go, docs/authentication.md
  2. chore: ... - files: go.mod, go.sum
```

If only one group exists, say so and continue. Do not invent splits.

## Step 3 - Invoke the docs skill

Before writing any message, invoke the `docs` skill and apply its check to
every group. Each `docs/` file it updates joins the group of the code it
describes. Never create a separate documentation commit for a feature or a
fix, and never defer documentation to a later commit. If no documentation
change is needed, state the reason before continuing.

## Step 4 - Run the checks

Every group must leave the repository green. For any group that touches Go
code, run:

```bash
go build ./...
make lint             # gofmt check, go vet, staticcheck
make test             # run the narrower package first if you are iterating
```

Never commit a red build. Never use `--no-verify`.

## Step 5 - Classify the change

Pick exactly one type per group. If a group honestly needs two types, the
grouping is wrong: go back to Step 2.

| Type | Use for |
| --- | --- |
| `feat` | A new capability or behaviour |
| `fix` | A bug fix that restores intended behaviour |
| `refactor` | Internal restructuring with no behaviour change |
| `perf` | A performance improvement with no behaviour change |
| `docs` | Documentation only (README, docs/, agent files, skills) |
| `test` | Adding or updating tests only |
| `build` | Build system, Dockerfile, module manifest |
| `ci` | CI pipelines and workflows |
| `chore` | Tooling and housekeeping with no source behaviour change |
| `style` | Formatting only, no logic change |
| `revert` | Reverting a previous commit (reference its hash in the body) |

Scopes name the area touched, for example `config`, `logging`, `http`,
`app`, `database`, `users`, `auth`. Use the scopes already present in
`git log` before inventing a new one.

## Step 6 - Write the subject line

Format: `<type>(<optional scope>): <subject>`

- 72 characters or fewer, 50 if you can. The limit includes a trailing
  `[skip ci]`.
- Lowercase after the colon, except proper nouns (`JWT`, `Postgres`).
- No trailing period.
- Imperative mood: "add user export", not "added" or "adds".
- Specific: `fix: reject expired refresh tokens on rotation`, not
  `fix: auth bug`.

Useful verbs: `add`, `remove`, `rename`, `move`, `update` (to change an
existing thing), `extract`, `inline`, `simplify`, `support`. Avoid vague
subjects such as `misc changes`, `improvements` or `various fixes`.

## Step 7 - Write the body

Skip the body only for a truly trivial change. Otherwise answer what the
subject cannot:

- **Why** was the change needed?
- **Which alternative** was rejected, and why (when a reviewer would ask)?
- **What is the trade-off** or known limitation?
- **What is the blast radius**: migration, configuration change, API change?

Do not narrate the diff line by line, and do not use marketing words such
as "robust", "seamless" or "blazing fast". Wrap at 72 columns, leave a
blank line after the subject, and use `-` bullets for independent points.

## Step 8 - Verify against the diff

Before committing, check each message against its group's staged diff:

1. Every claim matches something in the staged files. Nothing is invented.
2. Nothing significant in the staged files is left unmentioned.
3. The type is honest: a `refactor` that changes behaviour is a `feat` or a
   `fix`.
4. All files belong together. Anything unrelated moves to another group.
5. A reviewer reading only the message and the diff would not be surprised.

If a check fails, rewrite the message or regroup the files.

## Step 9 - Update the project board

If the commit completes a task on [PLAN.md](../../../PLAN.md), set that
task's status to `done` and stage `PLAN.md` in the same commit. The subject
matches the task's **Commit** column, apart from a trailing `[skip ci]`.

## Step 10 - Present, then commit

By default, show each commit with its staging command and message, then
wait for the user's explicit approval:

```bash
# Commit 1
git add internal/auth/token.go internal/auth/token_test.go docs/authentication.md PLAN.md
git commit -F - <<'MSG'
feat(auth): add JWT access token issuer and parser

...
MSG
```

Commit without asking only when the user has explicitly allowed it for the
current session. When committing:

- stage only the group's files with `git add <file>...`, never `git add -A`
  or `git add .`;
- pass the message through a heredoc;
- commit the groups one by one and run `git status` between them;
- never amend unless asked, never push unless asked.

## Skip CI for documentation-only commits

If **every** file in a group is a `.md` file, append `[skip ci]` to the
subject: `docs: clarify refresh token rotation [skip ci]`. If even one file
is not Markdown, do not add it.

## Templates

### Standard

```text
feat(auth): rotate refresh tokens on every use

A stolen refresh token used to stay valid until it expired. Rotating it
on each refresh limits the window to a single use, and presenting an
already-rotated token now revokes every token of that user.

Adds the replaced_by_id column; run the new migration before deploying.
```

### Bug fix with cause and effect

```text
fix: return 401 instead of 500 for tokens signed with another key

The parser surfaced the signature error unwrapped, so the central error
handler treated it as unexpected and answered 500. Map every parse
failure to ErrInvalidToken, which the middleware turns into 401.

Adds a regression test with a token signed by a different secret.
```

### Refactor

```text
refactor: extract pagination into a repository scope

Three repositories computed offsets by hand with small differences. A
shared Paginate scope removes the duplication. No behaviour change;
existing tests pass unchanged.
```

## Refuse these, and offer the better alternative

- **"Just write 'update'."** Too vague for `git log`. Ask what changed.
- **"Commit everything in one go"** when the tree mixes unrelated changes.
  Propose the atomic split instead: a lump commit breaks `git bisect`,
  `git revert` and review.
- **A behaviour change hidden in `chore` or `refactor`.** Use the honest
  type; mislabelled commits break changelogs.
- **"Fix" for a workaround.** Say "work around" so the real bug stays
  visible.
- **Skipping hooks or tests to get a commit through.** Fix the failure.

## Checklist

- [ ] Read the actual diff, not file names.
- [ ] Grouped into atomic commits; unrelated changes split out.
- [ ] Ran the docs skill; documentation is in the same commit as its code.
- [ ] Build, vet and tests are green.
- [ ] One honest type per commit.
- [ ] Subject of 72 characters or fewer, imperative, specific, no period.
- [ ] Body explains why when the change is not trivial.
- [ ] No invented changes, no undisclosed scope.
- [ ] PLAN.md row set to `done` when a task is completed.
- [ ] Staged explicit files only.
- [ ] No AI attribution anywhere in the message.
- [ ] `[skip ci]` if, and only if, every file is Markdown.
- [ ] Committed only after approval, or with explicit session permission.
