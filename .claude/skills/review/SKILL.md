---
name: review
description: Produce or update a prioritised security and quality code review of this Go API in review/code-review.md. Use this skill when the user asks for a code review, a security audit, a quality audit, a vulnerability check, or an update of the existing review. Findings are anchored on real file and line references, ranked by severity, given stable IDs, mapped to the OWASP API Security Top 10 when they are security issues, and followed by a prioritised roadmap.
---

# Review

Produce a complete, actionable review in **one canonical file**,
`review/code-review.md`, updated in place on every pass. There are no dated
copies: the history lives in git. The file always describes the **current**
state: open findings stay in their sections with their original IDs, new
findings are appended, and resolved findings move to an appendix.

## Golden rules

- **Never invent.** Every finding cites a real `file:line` and a verified
  mechanism. If you are unsure, say so instead of asserting.
- **Facts before opinions.** Run the tools first; their output feeds the
  findings.
- **Severity reflects exploitability and impact**, not effort to fix.
- **Each finding says why it is wrong and how to fix it**, with a code
  excerpt when it helps.
- **English, plain words.**

## Severity scale

| Level | Criteria |
| --- | --- |
| Critical | Authentication bypass, privilege escalation, data of other users readable or writable, secret exposure, remote code execution. Blocks production. |
| High | Exploitable under conditions, missing authorisation on a non-trivial path, unbounded resource consumption, data loss risk. |
| Medium | Architectural debt, inconsistency, moderate risk, missing test on a critical path. |
| Low | Hygiene, readability, nice to have. |

## Process

### Step 1 - Scope

Read `review/code-review.md` if it exists, to learn the open findings and
the highest ID per prefix.

- **Full review**: no previous review, or a major rewrite. Walk the whole
  code base, package by package.
- **Differential review** (default when a review exists): only what changed
  since the date in the review title.

```bash
git log --since=<review-date> --oneline
git log --since=<review-date> --name-only --pretty=format: -- cmd internal database \
  | grep -v '^$' | sort | uniq -c | sort -rn
```

In a differential review, first re-check every open finding (still present,
or fixed by a later commit?), then review the most changed areas.

### Step 2 - Anchor on facts

```bash
go build ./...
go vet ./...
staticcheck ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...   # known CVEs in dependencies
make test                                               # state of the suite
```

Never read real environment files. Read `.env.example` and
`internal/config` to understand the configuration.

### Step 3 - Inspect every dimension

**Security** (map each finding to the OWASP API Security Top 10, 2023):

- **API1 Broken Object Level Authorization:** every query by id is scoped to
  the caller, or checked against the owner before use.
- **API2 Broken Authentication:** JWT algorithm pinned, expiry required,
  issuer checked, secret length enforced; refresh tokens stored hashed,
  rotated, revocable; login answers the same for unknown email and wrong
  password; brute force limited.
- **API3 Broken Object Property Level Authorization:** request structs bind
  only allowed fields (no binding straight into models); resources expose
  only public fields; password hashes never serialised.
- **API4 Unrestricted Resource Consumption:** body size limit, request
  timeout, pagination with a maximum page size, rate limits on expensive or
  sensitive routes.
- **API5 Broken Function Level Authorization:** protected routes sit behind
  the authentication middleware; admin-only actions check the role.
- **API6 Unrestricted Access to Sensitive Business Flows:** registration,
  login and refresh can be automated at scale; check rate limits and any
  flow an attacker would script (sign-up spam, credential stuffing).
- **API7 Server Side Request Forgery:** any feature that fetches a URL
  supplied by a client (webhooks, avatars, imports) restricts schemes,
  hosts and private address ranges.
- **API8 Security Misconfiguration:** CORS origins, security headers, debug
  output, error messages that leak causes, default secrets.
- **API9 Improper Inventory Management:** undocumented or forgotten routes,
  versioning.
- **API10 Unsafe Consumption of APIs:** timeouts and validation on calls to
  third-party services.
- **Injection:** raw SQL or `fmt.Sprintf` inside `Where`, `Order` or `Raw`
  with user input.

**Quality and Go practice:**

- Layer violations (queries in controllers, HTTP types in services).
- Ignored errors, errors compared by string, panics on request paths.
- `context.Context` not propagated to database calls.
- Goroutines without cancellation, unbounded channels.
- N+1 queries (missing `Preload` or join), transactions around external
  calls.
- Duplicate logic that a shared helper already covers.

**Database:** missing indexes on filtered or joined columns, foreign keys,
nullable columns that should not be, irreversible migrations.

**Tests:** critical paths without tests (authentication, token rotation,
authorisation, validation), missing edge cases, tests that mock the
database where a real one is required.

### Step 4 - Write

#### Document skeleton

```markdown
# Code review - <YYYY-MM-DD>

## Summary
<Production verdict (ready / not ready), top risks in one sentence each,
strengths, IDs that block production.>

## Security
### Critical
### High
### Medium
### Low

## Quality
### High
### Medium
### Low

## Tests
### High
### Medium

## Roadmap

## Resolved
```

Leave out a severity heading that has no finding.

#### Identifiers

Each finding has a stable ID `<PREFIX>-<n>`. Never reuse or renumber one.

| Section | Prefix |
| --- | --- |
| Security | `SEC-` |
| Quality | `QUA-` |
| Tests | `TST-` |

#### Finding template

```markdown
- [ ] **SEC-1 - Short, actionable title**
  - **Problem:** the technical fact and the exact mechanism (function, field, route).
  - **Why it is wrong:** the principle violated.
  - **Impact:** the concrete consequence, in one line.
  - **Exploit scenario:** a concrete request or sequence, when relevant.
  - **Classification:** OWASP API<n>:2023 - <name>.
  - **Fix:** the precise action, with a corrected excerpt when useful.
  - **Files:** [internal/auth/token.go:42-51](../internal/auth/token.go#L42-L51)
```

`Impact`, `Exploit scenario` and `Classification` are required for security
findings only. Quality and test findings need at least `Problem`, `Why it is
wrong`, `Fix` and `Files`. The file lives in `review/`, so links to code
start with `../`.

#### Resolved findings

When a finding is fixed, remove it from its section and add one line to the
appendix:

```markdown
## Resolved

- **SEC-3 - Refresh tokens stored in clear** - fixed on 2026-07-01 (`a1b2c3d`)
```

#### Roadmap

Tables grouped by horizon, each row a batch of IDs and one action:

```markdown
## Roadmap

### Now - blocks production
| IDs | Action |
| --- | --- |
| SEC-1, SEC-4 | Scope project queries by owner |

### Next - hardening
### Later - debt and hygiene
```

## Output

- The updated `review/code-review.md`.
- In the reply: the verdict, the count of findings per severity, and the IDs
  that block production.
- Do not fix findings unless the user asks. The review is the deliverable.
