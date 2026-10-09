@AGENTS.md

# Claude Code

The rules above apply in full. This section only adds what is specific to
Claude Code.

## Skills activation

Project skills live in `.claude/skills/`. Activate the relevant skill as soon
as you work in its domain, not only when you are stuck.

| Skill | Activate when |
| --- | --- |
| `commit` | Committing, writing a commit message, or finishing a task on the board |
| `docs` | Before every commit (the commit skill calls it), or when asked to write or check documentation |
| `conventions` | Before writing or changing any Go code, SQL migration or test |
| `scaffold` | Adding a resource, a model, a CRUD or an endpoint group that stores data |
| `review` | Asked for a code review, a security or quality audit, or an update of the review |

## Settings

`.claude/settings.json` denies the Read tool on real environment files. The
deny list is a safety net, not a licence: the secrets rule in `AGENTS.md`
applies to every tool, including the shell.
