---
name: scaffold
description: Step-by-step procedure to add a new API resource to the starter kit, from the SQL migration to the routes, tests and documentation. Use this skill when the user asks to add a resource, an entity, a model, a CRUD, an endpoint group or a new feature that stores data (for example "add projects", "create a posts API", "add a model for invoices"). It follows the conventions skill and the existing user and authentication files as templates.
---

# Scaffold a resource

This procedure adds a resource end to end. The running example is a
`Project` owned by a user, exposed under `/api/v1/projects`. Replace the
names with the real ones.

Before writing anything:

1. Activate the `conventions` skill. Every step below assumes its rules.
2. Read the existing files of the same layer and follow them. The user and
   authentication module is the reference implementation: once it exists,
   its files win over the sketches below if the two ever disagree.
3. Agree with the user on the fields, their validation rules, who owns a
   record, and which endpoints are needed. Do not guess business rules.
4. Add the work to [PLAN.md](../../../PLAN.md) as one or more tasks, one
   commit each.

## Step 1 - Migration

```bash
make migrate-create name=create_projects_table
```

```sql
-- database/migrations/NNNNNN_create_projects_table.up.sql
CREATE TABLE projects (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name        text NOT NULL,
    description text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_projects_user_id ON projects (user_id);
```

```sql
-- database/migrations/NNNNNN_create_projects_table.down.sql
DROP TABLE IF EXISTS projects;
```

Run `make migrate-up`, then `make migrate-down` and `make migrate-up` again
to prove the migration is reversible.

## Step 2 - Model

`internal/models/project.go`: embed `Base` (UUID v7 id and timestamps).

```go
// Project is a named workspace owned by a user.
type Project struct {
    Base
    UserID      uuid.UUID `gorm:"type:uuid;not null"`
    Name        string    `gorm:"not null"`
    Description string    `gorm:"not null"`
}
```

## Step 3 - Repository

`internal/repositories/project_repository.go`, with
`project_repository_test.go` next to it.

- `NewProjectRepository(db *gorm.DB) *ProjectRepository`
- `WithTx(tx *gorm.DB) *ProjectRepository`
- `Create(ctx, *models.Project) error`
- `FindByIDForUser(ctx, id, userID uuid.UUID) (*models.Project, error)`:
  scoping by owner in the query makes "not yours" and "does not exist"
  indistinguishable.
- `ListForUser(ctx, userID uuid.UUID, page, perPage int) ([]models.Project, int64, error)`:
  go through the shared pagination scope, which defaults `per_page` and
  caps it (100) so a client cannot ask for the whole table.
- `Save(ctx, *models.Project) error` persists every column of a loaded
  record. Do not use `Updates(struct)` for partial updates: GORM skips zero
  values, so clearing a field to `""` or `0` would silently do nothing.
- `Delete(ctx, *models.Project) error`

Query through `r.db.WithContext(ctx)`. Translate `gorm.ErrRecordNotFound` to
`ErrNotFound` and duplicates to `ErrDuplicate`. Reuse the shared scopes
(pagination, ordering) instead of computing offsets by hand.

## Step 4 - Service

`internal/services/project_service.go`, with tests.

- Input structs (`CreateProjectInput`) carry plain values, never
  `echo.Context`.
- Map `repositories.ErrNotFound` to `apperror.NotFound()` and wrap any
  other error with `apperror.Internal(err)`.
- Business rules live here: quotas, state transitions, ownership.
- **Partial update:** load the record with `FindByIDForUser`, copy each
  non-nil field of the input onto it, then call `Save`.

## Step 5 - Requests

`internal/http/requests/project.go`:

```go
// CreateProject is the body of POST /api/v1/projects.
type CreateProject struct {
    Name        string `json:"name" validate:"required,max=120"`
    Description string `json:"description" validate:"max=2000"`
}
```

Every string has a `max`. Optional fields of an update use pointers so
"absent" and "empty" stay distinguishable. Query parameters (`page`,
`per_page`, filters) get a request struct and are validated like a body.

Path ids are parsed with `uuid.Parse(c.Param("id"))` through a shared
helper in `internal/http/requests` (create it the first time it is
needed). An invalid id answers 404, exactly like an unknown one.

## Step 6 - Resource

`internal/http/resources/project.go`: a struct with `json` tags and a
constructor per shape (`NewProject`, `NewProjects`). Only the fields a client
may see.

## Step 7 - Controller

`internal/http/controllers/project_controller.go`: one method per action
(`List`, `Show`, `Create`, `Update`, `Delete`). Each method binds, maps the request
to the service input (the service never sees HTTP types), calls one
service method, and responds:

```go
// Create stores a new project for the authenticated user.
func (pc *ProjectController) Create(c echo.Context) error {
    var input requests.CreateProject
    if err := requests.Bind(c, &input); err != nil {
        return err
    }

    project, err := pc.projects.Create(c.Request().Context(), services.CreateProjectInput{
        OwnerID:     middleware.CurrentUserID(c),
        Name:        input.Name,
        Description: input.Description,
    })
    if err != nil {
        return err
    }

    return respond.Created(c, resources.NewProject(project))
}
```

## Step 8 - Wiring and routes

1. Build the repository, service and controller in the application wiring
   (`internal/app`) with their constructors.
2. Add the controller to the route dependencies (`routes.Dependencies`).
3. Register the routes behind the authentication middleware, in
   `internal/routes/routes.go`:

```go
api.Group("/projects").Middleware(d.Authenticated).
    Get("", d.Projects.List).
    Post("", d.Projects.Create).
    Get("/:id", d.Projects.Show).
    Patch("/:id", d.Projects.Update).
    Delete("/:id", d.Projects.Delete)
```

## Step 9 - HTTP tests

`internal/http/controllers/project_controller_test.go`, through the real
application with the `internal/testutil` helpers. Cover at least:

- the happy path of each endpoint (status and body);
- 401 without a token;
- 404 on another user's record (not 403);
- 422 with field errors for invalid input;
- pagination metadata on the list.

## Step 10 - Documentation and commit

1. Run the `docs` skill. Expect at least `docs/api-reference.md` and
   `docs/database.md`, and the feature list of `README.md` if the resource
   is part of the kit itself.
2. Run the whole suite and the linters.
3. Use the `commit` skill: one commit per board task, docs included.

## Checklist

- [ ] Fields, ownership and endpoints agreed with the user.
- [ ] Migration with a reversible `down`, indexes on foreign keys.
- [ ] Model embeds `Base`.
- [ ] Repository scopes by owner, caps pagination and returns sentinel errors.
- [ ] Partial updates load, apply non-nil fields, then `Save`.
- [ ] Service maps errors to `apperror`, no HTTP types.
- [ ] Request structs validate every field and cap every string.
- [ ] Resource exposes only public fields.
- [ ] Controller is thin and uses `respond`.
- [ ] Routes are behind authentication where needed.
- [ ] Tests for success, 401, 404, 422 and pagination.
- [ ] Documentation updated in the same commit.
