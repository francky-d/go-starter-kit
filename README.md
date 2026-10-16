# Go Starter Kit

Un kit de démarrage pour construire des API REST en Go, sur Echo, GORM et
PostgreSQL. Il fournit un module complet d'authentification des
utilisateurs, une suite de tests exécutée contre une vraie base de données,
ainsi que des instructions et des skills pour les agents de code IA.

Un aperçu des routes et d'un contrôleur :

```go
api := r.Group("/api/v1")

authentication := api.Group("/auth")

authentication.Middleware(d.CredentialsRateLimit).
    Post("/register", d.Auth.Register).
    Post("/login", d.Auth.Login)

api.Middleware(d.Authenticated).
    Get("/me", d.Users.Me)
```

```go
func (ac *AuthController) Login(c echo.Context) error {
    var input requests.Login
    if err := requests.Bind(c, &input); err != nil {
        return err
    }

    session, err := ac.auth.Login(c.Request().Context(), input.Email, input.Password)
    if err != nil {
        return err
    }

    return respond.OK(c, resources.NewSession(session))
}
```

## Fonctionnalités

- **HTTP :** Echo v4 derrière un routeur fluide ; une seule enveloppe JSON
  pour chaque succès et chaque erreur ; validation des requêtes avec des
  messages par champ ; identifiant de requête, logs d'accès structurés,
  récupération des panics, en-têtes de sécurité, CORS, taille maximale du
  corps et délai maximal sur chaque requête.
- **Authentification :** inscription, connexion, rafraîchissement,
  déconnexion et utilisateur courant. Jetons d'accès JWT de courte durée ;
  jetons de rafraîchissement opaques stockés sous forme de hash, renouvelés
  à chaque usage, avec une détection de réutilisation qui révoque toutes
  les sessions. Mots de passe bcrypt, réponses identiques pour un email
  inconnu et un mauvais mot de passe, et limitation de débit par IP.
- **Base de données :** PostgreSQL 17 via GORM, identifiants UUID v7,
  migrations SQL versionnées embarquées dans les binaires, et une commande
  de migration.
- **Des erreurs qui ne divulguent rien :** les clients reçoivent des
  messages génériques ; les causes partent dans les logs.
- **Configuration :** typée, validée au démarrage, lue depuis
  l'environnement ou un fichier `.env`.
- **Tests :** tests unitaires, de repository, d'endpoint et de bout en bout
  contre une vraie base PostgreSQL ; rapport de couverture.
- **Outillage :** Makefile, Docker Compose, une image distroless de 28 Mo,
  rechargement à chaud, `staticcheck`, CI pour GitHub Actions et GitLab CI.
- **Prêt pour les agents :** `AGENTS.md`, `CLAUDE.md` et des skills qui
  apprennent à un agent IA les conventions, l'ajout d'une ressource, la
  revue du code, la mise à jour de la documentation et les commits.

## Démarrage rapide

Prérequis : Go 1.26 ou plus récent, Docker avec Compose v2, `make`.

```bash
git clone https://github.com/francky-d/go-starter-kit.git
cd go-starter-kit
cp .env.example .env
```

Renseignez `JWT_SECRET` dans `.env` avec la sortie de
`openssl rand -hex 32`, puis :

```bash
make docker-up      # démarre PostgreSQL
make migrate-up     # crée le schéma
make run            # démarre l'API sur :8080
```

```bash
curl -s http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"correct horse battery staple"}'
```

`make` liste toutes les commandes, et `make test` lance la suite de tests.

## Organisation du projet

```text
cmd/
  api/                 point d'entrée de l'API
  migrate/             commande de migration
internal/
  app/                 câblage des dépendances et serveur HTTP
  config/              configuration typée
  http/                contrôleurs, middleware, requests, resources, respond, router
  routes/              toutes les routes, dans un seul fichier
  services/            règles métier
  repositories/        requêtes, un repository par modèle
  models/              modèles GORM
  auth/                hash des mots de passe, jetons d'accès et de rafraîchissement
  database/            pool de connexions et exécution des migrations
  apperror/            erreurs portant un statut HTTP et un message sûr
  logging/             logger slog
  testutil/            helpers de test
database/
  migrations/          migrations SQL, embarquées dans les binaires
  init/                crée la base de test dans le conteneur PostgreSQL local
docs/                  documentation
.claude/skills/        skills pour les agents IA
.github/workflows/     CI GitHub Actions
.gitlab-ci.yml         CI GitLab
```

## Documentation

| Document | Contenu |
| --- | --- |
| [Prise en main](docs/getting-started.md) | Installation, commandes courantes, rechargement à chaud. |
| [Architecture](docs/architecture.md) | Packages, démarrage et arrêt, cycle de vie d'une requête, routes. |
| [Référence de l'API](docs/api-reference.md) | Enveloppes, erreurs, codes de statut, chaque endpoint. |
| [Authentification](docs/authentication.md) | Mots de passe, jetons, sessions, limitation de débit. |
| [Configuration](docs/configuration.md) | Chaque variable d'environnement. |
| [Base de données](docs/database.md) | PostgreSQL en local, migrations, schéma. |
| [Changer de base de données](docs/switching-database.md) | Ce qui dépend de PostgreSQL, les pièges, les étapes. |
| [Tests](docs/testing.md) | Lancer et écrire les tests, couverture, CI. |
| [Conventions](docs/conventions.md) | Organisation et écriture du code. |
| [Déploiement](docs/deployment.md) | Image Docker, migrations en production, check-list. |
| [Guide : to-do list partagée](docs/guides/todo-list.md) | Une fonctionnalité complète construite sur le kit, avec des rôles et des tests. |

## Travailler avec des agents IA

[AGENTS.md](AGENTS.md) contient les règles que suit chaque agent : la
stack, l'organisation, les commandes, le workflow git, et l'exigence d'une
critique franche et argumentée plutôt qu'un accord par défaut.
[CLAUDE.md](CLAUDE.md) l'importe et indique à Claude Code quand utiliser
chaque skill :

| Skill | Usage |
| --- | --- |
| `conventions` | Avant d'écrire du code Go, une migration ou un test. |
| `scaffold` | Ajouter une ressource, de la migration jusqu'aux routes, aux tests et à la documentation. |
| `review` | Une revue de sécurité et de qualité, priorisée, dans `review/code-review.md`. |
| `docs` | Avant chaque commit : détermine quels documents le changement touche. |
| `commit` | Des commits atomiques au format Conventional Commits, après les vérifications et la documentation. |

Les agents écrivent le code, les commentaires, les messages de commit et
leurs propres fichiers en anglais, et la documentation en français.

Le travail est planifié sur le [tableau de projet](PLAN.md) : une tâche par
commit, chaque tâche passant de `todo` à `in progress` puis à `done`.

## Contribuer

- Une tâche, un commit, avec un message au format
  [Conventional Commits](https://www.conventionalcommits.org).
- `make lint` et `make test` passent avant chaque commit.
- La documentation part dans le même commit que le code qu'elle décrit.

## Licence

Aucune licence n'a encore été choisie.
