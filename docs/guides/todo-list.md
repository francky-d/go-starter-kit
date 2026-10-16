# Guide : une to-do list partagée, avec des rôles

Ce guide construit, sur le kit, une API de to-do lists que leur
propriétaire partage avec d'autres utilisateurs, chacun avec un rôle. Il
montre comment ajouter une ressource de bout en bout, comment protéger
chaque action par un droit, et comment le prouver par des tests.

Le code de ce guide ne fait pas partie du kit : le dépôt reste un point de
départ vierge. Il a été écrit dans une copie du kit puis vérifié :
`make lint` passe, et toute la suite passe avec le détecteur de courses
(`go test -race -p 1 ./...`). Chaque nouveau fichier est donné en entier ;
une modification d'un fichier existant du kit est donnée sous forme
d'extrait. Les commentaires du code sont en français pour ce guide ; le
code du kit, lui, est commenté en anglais.

Le guide suit la procédure du
[skill scaffold](../../.claude/skills/scaffold/SKILL.md) et les
[conventions](../conventions.md). Il suppose le kit installé et la base
locale démarrée (voir [Prise en main](../getting-started.md)).

## Sommaire

1. [Ce que l'on construit](#ce-que-lon-construit)
2. [Choix de conception](#choix-de-conception)
3. [Étape 1 : migrations](#étape-1--migrations)
4. [Étape 2 : modèles](#étape-2--modèles)
5. [Étape 3 : repositories](#étape-3--repositories)
6. [Étape 4 : services](#étape-4--services)
7. [Étape 5 : requests](#étape-5--requests)
8. [Étape 6 : resources](#étape-6--resources)
9. [Étape 7 : contrôleurs](#étape-7--contrôleurs)
10. [Étape 8 : routes et câblage](#étape-8--routes-et-câblage)
11. [Étape 9 : tests](#étape-9--tests)
12. [Étape 10 : essayer avec curl](#étape-10--essayer-avec-curl)
13. [Documenter](#documenter)
14. [Limites](#limites)
15. [Et pour un blog ?](#et-pour-un-blog-)

## Ce que l'on construit

### Les règles

- Un utilisateur crée des listes : il en est le **propriétaire**
  (`owner`).
- Le propriétaire partage une liste avec un autre utilisateur, par son
  email, avec le rôle `editor` ou `viewer`. Partager à nouveau avec un
  autre rôle change le rôle.
- Une liste n'a qu'un propriétaire : le rôle `owner` ne se donne pas.
- Chaque rôle inclut les droits des rôles en dessous :
  `owner` > `editor` > `viewer`.
- Un utilisateur sans accès reçoit un `404`, exactement comme pour une
  liste qui n'existe pas : il ne peut pas apprendre qu'elle existe. Un
  membre dont le rôle ne suffit pas reçoit un `403`.
- Un membre peut quitter une liste ; le propriétaire peut retirer
  n'importe quel membre.
- Supprimer une liste supprime ses tâches et ses membres.

### Matrice des droits

| Action | `owner` | `editor` | `viewer` | sans accès |
| --- | --- | --- | --- | --- |
| Voir la liste, ses tâches et ses membres | `200` | `200` | `200` | `404` |
| Créer une tâche | `201` | `201` | `403` | `404` |
| Modifier ou supprimer une tâche | `200` / `204` | `200` / `204` | `403` | `404` |
| Partager la liste, changer un rôle | `200` | `403` | `403` | `404` |
| Retirer un autre membre | `204` | `403` | `403` | `404` |
| Quitter la liste | `404` (pas membre) | `204` | `204` | `404` |
| Supprimer la liste | `204` | `403` | `403` | `404` |

Lister ses listes et en créer une demandent seulement d'être authentifié.

### Les endpoints

Tous demandent un jeton d'accès (voir
[authentification](../authentication.md#authentifier-les-requêtes)).

| Méthode | Chemin | Rôle minimal | Réponse |
| --- | --- | --- | --- |
| `GET` | `/api/v1/lists` | aucun | Les listes possédées ou partagées, paginées, avec le rôle de l'appelant. |
| `POST` | `/api/v1/lists` | aucun | `201` et la liste créée. |
| `GET` | `/api/v1/lists/:list` | `viewer` | La liste. |
| `DELETE` | `/api/v1/lists/:list` | `owner` | `204`. |
| `GET` | `/api/v1/lists/:list/tasks` | `viewer` | Les tâches, paginées. |
| `POST` | `/api/v1/lists/:list/tasks` | `editor` | `201` et la tâche créée. |
| `PATCH` | `/api/v1/lists/:list/tasks/:task` | `editor` | La tâche modifiée : `title` et `completed`, chacun facultatif. |
| `DELETE` | `/api/v1/lists/:list/tasks/:task` | `editor` | `204`. |
| `GET` | `/api/v1/lists/:list/members` | `viewer` | Les membres : identifiant, nom et rôle. |
| `PUT` | `/api/v1/lists/:list/members` | `owner` | Donne ou change le rôle de l'utilisateur d'un email ; débit limité. |
| `DELETE` | `/api/v1/lists/:list/members/:user` | `owner`, ou le membre lui-même | `204`. |

## Choix de conception

**Les droits se vérifient dans les services.** Chaque méthode de service
commence par `access.Require(ctx, userID, listID, rôle)`. Un middleware de
route qui chargerait la liste se lirait bien aussi dans le fichier des
routes, mais une route enregistrée sans lui ne vérifierait rien. Dans le
service, aucun chemin ne contourne le contrôle.

**Une seule requête charge la liste et le rôle.** Le repository joint
`list_members` et calcule le rôle dans le `SELECT`. Une liste que
l'utilisateur ne peut pas voir est introuvable par construction : « pas à
vous » et « n'existe pas » donnent la même erreur, comme partout dans le
kit.

**Le `404` passe avant le `403`.** Le `403` ne s'adresse qu'à un membre,
qui sait déjà que la liste existe. Un identifiant mal formé donne aussi un
`404`.

**Le partage révèle qu'un email a un compte, et le nom de ce compte.** Le
propriétaire doit savoir si son partage a abouti, et la liste des membres
montrerait le nom de toute façon : n'importe quel utilisateur connecté qui
crée une liste peut donc relier un email à un nom. Comme pour
l'inscription, la route a sa propre limitation de débit. Masquer ces
informations exigerait un parcours d'invitation par email, avec
acceptation.

**Les membres ne voient pas les emails des autres membres**, seulement
leur nom et leur rôle.

**Une tâche se met à jour avec `Select(...).Updates`, pas avec `Save`.**
Quand l'`UPDATE` de `Save` ne touche aucune ligne, GORM insère
l'enregistrement : une tâche supprimée par une autre requête entre la
lecture et l'écriture réapparaîtrait. `Select` liste les colonnes écrites,
y compris une valeur vide comme `completed_at` à `NULL`.

**Un champ facultatif d'une mise à jour est un pointeur**, pour distinguer
« absent » de « vide ». Sur un pointeur, la règle `required` vérifie
seulement que le pointeur n'est pas nul : `""` passerait. Le titre utilise
donc `omitnil,min=1`.

## Étape 1 : migrations

```bash
make migrate-create name=create_todo_lists_table
make migrate-create name=create_list_members_table
make migrate-create name=create_tasks_table
```

Le propriétaire est une colonne de `todo_lists`, pas une ligne de
`list_members` : une liste a toujours exactement un propriétaire, et la
contrainte `list_members_role` interdit d'en ajouter un second.

`database/migrations/000003_create_todo_lists_table.up.sql`

```sql
CREATE TABLE todo_lists (
    id         uuid PRIMARY KEY,
    owner_id   uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title      text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_todo_lists_owner_id ON todo_lists (owner_id);
```

`database/migrations/000004_create_list_members_table.up.sql`

```sql
-- Le propriétaire d'une liste n'en est pas membre : ce rôle est porté par
-- todo_lists.owner_id.
CREATE TABLE list_members (
    id         uuid PRIMARY KEY,
    list_id    uuid NOT NULL REFERENCES todo_lists (id) ON DELETE CASCADE,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT list_members_role CHECK (role IN ('viewer', 'editor'))
);

CREATE UNIQUE INDEX idx_list_members_list_id_user_id ON list_members (list_id, user_id);
CREATE INDEX idx_list_members_user_id ON list_members (user_id);
```

`database/migrations/000005_create_tasks_table.up.sql`

```sql
CREATE TABLE tasks (
    id           uuid PRIMARY KEY,
    list_id      uuid NOT NULL REFERENCES todo_lists (id) ON DELETE CASCADE,
    title        text NOT NULL,
    completed_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_tasks_list_id ON tasks (list_id);
```

Chaque fichier `down` supprime sa table : `DROP TABLE IF EXISTS todo_lists;`,
`DROP TABLE IF EXISTS list_members;` et `DROP TABLE IF EXISTS tasks;`.

```bash
make migrate-up && make migrate-down && make migrate-up
```

`make migrate-down` n'annule que la dernière migration ;
`TestMigrator_AppliesAndRollsBackEveryMigration`, lancé par `make test`,
applique et annule les trois.

## Étape 2 : modèles

Les rôles sont ordonnés : `AtLeast` répond à la question « ce rôle
suffit-il ? ». Un rôle inconnu, y compris vide, n'accorde rien.

`internal/models/role.go`

```go
package models

// Role est ce qu'un utilisateur peut faire sur une to-do list. Chaque rôle
// inclut les droits des rôles inférieurs : un editor peut faire tout ce que
// fait un viewer.
type Role string

const (
	RoleViewer Role = "viewer"
	RoleEditor Role = "editor"
	RoleOwner  Role = "owner"
)

var roleRanks = map[Role]int{RoleViewer: 1, RoleEditor: 2, RoleOwner: 3}

// AtLeast vaut false pour un rôle inconnu : un rôle absent n'accorde rien.
func (r Role) AtLeast(required Role) bool {
	rank, known := roleRanks[r]

	return known && rank >= roleRanks[required]
}
```

`Role` n'est pas une colonne de `todo_lists` : la requête du repository le
calcule. Le tag `gorm:"->"` le rend lisible mais jamais écrit.

`internal/models/todo_list.go`

```go
package models

import "github.com/google/uuid"

type TodoList struct {
	Base
	OwnerID uuid.UUID `gorm:"type:uuid"`
	Title   string
	// Role est le rôle de l'utilisateur pour qui la liste a été
	// chargée. La requête le calcule ; ce n'est pas une colonne de
	// todo_lists, d'où la lecture seule.
	Role Role `gorm:"->"`
}
```

`internal/models/list_member.go`

```go
package models

import "github.com/google/uuid"

// ListMember donne à un utilisateur un rôle sur une liste dont il n'est pas
// le propriétaire.
type ListMember struct {
	Base
	ListID uuid.UUID `gorm:"type:uuid"`
	UserID uuid.UUID `gorm:"type:uuid"`
	Role   Role
	User   *User
}
```

`internal/models/task.go`

```go
package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	Base
	ListID      uuid.UUID `gorm:"type:uuid"`
	Title       string
	CompletedAt *time.Time
}

func (t *Task) IsCompleted() bool {
	return t.CompletedAt != nil
}

// SetCompleted conserve la première date d'achèvement quand une tâche déjà
// terminée est de nouveau marquée comme terminée.
func (t *Task) SetCompleted(completed bool, now time.Time) {
	switch {
	case !completed:
		t.CompletedAt = nil
	case !t.IsCompleted():
		t.CompletedAt = &now
	}
}
```

## Étape 3 : repositories

Ajoutez à `internal/repositories/errors.go` un helper pour les écritures
qui doivent toucher une ligne :

```go
// affected renvoie ErrNotFound quand une écriture ne touche aucune ligne,
// par exemple un enregistrement qu'une autre requête a supprimé entre-temps.
func affected(result *gorm.DB) error {
	if result.Error == nil && result.RowsAffected == 0 {
		return ErrNotFound
	}

	return translate(result.Error)
}
```

`visibleTo` est le cœur du contrôle d'accès : il ne garde que les listes
que l'utilisateur possède ou dont il est membre. `withRoleOf` ajoute le
rôle au `SELECT`. Le comptage et la page utilisent chacun une requête
neuve : une chaîne GORM déjà exécutée ne se réutilise pas.

`internal/repositories/todo_list_repository.go`

```go
package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type TodoListRepository struct {
	db *gorm.DB
}

func NewTodoListRepository(db *gorm.DB) *TodoListRepository {
	return &TodoListRepository{db: db}
}

func (r *TodoListRepository) Create(ctx context.Context, list *models.TodoList) error {
	return translate(r.db.WithContext(ctx).Create(list).Error)
}

// FindForUser charge la liste avec le rôle que userID y détient. Une liste
// qui n'existe pas et une liste que userID ne peut pas voir donnent toutes
// deux ErrNotFound.
func (r *TodoListRepository) FindForUser(ctx context.Context, id, userID uuid.UUID) (*models.TodoList, error) {
	var list models.TodoList
	err := r.visibleTo(ctx, userID).
		Scopes(withRoleOf(userID)).
		Where("todo_lists.id = ?", id).
		Take(&list).Error
	if err != nil {
		return nil, translate(err)
	}

	return &list, nil
}

func (r *TodoListRepository) FindAllForUser(ctx context.Context, userID uuid.UUID, page, perPage int) ([]models.TodoList, int64, error) {
	var total int64
	if err := r.visibleTo(ctx, userID).Count(&total).Error; err != nil {
		return nil, 0, translate(err)
	}

	var lists []models.TodoList
	err := r.visibleTo(ctx, userID).
		Scopes(withRoleOf(userID), Paginate(page, perPage)).
		Order("todo_lists.created_at DESC, todo_lists.id DESC").
		Find(&lists).Error

	return lists, total, translate(err)
}

func (r *TodoListRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return affected(r.db.WithContext(ctx).Delete(&models.TodoList{}, "id = ?", id))
}

// visibleTo retient les listes dont userID est propriétaire ou membre.
func (r *TodoListRepository) visibleTo(ctx context.Context, userID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).
		Model(&models.TodoList{}).
		Joins("LEFT JOIN list_members ON list_members.list_id = todo_lists.id AND list_members.user_id = ?", userID).
		Where("(todo_lists.owner_id = ? OR list_members.user_id IS NOT NULL)", userID)
}

// withRoleOf sélectionne, avec chaque liste, le rôle que userID y détient.
func withRoleOf(userID uuid.UUID) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Select(
			"todo_lists.*, CASE WHEN todo_lists.owner_id = ? THEN ? ELSE list_members.role END AS role",
			userID, models.RoleOwner,
		)
	}
}
```

`AddOrUpdate` s'appuie sur l'index unique `(list_id, user_id)` : partager
deux fois avec le même utilisateur change son rôle au lieu de créer un
doublon. `Omit(clause.Associations)` empêche GORM d'écrire l'utilisateur
attaché au membre.

`internal/repositories/list_member_repository.go`

```go
package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type ListMemberRepository struct {
	db *gorm.DB
}

func NewListMemberRepository(db *gorm.DB) *ListMemberRepository {
	return &ListMemberRepository{db: db}
}

// AddOrUpdate ajoute le membre, ou change son rôle s'il est déjà membre de
// la liste. Il n'écrit jamais l'utilisateur rattaché au membre.
func (r *ListMemberRepository) AddOrUpdate(ctx context.Context, member *models.ListMember) error {
	return translate(r.db.WithContext(ctx).
		Omit(clause.Associations).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "list_id"}, {Name: "user_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"role", "updated_at"}),
		}).
		Create(member).Error)
}

func (r *ListMemberRepository) FindAllInList(ctx context.Context, listID uuid.UUID) ([]models.ListMember, error) {
	var members []models.ListMember
	err := r.db.WithContext(ctx).
		Preload("User").
		Where("list_id = ?", listID).
		Order("created_at, id").
		Find(&members).Error

	return members, translate(err)
}

func (r *ListMemberRepository) Delete(ctx context.Context, listID, userID uuid.UUID) error {
	return affected(r.db.WithContext(ctx).Delete(&models.ListMember{}, "list_id = ? AND user_id = ?", listID, userID))
}
```

`internal/repositories/task_repository.go`

```go
package repositories

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type TaskRepository struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, task *models.Task) error {
	return translate(r.db.WithContext(ctx).Create(task).Error)
}

// FindInList cherche la tâche dans sa liste : l'identifiant d'une tâche
// d'une autre liste donne ErrNotFound, même si l'utilisateur voit cette
// autre liste.
func (r *TaskRepository) FindInList(ctx context.Context, listID, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	if err := r.inList(ctx, listID).Where("id = ?", id).Take(&task).Error; err != nil {
		return nil, translate(err)
	}

	return &task, nil
}

func (r *TaskRepository) FindAllInList(ctx context.Context, listID uuid.UUID, page, perPage int) ([]models.Task, int64, error) {
	var total int64
	if err := r.inList(ctx, listID).Count(&total).Error; err != nil {
		return nil, 0, translate(err)
	}

	var tasks []models.Task
	err := r.inList(ctx, listID).
		Scopes(Paginate(page, perPage)).
		Order("created_at, id").
		Find(&tasks).Error

	return tasks, total, translate(err)
}

// Update écrit le titre et l'achèvement, y compris un achèvement effacé.
// Contrairement à Save, il ne réinsère jamais la tâche si une autre
// requête l'a supprimée entre-temps.
func (r *TaskRepository) Update(ctx context.Context, task *models.Task) error {
	return affected(r.db.WithContext(ctx).Select("title", "completed_at").Updates(task))
}

func (r *TaskRepository) Delete(ctx context.Context, task *models.Task) error {
	return affected(r.db.WithContext(ctx).Delete(task))
}

func (r *TaskRepository) inList(ctx context.Context, listID uuid.UUID) *gorm.DB {
	return r.db.WithContext(ctx).Model(&models.Task{}).Where("list_id = ?", listID)
}
```

## Étape 4 : services

`Page` porte la page et sa taille réellement appliquées, après les valeurs
par défaut et le plafond de `repositories.NormalizePage`, pour que la
réponse les annonce.

`internal/services/page.go`

```go
package services

import "github.com/francky-d/go-starter-kit/internal/repositories"

// Page est une page d'une collection, avec le numéro et la taille de page
// réellement appliqués après les valeurs par défaut et les plafonds.
type Page[T any] struct {
	Items  []T
	Number int
	Size   int
	Total  int64
}

// findPage normalise la page demandée, puis l'utilise pour exécuter find.
func findPage[T any](page, perPage int, find func(page, perPage int) ([]T, int64, error)) (Page[T], error) {
	page, perPage = repositories.NormalizePage(page, perPage)

	items, total, err := find(page, perPage)
	if err != nil {
		return Page[T]{}, err
	}

	return Page[T]{Items: items, Number: page, Size: perPage, Total: total}, nil
}
```

`ListAccess` est le seul endroit qui décide d'un droit. Les trois services
du guide le reçoivent par leur constructeur et l'appellent en premier.

`internal/services/list_access.go`

```go
package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
)

// ListAccess décide de ce qu'un utilisateur peut faire sur une to-do list.
// Chaque service de la fonctionnalité l'interroge en premier : aucune action
// n'échappe à la vérification.
type ListAccess struct {
	lists *repositories.TodoListRepository
}

func NewListAccess(lists *repositories.TodoListRepository) *ListAccess {
	return &ListAccess{lists: lists}
}

// Require renvoie la liste quand userID y détient au moins le rôle requis.
// Un utilisateur sans accès reçoit un 404 et ne peut donc pas savoir si la
// liste existe ; un membre dont le rôle est insuffisant reçoit un 403.
func (a *ListAccess) Require(ctx context.Context, userID, listID uuid.UUID, required models.Role) (*models.TodoList, error) {
	list, err := a.lists.FindForUser(ctx, listID, userID)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, apperror.NotFound()
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	if !list.Role.AtLeast(required) {
		return nil, apperror.Forbidden()
	}

	return list, nil
}

func notFoundOrInternal(err error) error {
	if errors.Is(err, repositories.ErrNotFound) {
		return apperror.NotFound()
	}

	return apperror.Internal(err)
}
```

`internal/services/todo_list_service.go`

```go
package services

import (
	"context"

	"github.com/google/uuid"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
)

type TodoListService struct {
	lists  *repositories.TodoListRepository
	access *ListAccess
}

func NewTodoListService(lists *repositories.TodoListRepository, access *ListAccess) *TodoListService {
	return &TodoListService{lists: lists, access: access}
}

type CreateTodoListInput struct {
	OwnerID uuid.UUID
	Title   string
}

func (s *TodoListService) List(ctx context.Context, userID uuid.UUID, page, perPage int) (Page[models.TodoList], error) {
	lists, err := findPage(page, perPage, func(page, perPage int) ([]models.TodoList, int64, error) {
		return s.lists.FindAllForUser(ctx, userID, page, perPage)
	})
	if err != nil {
		return Page[models.TodoList]{}, apperror.Internal(err)
	}

	return lists, nil
}

func (s *TodoListService) Create(ctx context.Context, input CreateTodoListInput) (*models.TodoList, error) {
	list := &models.TodoList{OwnerID: input.OwnerID, Title: input.Title, Role: models.RoleOwner}
	if err := s.lists.Create(ctx, list); err != nil {
		return nil, apperror.Internal(err)
	}

	return list, nil
}

func (s *TodoListService) Show(ctx context.Context, userID, listID uuid.UUID) (*models.TodoList, error) {
	return s.access.Require(ctx, userID, listID, models.RoleViewer)
}

// Delete supprime la liste avec ses tâches et ses membres.
func (s *TodoListService) Delete(ctx context.Context, userID, listID uuid.UUID) error {
	if _, err := s.access.Require(ctx, userID, listID, models.RoleOwner); err != nil {
		return err
	}

	if err := s.lists.Delete(ctx, listID); err != nil {
		return notFoundOrInternal(err)
	}

	return nil
}
```

`editableTask` vérifie le rôle avant de chercher la tâche : un `viewer`
reçoit le même `403`, que la tâche existe ou non.

`internal/services/task_service.go`

```go
package services

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
)

type TaskService struct {
	tasks  *repositories.TaskRepository
	access *ListAccess
}

func NewTaskService(tasks *repositories.TaskRepository, access *ListAccess) *TaskService {
	return &TaskService{tasks: tasks, access: access}
}

type CreateTaskInput struct {
	UserID uuid.UUID
	ListID uuid.UUID
	Title  string
}

// UpdateTaskInput laisse inchangé un champ qui vaut nil.
type UpdateTaskInput struct {
	UserID    uuid.UUID
	ListID    uuid.UUID
	TaskID    uuid.UUID
	Title     *string
	Completed *bool
}

func (s *TaskService) List(ctx context.Context, userID, listID uuid.UUID, page, perPage int) (Page[models.Task], error) {
	if _, err := s.access.Require(ctx, userID, listID, models.RoleViewer); err != nil {
		return Page[models.Task]{}, err
	}

	tasks, err := findPage(page, perPage, func(page, perPage int) ([]models.Task, int64, error) {
		return s.tasks.FindAllInList(ctx, listID, page, perPage)
	})
	if err != nil {
		return Page[models.Task]{}, apperror.Internal(err)
	}

	return tasks, nil
}

func (s *TaskService) Create(ctx context.Context, input CreateTaskInput) (*models.Task, error) {
	if _, err := s.access.Require(ctx, input.UserID, input.ListID, models.RoleEditor); err != nil {
		return nil, err
	}

	task := &models.Task{ListID: input.ListID, Title: input.Title}
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, apperror.Internal(err)
	}

	return task, nil
}

func (s *TaskService) Update(ctx context.Context, input UpdateTaskInput) (*models.Task, error) {
	task, err := s.editableTask(ctx, input.UserID, input.ListID, input.TaskID)
	if err != nil {
		return nil, err
	}

	applyChanges(task, input, time.Now())
	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, notFoundOrInternal(err)
	}

	return task, nil
}

func (s *TaskService) Delete(ctx context.Context, userID, listID, taskID uuid.UUID) error {
	task, err := s.editableTask(ctx, userID, listID, taskID)
	if err != nil {
		return err
	}

	if err := s.tasks.Delete(ctx, task); err != nil {
		return notFoundOrInternal(err)
	}

	return nil
}

// editableTask vérifie le rôle avant de chercher la tâche : un viewer
// reçoit le même 403, que la tâche existe ou non.
func (s *TaskService) editableTask(ctx context.Context, userID, listID, taskID uuid.UUID) (*models.Task, error) {
	if _, err := s.access.Require(ctx, userID, listID, models.RoleEditor); err != nil {
		return nil, err
	}

	task, err := s.tasks.FindInList(ctx, listID, taskID)
	if err != nil {
		return nil, notFoundOrInternal(err)
	}

	return task, nil
}

func applyChanges(task *models.Task, input UpdateTaskInput, now time.Time) {
	if input.Title != nil {
		task.Title = *input.Title
	}

	if input.Completed != nil {
		task.SetCompleted(*input.Completed, now)
	}
}
```

`internal/services/member_service.go`

```go
package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/francky-d/go-starter-kit/internal/apperror"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
)

type MemberService struct {
	members *repositories.ListMemberRepository
	users   *repositories.UserRepository
	access  *ListAccess
}

func NewMemberService(members *repositories.ListMemberRepository, users *repositories.UserRepository, access *ListAccess) *MemberService {
	return &MemberService{members: members, users: users, access: access}
}

type ShareListInput struct {
	UserID uuid.UUID
	ListID uuid.UUID
	Email  string
	Role   models.Role
}

func (s *MemberService) List(ctx context.Context, userID, listID uuid.UUID) ([]models.ListMember, error) {
	if _, err := s.access.Require(ctx, userID, listID, models.RoleViewer); err != nil {
		return nil, err
	}

	members, err := s.members.FindAllInList(ctx, listID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return members, nil
}

// Share donne un rôle sur la liste à l'utilisateur qui possède l'email, ou
// change le rôle qu'il a déjà.
func (s *MemberService) Share(ctx context.Context, input ShareListInput) (*models.ListMember, error) {
	if _, err := s.access.Require(ctx, input.UserID, input.ListID, models.RoleOwner); err != nil {
		return nil, err
	}

	recipient, err := s.recipient(ctx, input)
	if err != nil {
		return nil, err
	}

	member := &models.ListMember{ListID: input.ListID, UserID: recipient.ID, Role: input.Role, User: recipient}
	if err := s.members.AddOrUpdate(ctx, member); err != nil {
		return nil, apperror.Internal(err)
	}

	return member, nil
}

// Remove permet au propriétaire de retirer n'importe quel membre, et à tout
// membre de quitter la liste.
func (s *MemberService) Remove(ctx context.Context, userID, listID, memberID uuid.UUID) error {
	list, err := s.access.Require(ctx, userID, listID, models.RoleViewer)
	if err != nil {
		return err
	}

	if list.Role != models.RoleOwner && userID != memberID {
		return apperror.Forbidden()
	}

	if err := s.members.Delete(ctx, listID, memberID); err != nil {
		return notFoundOrInternal(err)
	}

	return nil
}

// recipient signale au propriétaire qu'un email n'a pas de compte. Comme
// l'inscription, cela révèle quels emails ont un compte, et le partage
// affiche ensuite le nom du compte, mais seulement à un utilisateur
// connecté qui partage sa propre liste, sous limitation de débit.
func (s *MemberService) recipient(ctx context.Context, input ShareListInput) (*models.User, error) {
	user, err := s.users.FindByEmail(ctx, input.Email)
	if errors.Is(err, repositories.ErrNotFound) {
		return nil, invalidEmail("does not belong to any account")
	}
	if err != nil {
		return nil, apperror.Internal(err)
	}

	if user.ID == input.UserID {
		return nil, invalidEmail("is the owner of the list")
	}

	return user, nil
}

func invalidEmail(message string) error {
	return apperror.Validation(map[string][]string{"email": {message}})
}
```

## Étape 5 : requests

Les identifiants du chemin sont lus par un helper partagé : un identifiant
mal formé donne un `404`, comme un identifiant inconnu.

`internal/http/requests/path.go`

```go
package requests

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/apperror"
)

// PathID lit un paramètre de chemin UUID. Un identifiant mal formé répond
// 404, comme un identifiant qui ne correspond à rien : les deux cas sont
// indiscernables.
func PathID(c echo.Context, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		return uuid.Nil, apperror.NotFound().Wrap(err)
	}

	return id, nil
}

// PathIDs lit deux paramètres de chemin UUID, par exemple une liste et l'une
// de ses tâches.
func PathIDs(c echo.Context, first, second string) (uuid.UUID, uuid.UUID, error) {
	firstID, err := PathID(c, first)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	secondID, err := PathID(c, second)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	return firstID, secondID, nil
}
```

La pagination se lit dans la query string. Les tags `json` ne servent qu'à
nommer les champs dans les erreurs de validation.

`internal/http/requests/pagination.go`

```go
package requests

// Pagination est la query string d'une collection paginée. Une valeur
// absente prend la page et la taille de page par défaut. Les tags json ne
// servent qu'à nommer les champs dans les erreurs de validation.
type Pagination struct {
	Page    int `json:"page" query:"page" validate:"omitempty,min=1"`
	PerPage int `json:"per_page" query:"per_page" validate:"omitempty,min=1,max=100"`
}
```

`internal/http/requests/todo.go`

```go
package requests

type CreateTodoList struct {
	Title string `json:"title" validate:"required,max=200"`
}

type CreateTask struct {
	Title string `json:"title" validate:"required,max=200"`
}

// UpdateTask utilise des pointeurs : un champ que le client omet reste
// inchangé, tandis que "completed": false rouvre la tâche.
type UpdateTask struct {
	Title     *string `json:"title" validate:"omitnil,min=1,max=200"`
	Completed *bool   `json:"completed"`
}

// ShareTodoList ne peut pas accorder le rôle owner : une liste a exactement
// un propriétaire.
type ShareTodoList struct {
	Email string `json:"email" validate:"required,email,max=254"`
	Role  string `json:"role" validate:"required,oneof=viewer editor"`
}
```

## Étape 6 : resources

`internal/http/resources/collection.go`

```go
package resources

// collection met en forme chaque élément d'une slice. Une slice vide reste
// un tableau JSON vide, jamais null.
func collection[Record, Resource any](records []Record, shape func(*Record) Resource) []Resource {
	shaped := make([]Resource, 0, len(records))
	for i := range records {
		shaped = append(shaped, shape(&records[i]))
	}

	return shaped
}
```

`internal/http/resources/todo.go`

```go
package resources

import (
	"time"

	"github.com/francky-d/go-starter-kit/internal/models"
)

type TodoList struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	OwnerID   string    `json:"owner_id"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

func NewTodoList(list *models.TodoList) TodoList {
	return TodoList{
		ID:        list.ID.String(),
		Title:     list.Title,
		OwnerID:   list.OwnerID.String(),
		Role:      string(list.Role),
		CreatedAt: list.CreatedAt,
	}
}

func NewTodoLists(lists []models.TodoList) []TodoList {
	return collection(lists, NewTodoList)
}

type Task struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func NewTask(task *models.Task) Task {
	return Task{
		ID:          task.ID.String(),
		Title:       task.Title,
		Completed:   task.IsCompleted(),
		CompletedAt: task.CompletedAt,
		CreatedAt:   task.CreatedAt,
	}
}

func NewTasks(tasks []models.Task) []Task {
	return collection(tasks, NewTask)
}

// ListMember omet l'email : les membres d'une liste partagée voient les noms
// des uns et des autres, pas leurs adresses.
type ListMember struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Role   string `json:"role"`
}

func NewListMember(member *models.ListMember) ListMember {
	return ListMember{
		UserID: member.UserID.String(),
		Name:   member.User.Name,
		Role:   string(member.Role),
	}
}

func NewListMembers(members []models.ListMember) []ListMember {
	return collection(members, NewListMember)
}
```

## Étape 7 : contrôleurs

Les contrôleurs restent minces : lire l'entrée, appeler un service,
répondre.

`internal/http/controllers/pagination.go`

```go
package controllers

import (
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/services"
)

func pagination[T any](page services.Page[T]) respond.Pagination {
	return respond.Pagination{Page: page.Number, PerPage: page.Size, Total: page.Total}
}
```

`internal/http/controllers/todo_list_controller.go`

```go
package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/requests"
	"github.com/francky-d/go-starter-kit/internal/http/resources"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/services"
)

type TodoListController struct {
	lists *services.TodoListService
}

func NewTodoListController(lists *services.TodoListService) *TodoListController {
	return &TodoListController{lists: lists}
}

func (lc *TodoListController) List(c echo.Context) error {
	var query requests.Pagination
	if err := requests.Bind(c, &query); err != nil {
		return err
	}

	page, err := lc.lists.List(c.Request().Context(), middleware.CurrentUserID(c), query.Page, query.PerPage)
	if err != nil {
		return err
	}

	return respond.Paginated(c, resources.NewTodoLists(page.Items), pagination(page))
}

func (lc *TodoListController) Create(c echo.Context) error {
	var input requests.CreateTodoList
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	list, err := lc.lists.Create(c.Request().Context(), services.CreateTodoListInput{
		OwnerID: middleware.CurrentUserID(c),
		Title:   input.Title,
	})
	if err != nil {
		return err
	}

	return respond.Created(c, resources.NewTodoList(list))
}

func (lc *TodoListController) Show(c echo.Context) error {
	listID, err := requests.PathID(c, "list")
	if err != nil {
		return err
	}

	list, err := lc.lists.Show(c.Request().Context(), middleware.CurrentUserID(c), listID)
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewTodoList(list))
}

func (lc *TodoListController) Delete(c echo.Context) error {
	listID, err := requests.PathID(c, "list")
	if err != nil {
		return err
	}

	if err := lc.lists.Delete(c.Request().Context(), middleware.CurrentUserID(c), listID); err != nil {
		return err
	}

	return respond.NoContent(c)
}
```

`internal/http/controllers/task_controller.go`

```go
package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/requests"
	"github.com/francky-d/go-starter-kit/internal/http/resources"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/services"
)

type TaskController struct {
	tasks *services.TaskService
}

func NewTaskController(tasks *services.TaskService) *TaskController {
	return &TaskController{tasks: tasks}
}

func (tc *TaskController) List(c echo.Context) error {
	listID, err := requests.PathID(c, "list")
	if err != nil {
		return err
	}

	var query requests.Pagination
	if err := requests.Bind(c, &query); err != nil {
		return err
	}

	page, err := tc.tasks.List(c.Request().Context(), middleware.CurrentUserID(c), listID, query.Page, query.PerPage)
	if err != nil {
		return err
	}

	return respond.Paginated(c, resources.NewTasks(page.Items), pagination(page))
}

func (tc *TaskController) Create(c echo.Context) error {
	listID, err := requests.PathID(c, "list")
	if err != nil {
		return err
	}

	var input requests.CreateTask
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	task, err := tc.tasks.Create(c.Request().Context(), services.CreateTaskInput{
		UserID: middleware.CurrentUserID(c),
		ListID: listID,
		Title:  input.Title,
	})
	if err != nil {
		return err
	}

	return respond.Created(c, resources.NewTask(task))
}

func (tc *TaskController) Update(c echo.Context) error {
	listID, taskID, err := requests.PathIDs(c, "list", "task")
	if err != nil {
		return err
	}

	var input requests.UpdateTask
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	task, err := tc.tasks.Update(c.Request().Context(), services.UpdateTaskInput{
		UserID:    middleware.CurrentUserID(c),
		ListID:    listID,
		TaskID:    taskID,
		Title:     input.Title,
		Completed: input.Completed,
	})
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewTask(task))
}

func (tc *TaskController) Delete(c echo.Context) error {
	listID, taskID, err := requests.PathIDs(c, "list", "task")
	if err != nil {
		return err
	}

	if err := tc.tasks.Delete(c.Request().Context(), middleware.CurrentUserID(c), listID, taskID); err != nil {
		return err
	}

	return respond.NoContent(c)
}
```

`internal/http/controllers/member_controller.go`

```go
package controllers

import (
	"github.com/labstack/echo/v4"

	"github.com/francky-d/go-starter-kit/internal/http/middleware"
	"github.com/francky-d/go-starter-kit/internal/http/requests"
	"github.com/francky-d/go-starter-kit/internal/http/resources"
	"github.com/francky-d/go-starter-kit/internal/http/respond"
	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/services"
)

type MemberController struct {
	members *services.MemberService
}

func NewMemberController(members *services.MemberService) *MemberController {
	return &MemberController{members: members}
}

func (mc *MemberController) List(c echo.Context) error {
	listID, err := requests.PathID(c, "list")
	if err != nil {
		return err
	}

	members, err := mc.members.List(c.Request().Context(), middleware.CurrentUserID(c), listID)
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewListMembers(members))
}

func (mc *MemberController) Share(c echo.Context) error {
	listID, err := requests.PathID(c, "list")
	if err != nil {
		return err
	}

	var input requests.ShareTodoList
	if err := requests.Bind(c, &input); err != nil {
		return err
	}

	member, err := mc.members.Share(c.Request().Context(), services.ShareListInput{
		UserID: middleware.CurrentUserID(c),
		ListID: listID,
		Email:  input.Email,
		Role:   models.Role(input.Role),
	})
	if err != nil {
		return err
	}

	return respond.OK(c, resources.NewListMember(member))
}

func (mc *MemberController) Remove(c echo.Context) error {
	listID, memberID, err := requests.PathIDs(c, "list", "user")
	if err != nil {
		return err
	}

	if err := mc.members.Remove(c.Request().Context(), middleware.CurrentUserID(c), listID, memberID); err != nil {
		return err
	}

	return respond.NoContent(c)
}
```

## Étape 8 : routes et câblage

Dans `internal/routes/routes.go`, ajoutez les contrôleurs et la limitation
de débit du partage aux dépendances :

```go
type Dependencies struct {
	Health               *controllers.HealthController
	Auth                 *controllers.AuthController
	Users                *controllers.UserController
	Lists                *controllers.TodoListController
	Tasks                *controllers.TaskController
	Members              *controllers.MemberController
	Authenticated        echo.MiddlewareFunc
	CredentialsRateLimit echo.MiddlewareFunc
	RefreshRateLimit     echo.MiddlewareFunc
	SharingRateLimit     echo.MiddlewareFunc
}
```

puis déclarez les routes à la fin de `Register`. Toutes sont derrière
`Authenticated` ; le rôle, lui, se vérifie dans les services.

```go
	lists := api.Group("/lists").Middleware(d.Authenticated)

	lists.
		Get("", d.Lists.List).
		Post("", d.Lists.Create).
		Get("/:list", d.Lists.Show).
		Delete("/:list", d.Lists.Delete)

	lists.
		Get("/:list/tasks", d.Tasks.List).
		Post("/:list/tasks", d.Tasks.Create).
		Patch("/:list/tasks/:task", d.Tasks.Update).
		Delete("/:list/tasks/:task", d.Tasks.Delete)

	lists.
		Get("/:list/members", d.Members.List).
		// Le partage révèle si un email a un compte, comme
		// l'inscription : il est donc lui aussi soumis à la
		// limitation de débit.
		Put("/:list/members", d.Members.Share, d.SharingRateLimit).
		Delete("/:list/members/:user", d.Members.Remove)
```

Dans `internal/app/dependencies.go`, construisez les contrôleurs. Ils
partagent un seul `ListAccess` :

```go
// todoControllers partagent un seul ListAccess : chaque action sur les to-do
// lists vérifie les droits de la même façon.
func (a *App) todoControllers(users *repositories.UserRepository) (*controllers.TodoListController, *controllers.TaskController, *controllers.MemberController) {
	lists := repositories.NewTodoListRepository(a.database)
	access := services.NewListAccess(lists)

	return controllers.NewTodoListController(services.NewTodoListService(lists, access)),
		controllers.NewTaskController(services.NewTaskService(repositories.NewTaskRepository(a.database), access)),
		controllers.NewMemberController(services.NewMemberService(repositories.NewListMemberRepository(a.database), users, access))
}
```

puis branchez-les dans `dependencies()`. Le partage reprend la limite de
l'inscription et de la connexion, `AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE`,
avec son propre compteur ; ajoutez une variable dédiée si elles doivent
différer.

```go
	lists, tasks, members := a.todoControllers(users)

	return routes.Dependencies{
		Health:               controllers.NewHealthController(pool),
		Auth:                 controllers.NewAuthController(authService),
		Users:                controllers.NewUserController(services.NewUserService(users)),
		Lists:                lists,
		Tasks:                tasks,
		Members:              members,
		Authenticated:        middleware.Authenticated(accessTokens),
		CredentialsRateLimit: middleware.RateLimit(a.config.Auth.CredentialsRateLimitPerMinute),
		RefreshRateLimit:     middleware.RateLimit(a.config.Auth.RefreshRateLimitPerMinute),
		SharingRateLimit:     middleware.RateLimit(a.config.Auth.CredentialsRateLimitPerMinute),
	}, nil
```

## Étape 9 : tests

Le client de test du kit ne connaît que `GET` et `POST`. Ajoutez à
`internal/testutil/http.go` les autres méthodes, et un décodeur du membre
`meta` des réponses paginées :

```go
func (c *Client) Put(path string, body any) *Response {
	c.t.Helper()
	return c.send(http.MethodPut, path, c.encode(body))
}

func (c *Client) Patch(path string, body any) *Response {
	c.t.Helper()
	return c.send(http.MethodPatch, path, c.encode(body))
}

func (c *Client) Delete(path string) *Response {
	c.t.Helper()
	return c.send(http.MethodDelete, path, nil)
}
```

```go
// Meta décode le membre meta d'une réponse paginée dans target.
func (r *Response) Meta(target any) {
	r.t.Helper()
	var envelope struct {
		Meta json.RawMessage `json:"meta"`
	}
	require.NoError(r.t, json.Unmarshal(r.Body.Bytes(), &envelope), r.Body.String())
	require.NoError(r.t, json.Unmarshal(envelope.Meta, target), r.Body.String())
}
```

### Modèles

`internal/models/role_test.go`

```go
package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/models"
)

func TestRole_IncludesTheRightsOfTheRolesBelowIt(t *testing.T) {
	assert.True(t, models.RoleOwner.AtLeast(models.RoleEditor))
	assert.True(t, models.RoleEditor.AtLeast(models.RoleEditor))
	assert.True(t, models.RoleEditor.AtLeast(models.RoleViewer))
	assert.False(t, models.RoleViewer.AtLeast(models.RoleEditor))
	assert.False(t, models.RoleEditor.AtLeast(models.RoleOwner))
}

func TestRole_UnknownRoleGrantsNothing(t *testing.T) {
	assert.False(t, models.Role("").AtLeast(models.RoleViewer))
	assert.False(t, models.Role("admin").AtLeast(models.RoleViewer))
}
```

`internal/models/task_test.go`

```go
package models_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/models"
)

func TestTask_SetCompletedKeepsTheFirstCompletionTime(t *testing.T) {
	first := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	task := &models.Task{}

	task.SetCompleted(true, first)
	task.SetCompleted(true, first.Add(time.Hour))

	require.True(t, task.IsCompleted())
	assert.Equal(t, first, *task.CompletedAt)
}

func TestTask_SetCompletedFalseReopensTheTask(t *testing.T) {
	task := &models.Task{}
	task.SetCompleted(true, time.Now())

	task.SetCompleted(false, time.Now())

	assert.False(t, task.IsCompleted())
}
```

### Repositories

Ces tests réutilisent `newUser`, défini dans
`internal/repositories/user_repository_test.go`.

`internal/repositories/todo_list_repository_test.go`

```go
package repositories_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/francky-d/go-starter-kit/internal/models"
	"github.com/francky-d/go-starter-kit/internal/repositories"
	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func createUser(t *testing.T, db *gorm.DB, email string) *models.User {
	t.Helper()
	user := newUser(email)
	require.NoError(t, repositories.NewUserRepository(db).Create(t.Context(), user))

	return user
}

func createList(t *testing.T, db *gorm.DB, owner *models.User, title string) *models.TodoList {
	t.Helper()
	list := &models.TodoList{OwnerID: owner.ID, Title: title}
	require.NoError(t, repositories.NewTodoListRepository(db).Create(t.Context(), list))

	return list
}

func addMember(t *testing.T, db *gorm.DB, list *models.TodoList, user *models.User, role models.Role) {
	t.Helper()
	member := &models.ListMember{ListID: list.ID, UserID: user.ID, Role: role}
	require.NoError(t, repositories.NewListMemberRepository(db).AddOrUpdate(t.Context(), member))
}

func TestTodoListRepository_FindsTheListsAUserOwnsOrIsAMemberOf(t *testing.T) {
	db := testutil.DB(t)
	ada, grace := createUser(t, db, "ada@example.com"), createUser(t, db, "grace@example.com")
	owned := createList(t, db, ada, "Groceries")
	shared := createList(t, db, grace, "Holidays")
	createList(t, db, grace, "Private")
	addMember(t, db, shared, ada, models.RoleEditor)

	lists, total, err := repositories.NewTodoListRepository(db).FindAllForUser(t.Context(), ada.ID, 1, 20)

	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, lists, 2)
	assert.Equal(t, shared.ID, lists[0].ID)
	assert.Equal(t, models.RoleEditor, lists[0].Role)
	assert.Equal(t, owned.ID, lists[1].ID)
	assert.Equal(t, models.RoleOwner, lists[1].Role)
}

func TestTodoListRepository_HidesAListFromAUserWithoutAccess(t *testing.T) {
	db := testutil.DB(t)
	ada, grace := createUser(t, db, "ada@example.com"), createUser(t, db, "grace@example.com")
	list := createList(t, db, ada, "Groceries")

	_, err := repositories.NewTodoListRepository(db).FindForUser(t.Context(), list.ID, grace.ID)

	assert.ErrorIs(t, err, repositories.ErrNotFound)
}

func TestListMemberRepository_AddOrUpdateChangesTheRoleOfAnExistingMember(t *testing.T) {
	db := testutil.DB(t)
	ada, grace := createUser(t, db, "ada@example.com"), createUser(t, db, "grace@example.com")
	list := createList(t, db, ada, "Groceries")
	addMember(t, db, list, grace, models.RoleViewer)

	addMember(t, db, list, grace, models.RoleEditor)

	found, err := repositories.NewTodoListRepository(db).FindForUser(t.Context(), list.ID, grace.ID)
	require.NoError(t, err)
	assert.Equal(t, models.RoleEditor, found.Role)
	members, err := repositories.NewListMemberRepository(db).FindAllInList(t.Context(), list.ID)
	require.NoError(t, err)
	require.Len(t, members, 1)
	assert.Equal(t, "grace@example.com", members[0].User.Email)
}
```

### Endpoints

Les tests d'endpoint réutilisent `newClient`, `newClientWith`,
`newClientAndDB` et `register`, définis dans
`internal/http/controllers/auth_controller_test.go`. `sharedList` prépare
le cas complet : un propriétaire, un `editor`, un `viewer`, et un
utilisateur sans accès.

`internal/http/controllers/todo_helpers_test.go`

```go
package controllers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/testutil"
)

type listData struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	OwnerID string `json:"owner_id"`
	Role    string `json:"role"`
}

type pageMeta struct {
	Page     int `json:"page"`
	PerPage  int `json:"per_page"`
	Total    int `json:"total"`
	LastPage int `json:"last_page"`
}

type taskData struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`
}

func createList(t *testing.T, client *testutil.Client, title string) listData {
	t.Helper()
	response := client.Post("/api/v1/lists", map[string]string{"title": title})
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var list listData
	response.Data(&list)

	return list
}

func createTask(t *testing.T, client *testutil.Client, listID, title string) taskData {
	t.Helper()
	response := client.Post("/api/v1/lists/"+listID+"/tasks", map[string]string{"title": title})
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var task taskData
	response.Data(&task)

	return task
}

func share(t *testing.T, owner *testutil.Client, listID, email, role string) {
	t.Helper()
	response := owner.Put("/api/v1/lists/"+listID+"/members", map[string]string{"email": email, "role": role})
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
}

// sharedList est une liste avec une tâche, partagée avec un editor et un
// viewer ; un quatrième utilisateur, stranger, n'y a pas accès.
type sharedList struct {
	listID  string
	taskID  string
	userIDs map[string]string
	clients map[string]*testutil.Client
}

func newSharedList(t *testing.T) sharedList {
	t.Helper()
	client := newClient(t)
	userIDs := map[string]string{}
	clients := map[string]*testutil.Client{}
	for _, actor := range []string{"owner", "editor", "viewer", "stranger"} {
		registered := register(t, client, actor+"@example.com")
		userIDs[actor] = registered.User.ID
		clients[actor] = client.WithToken(registered.AccessToken)
	}

	list := createList(t, clients["owner"], "Groceries")
	share(t, clients["owner"], list.ID, "editor@example.com", "editor")
	share(t, clients["owner"], list.ID, "viewer@example.com", "viewer")

	return sharedList{
		listID:  list.ID,
		taskID:  createTask(t, clients["owner"], list.ID, "Milk").ID,
		userIDs: userIDs,
		clients: clients,
	}
}

func (l sharedList) path(suffix string) string {
	return "/api/v1/lists/" + l.listID + suffix
}
```

Le test des droits rejoue chaque action avec chaque utilisateur, sur une
liste neuve à chaque fois : c'est la matrice des droits, sous forme
exécutable.

`internal/http/controllers/todo_rights_test.go`

```go
package controllers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/francky-d/go-starter-kit/internal/testutil"
)

type statuses struct {
	owner, editor, viewer, stranger int
}

func (s statuses) byActor() map[string]int {
	return map[string]int{"owner": s.owner, "editor": s.editor, "viewer": s.viewer, "stranger": s.stranger}
}

// Chaque cas s'exécute une fois par acteur, sur une nouvelle liste
// partagée : stranger reçoit toujours un 404, jamais un 403, si bien que
// l'existence d'une liste ne fuit jamais.
func TestTodoLists_GiveEachRoleItsRights(t *testing.T) {
	cases := []struct {
		action string
		call   func(client *testutil.Client, list sharedList) *testutil.Response
		want   statuses
	}{
		{"show the list", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Get(l.path(""))
		}, statuses{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusNotFound}},
		{"list the tasks", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Get(l.path("/tasks"))
		}, statuses{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusNotFound}},
		{"list the members", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Get(l.path("/members"))
		}, statuses{http.StatusOK, http.StatusOK, http.StatusOK, http.StatusNotFound}},
		{"create a task", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Post(l.path("/tasks"), map[string]string{"title": "Bread"})
		}, statuses{http.StatusCreated, http.StatusCreated, http.StatusForbidden, http.StatusNotFound}},
		{"complete a task", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Patch(l.path("/tasks/"+l.taskID), map[string]bool{"completed": true})
		}, statuses{http.StatusOK, http.StatusOK, http.StatusForbidden, http.StatusNotFound}},
		{"delete a task", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Delete(l.path("/tasks/" + l.taskID))
		}, statuses{http.StatusNoContent, http.StatusNoContent, http.StatusForbidden, http.StatusNotFound}},
		{"share the list", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Put(l.path("/members"), map[string]string{"email": "stranger@example.com", "role": "viewer"})
		}, statuses{http.StatusOK, http.StatusForbidden, http.StatusForbidden, http.StatusNotFound}},
		{"remove the viewer", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Delete(l.path("/members/" + l.userIDs["viewer"]))
		}, statuses{http.StatusNoContent, http.StatusForbidden, http.StatusNoContent, http.StatusNotFound}},
		{"delete the list", func(c *testutil.Client, l sharedList) *testutil.Response {
			return c.Delete(l.path(""))
		}, statuses{http.StatusNoContent, http.StatusForbidden, http.StatusForbidden, http.StatusNotFound}},
	}

	for _, tc := range cases {
		for actor, want := range tc.want.byActor() {
			t.Run(tc.action+" as "+actor, func(t *testing.T) {
				list := newSharedList(t)

				response := tc.call(list.clients[actor], list)

				assert.Equal(t, want, response.Code, response.Body.String())
			})
		}
	}
}
```

`internal/http/controllers/todo_list_controller_test.go`

```go
package controllers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func TestTodoLists_ListTheListsOfTheUserWithTheirRole(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	grace := client.WithToken(register(t, client, "grace@example.com").AccessToken)
	created := createList(t, ada, "Groceries")
	createList(t, ada, "Books")
	shared := createList(t, grace, "Holidays")
	share(t, grace, shared.ID, "ada@example.com", "viewer")
	createList(t, grace, "Private")

	response := ada.Get("/api/v1/lists?per_page=2")

	assert.Equal(t, "owner", created.Role)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var meta pageMeta
	response.Meta(&meta)
	assert.Equal(t, pageMeta{Page: 1, PerPage: 2, Total: 3, LastPage: 2}, meta)
	var lists []listData
	response.Data(&lists)
	require.Len(t, lists, 2)
	assert.Equal(t, "Holidays", lists[0].Title)
	assert.Equal(t, "viewer", lists[0].Role)
	assert.Equal(t, "Books", lists[1].Title)
	assert.Equal(t, "owner", lists[1].Role)
}

func TestTodoLists_DeletingAListDeletesItsTasksAndMembers(t *testing.T) {
	client, db := newClientAndDB(t, testutil.Config())
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	register(t, client, "grace@example.com")
	list := createList(t, ada, "Groceries")
	createTask(t, ada, list.ID, "Milk")
	share(t, ada, list.ID, "grace@example.com", "viewer")

	response := ada.Delete("/api/v1/lists/" + list.ID)

	require.Equal(t, http.StatusNoContent, response.Code)
	var tasks, members int64
	require.NoError(t, db.Table("tasks").Where("list_id = ?", list.ID).Count(&tasks).Error)
	require.NoError(t, db.Table("list_members").Where("list_id = ?", list.ID).Count(&members).Error)
	assert.Zero(t, tasks)
	assert.Zero(t, members)
}

func TestTodoLists_RequireAnAccessToken(t *testing.T) {
	response := newClient(t).Get("/api/v1/lists")

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestTodoLists_AnswerNotFoundForAMalformedID(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)

	response := ada.Get("/api/v1/lists/not-a-uuid")

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestTodoLists_ValidateTheTitleAndThePageSize(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)

	missingTitle := ada.Post("/api/v1/lists", map[string]string{})
	pageTooLarge := ada.Get("/api/v1/lists?per_page=101")

	assert.Equal(t, http.StatusUnprocessableEntity, missingTitle.Code)
	assert.Equal(t, []string{"is required"}, missingTitle.ErrorBody().Errors["title"])
	assert.Equal(t, http.StatusUnprocessableEntity, pageTooLarge.Code)
	assert.Equal(t, []string{"must be at most 100"}, pageTooLarge.ErrorBody().Errors["per_page"])
}
```

`internal/http/controllers/task_controller_test.go`

```go
package controllers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTasks_CompleteRenameAndReopenATask(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	list := createList(t, ada, "Groceries")
	path := "/api/v1/lists/" + list.ID + "/tasks/" + createTask(t, ada, list.ID, "Milk").ID

	var completed, renamed, reopened taskData
	ada.Patch(path, map[string]bool{"completed": true}).Data(&completed)
	ada.Patch(path, map[string]string{"title": "Oat milk"}).Data(&renamed)
	ada.Patch(path, map[string]bool{"completed": false}).Data(&reopened)

	assert.True(t, completed.Completed)
	require.NotNil(t, completed.CompletedAt)
	assert.Equal(t, "Oat milk", renamed.Title)
	assert.True(t, renamed.Completed, "a rename keeps the task completed")
	assert.False(t, reopened.Completed)
	assert.Nil(t, reopened.CompletedAt)
}

func TestTasks_RejectAnEmptyTitle(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	list := createList(t, ada, "Groceries")
	task := createTask(t, ada, list.ID, "Milk")

	response := ada.Patch("/api/v1/lists/"+list.ID+"/tasks/"+task.ID, map[string]string{"title": ""})

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Equal(t, []string{"must be at least 1 character"}, response.ErrorBody().Errors["title"])
}

func TestTasks_AnswerNotFoundForATaskOfAnotherList(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	groceries := createList(t, ada, "Groceries")
	books := createList(t, ada, "Books")
	novel := createTask(t, ada, books.ID, "A novel")

	response := ada.Delete("/api/v1/lists/" + groceries.ID + "/tasks/" + novel.ID)

	assert.Equal(t, http.StatusNotFound, response.Code)
}
```

`internal/http/controllers/member_controller_test.go`

```go
package controllers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/francky-d/go-starter-kit/internal/testutil"
)

func TestMembers_ShareRefusesAnUnknownEmailAndTheOwner(t *testing.T) {
	client := newClient(t)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	list := createList(t, ada, "Groceries")

	unknown := ada.Put("/api/v1/lists/"+list.ID+"/members", map[string]string{"email": "nobody@example.com", "role": "viewer"})
	owner := ada.Put("/api/v1/lists/"+list.ID+"/members", map[string]string{"email": "ada@example.com", "role": "editor"})

	assert.Equal(t, http.StatusUnprocessableEntity, unknown.Code)
	assert.Equal(t, []string{"does not belong to any account"}, unknown.ErrorBody().Errors["email"])
	assert.Equal(t, http.StatusUnprocessableEntity, owner.Code)
	assert.Equal(t, []string{"is the owner of the list"}, owner.ErrorBody().Errors["email"])
}

func TestMembers_ShareIsRateLimited(t *testing.T) {
	cfg := testutil.Config()
	cfg.Auth.CredentialsRateLimitPerMinute = 2
	client := newClientWith(t, cfg)
	ada := client.WithToken(register(t, client, "ada@example.com").AccessToken)
	register(t, client, "grace@example.com")
	list := createList(t, ada, "Groceries")

	var codes []int
	for range 3 {
		codes = append(codes, ada.Put("/api/v1/lists/"+list.ID+"/members", map[string]string{"email": "grace@example.com", "role": "viewer"}).Code)
	}

	assert.Equal(t, []int{http.StatusOK, http.StatusOK, http.StatusTooManyRequests}, codes)
}

func TestMembers_ShareCannotGrantTheOwnerRole(t *testing.T) {
	list := newSharedList(t)

	response := list.clients["owner"].Put(list.path("/members"), map[string]string{"email": "editor@example.com", "role": "owner"})

	assert.Equal(t, http.StatusUnprocessableEntity, response.Code)
	assert.Equal(t, []string{"must be one of: viewer editor"}, response.ErrorBody().Errors["role"])
}

func TestMembers_SharingAgainChangesTheRole(t *testing.T) {
	list := newSharedList(t)

	share(t, list.clients["owner"], list.listID, "viewer@example.com", "editor")
	response := list.clients["viewer"].Post(list.path("/tasks"), map[string]string{"title": "Bread"})

	assert.Equal(t, http.StatusCreated, response.Code, response.Body.String())
}

func TestMembers_ListShowsNamesAndRolesButNoEmails(t *testing.T) {
	list := newSharedList(t)

	response := list.clients["viewer"].Get(list.path("/members"))

	require.Equal(t, http.StatusOK, response.Code)
	var members []struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	response.Data(&members)
	require.Len(t, members, 2)
	assert.Equal(t, "editor", members[0].Role)
	assert.Equal(t, "viewer", members[1].Role)
	assert.NotContains(t, response.Body.String(), "@example.com")
}

func TestMembers_ARemovedMemberLosesAccess(t *testing.T) {
	list := newSharedList(t)
	removed := list.clients["owner"].Delete(list.path("/members/" + list.userIDs["viewer"]))
	require.Equal(t, http.StatusNoContent, removed.Code)

	response := list.clients["viewer"].Get(list.path(""))

	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestMembers_AMemberCanLeaveButTheOwnerIsNotAMember(t *testing.T) {
	list := newSharedList(t)

	editorLeaves := list.clients["editor"].Delete(list.path("/members/" + list.userIDs["editor"]))
	ownerLeaves := list.clients["owner"].Delete(list.path("/members/" + list.userIDs["owner"]))

	assert.Equal(t, http.StatusNoContent, editorLeaves.Code)
	assert.Equal(t, http.StatusNotFound, ownerLeaves.Code)
}
```

Lancez le package qui change, depuis `make shell` :

```bash
go test ./internal/http/controllers/...
```

puis toute la suite, depuis votre terminal :

```bash
make lint
make test
```

## Étape 10 : essayer avec curl

Appliquez les migrations, puis démarrez l'API si `make dev` ne tourne pas
déjà :

```bash
make migrate-up
make dev
```

Dans un autre terminal, inscrivez trois utilisateurs et gardez leurs
jetons d'accès (avec `jq`) :

```bash
API=http://localhost:8080/api/v1
signup() {
  curl -s $API/auth/register -H 'Content-Type: application/json' \
    -d "{\"name\":\"$1\",\"email\":\"$2\",\"password\":\"correct horse battery staple\"}"
}
ADA=$(signup "Ada Lovelace" ada@example.com | jq -r .data.access_token)
GRACE_SESSION=$(signup "Grace Hopper" grace@example.com)
GRACE=$(echo "$GRACE_SESSION" | jq -r .data.access_token)
GRACE_ID=$(echo "$GRACE_SESSION" | jq -r .data.user.id)
ALAN=$(signup "Alan Turing" alan@example.com | jq -r .data.access_token)
```

Ada crée une liste et une tâche :

```bash
LIST=$(curl -s $API/lists -H "Authorization: Bearer $ADA" \
  -H 'Content-Type: application/json' -d '{"title":"Courses"}' | jq -r .data.id)
TASK=$(curl -s $API/lists/$LIST/tasks -H "Authorization: Bearer $ADA" \
  -H 'Content-Type: application/json' -d '{"title":"Lait"}' | jq -r .data.id)
```

Elle partage la liste avec Grace, en lecture seule :

```bash
curl -s -X PUT $API/lists/$LIST/members -H "Authorization: Bearer $ADA" \
  -H 'Content-Type: application/json' -d '{"email":"grace@example.com","role":"viewer"}'
```

```json
{
  "data": {
    "user_id": "01a121f8-e2d0-7071-af9c-1cce873ae0d5",
    "name": "Grace Hopper",
    "role": "viewer"
  }
}
```

Grace voit la liste avec son rôle :

```bash
curl -s $API/lists -H "Authorization: Bearer $GRACE"
```

```json
{
  "data": [
    {
      "id": "01a121f8-e429-7da4-8059-42e8590813c2",
      "title": "Courses",
      "owner_id": "01a121f8-e1bc-7904-9f79-f41dada6bf65",
      "role": "viewer",
      "created_at": "2026-10-09T20:42:01.897896+02:00"
    }
  ],
  "meta": {
    "last_page": 1,
    "page": 1,
    "per_page": 20,
    "total": 1
  }
}
```

Mais elle ne peut pas ajouter de tâche, et Alan ne voit pas la liste du
tout :

```bash
curl -s $API/lists/$LIST/tasks -H "Authorization: Bearer $GRACE" \
  -H 'Content-Type: application/json' -d '{"title":"Pain"}'
# {"message":"This action is forbidden."}

curl -s $API/lists/$LIST -H "Authorization: Bearer $ALAN"
# {"message":"Resource not found."}
```

Ada fait de Grace une `editor` en partageant à nouveau ; Grace peut alors
cocher la tâche :

```bash
curl -s -X PUT $API/lists/$LIST/members -H "Authorization: Bearer $ADA" \
  -H 'Content-Type: application/json' -d '{"email":"grace@example.com","role":"editor"}'

curl -s -X PATCH $API/lists/$LIST/tasks/$TASK -H "Authorization: Bearer $GRACE" \
  -H 'Content-Type: application/json' -d '{"completed":true}'
```

```json
{
  "data": {
    "id": "01a121f8-e47a-7cf0-9ad7-9b1b185ff69a",
    "title": "Lait",
    "completed": true,
    "completed_at": "2026-10-09T20:42:02.145523+02:00",
    "created_at": "2026-10-09T20:42:01.978851+02:00"
  }
}
```

Partager avec un email sans compte est refusé :

```bash
curl -s -X PUT $API/lists/$LIST/members -H "Authorization: Bearer $ADA" \
  -H 'Content-Type: application/json' -d '{"email":"nobody@example.com","role":"viewer"}'
```

```json
{
  "message": "The given data was invalid.",
  "errors": {
    "email": ["does not belong to any account"]
  }
}
```

Enfin, Grace quitte la liste et perd l'accès :

```bash
curl -s -o /dev/null -w '%{http_code}\n' -X DELETE \
  $API/lists/$LIST/members/$GRACE_ID -H "Authorization: Bearer $GRACE"
# 204

curl -s $API/lists/$LIST -H "Authorization: Bearer $GRACE"
# {"message":"Resource not found."}
```

Les horodatages portent le fuseau du serveur qui a produit ces réponses.

## Documenter

Dans votre projet, la fonctionnalité n'est pas finie sans sa
documentation (voir le skill `docs`) :

- les endpoints, leurs corps et leurs statuts dans
  [api-reference.md](../api-reference.md) ;
- les trois tables dans [database.md](../database.md) ;
- la fonctionnalité dans la liste du `README.md`.

## Limites

- **Les membres ne sont pas paginés.** Le propriétaire contrôle leur
  nombre ; plafonnez-le, ou paginez comme les tâches, si une liste peut en
  compter beaucoup.
- **La limite du partage compte par IP**, comme celle de l'inscription :
  des utilisateurs derrière un même NAT la partagent (voir
  [authentification](../authentication.md#limitation-de-débit)).
- **Les droits se vérifient au début de chaque requête.** Une requête déjà
  autorisée se termine même si le rôle change pendant qu'elle s'exécute.
- **La propriété ne se transfère pas.** Ce serait une action réservée au
  propriétaire, qui échange les rôles dans une transaction.
- **Pas de renommage de liste.** Il suivrait le modèle de la mise à jour
  d'une tâche, avec le rôle `owner` ou `editor` selon votre choix.

## Et pour un blog ?

Les mêmes briques s'appliquent :

| To-do list | Blog |
| --- | --- |
| `todo_lists.owner_id` | `posts.author_id` |
| `list_members` et ses rôles | des co-auteurs, ou aucun partage |
| `tasks`, rattachées à une liste | `comments`, rattachés à un article |
| `visibleTo` : propriétaire ou membre | publié, ou brouillon de l'auteur |
| `ListAccess.Require` | un `PostAccess` qui distingue lire, commenter et modifier |

Un article publié se lit sans être membre : sa route `GET` peut même se
passer d'`Authenticated`. Le reste du guide s'applique tel quel :
`404` pour ce que l'on ne peut pas voir, `403` pour ce que l'on voit sans
pouvoir le modifier, et un test qui rejoue la matrice des droits.
