# Base de données

L'API stocke ses données dans PostgreSQL 17 et y accède via
[GORM](https://gorm.io) avec le driver `pgx`. Pour passer à une autre
base, voir [switching-database.md](switching-database.md).

## Base locale

`docker-compose.yml` lance PostgreSQL pour le développement et les tests.
`make dev` et les cibles Go le démarrent au besoin ; ces commandes le
pilotent directement :

```bash
make docker-up      # démarre et attend que la base soit prête
make docker-down    # arrête les conteneurs, conserve les données
make docker-reset   # arrête les conteneurs, supprime les données et les caches Go
```

| Réglage | Valeur |
| --- | --- |
| Hôte et port | `localhost:5432`, joignable depuis cette machine seulement |
| Depuis les conteneurs | `postgres:5432` |
| Utilisateur et mot de passe | `app` / `app` |
| Base de l'application | `app` |
| Base de test | `app_test` |

La base de test est créée par `database/init/01_create_test_database.sql`
au premier démarrage du conteneur sur un volume vide. Si vous avez créé le
volume avant l'existence de ce script, lancez `make docker-reset`.

Ces identifiants servent uniquement en local.

Pour utiliser un autre port sur l'hôte, réglez `DB_FORWARD_PORT` (dans
`.env` ou le shell). Les conteneurs n'en dépendent pas ; avec Go sur
l'hôte (`GO_IN_DOCKER=false`), changez aussi le port dans `DATABASE_URL` et
`TEST_DATABASE_URL`.

## Base de test

Dans le conteneur `go`, `docker-compose.yml` fixe `TEST_DATABASE_URL` sur
la base `app_test` du service `postgres`.

Avec Go sur l'hôte, les tests se connectent par défaut à `app_test` sur
`localhost:5432`. Réglez `TEST_DATABASE_URL` pour utiliser une autre base,
par exemple en CI ou avec un autre port :

```bash
GO_IN_DOCKER=false TEST_DATABASE_URL=postgres://app:app@localhost:5433/app_test?sslmode=disable make test
```

## Migrations

Le schéma est défini par des fichiers SQL versionnés dans
`database/migrations/`, appliqués avec
[golang-migrate](https://github.com/golang-migrate/migrate). Les fichiers
sont embarqués dans les binaires : les appliquer ne demande ni les sources
ni un outil externe.

```bash
make migrate-up                                  # applique toutes les migrations en attente
make migrate-down                                # annule la dernière migration
make migrate-version                             # affiche la version actuelle
make migrate-create name=create_posts_table      # crée une paire up/down vide
```

En développement, le conteneur `api-dev` de `make dev` applique les
migrations en attente à son démarrage, jamais lors d'un rechargement :
appliquez une nouvelle migration avec `make migrate-up` une fois écrite.

Les mêmes commandes existent dans le binaire compilé, par exemple pour un
job de déploiement : `migrate up`, `migrate down [steps]`,
`migrate version`, `migrate force <version>`. Demander à `down` plus
d'étapes qu'il n'y en a d'appliquées annule toutes les migrations et
réussit. Comme l'API, la commande abandonne après `DB_CONNECT_TIMEOUT`
quand la base est injoignable. Elle ne lit que `DATABASE_URL` et les
variables `DB_*` : un job de migration n'a besoin d'aucun secret de l'API.

### Appliquer les migrations au démarrage

Avec `DB_AUTO_MIGRATE=true`, l'API applique les migrations en attente juste
après la connexion et avant de servir des requêtes. golang-migrate prend un
verrou consultatif (advisory lock) PostgreSQL : plusieurs instances qui
démarrent ensemble appliquent chaque migration une seule fois. Laissez
`false` quand les migrations s'exécutent dans une étape de déploiement
séparée.

### Une migration qui échoue

Si une migration échoue à mi-chemin, la version est marquée *dirty* et plus
rien ne s'exécute. Réparez la base à la main, puis indiquez la version
atteinte, depuis `make shell` :

```bash
go run ./cmd/migrate version     # 2 (dirty: ...)
go run ./cmd/migrate force 1     # la dernière version entièrement appliquée
```

### Écrire une migration

- Un changement par migration, avec un fichier `down` qui l'annule
  exactement.
- Ne modifiez jamais une migration appliquée ailleurs que sur votre
  machine ; écrivez-en une nouvelle.
- Les clés primaires sont des `uuid` sans valeur par défaut : l'application
  génère des valeurs UUID v7.
- Voir les [conventions](conventions.md) pour les index et les
  horodatages.

## Schéma

### `users`

| Colonne | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Clé primaire, générée par l'application. |
| `name` | `text` | Obligatoire. |
| `email` | `text` | Obligatoire, unique, stocké en minuscules (contrainte `users_email_lowercase`). |
| `password_hash` | `text` | Obligatoire. |
| `created_at` | `timestamptz` | Vaut `now()` par défaut. |
| `updated_at` | `timestamptz` | Vaut `now()` par défaut. |

### `refresh_tokens`

| Colonne | Type | Notes |
| --- | --- | --- |
| `id` | `uuid` | Clé primaire, générée par l'application. |
| `user_id` | `uuid` | Obligatoire, référence `users`. Supprimer un utilisateur supprime ses jetons. Indexée. |
| `token_hash` | `text` | Obligatoire, unique. SHA-256 du jeton ; le jeton lui-même n'est jamais stocké. |
| `expires_at` | `timestamptz` | Obligatoire. |
| `revoked_at` | `timestamptz` | Renseignée à la révocation du jeton ; vide tant qu'il est actif. |
| `replaced_by_id` | `uuid` | Référence le jeton qui a remplacé celui-ci lors de son échange. Indexée. |
| `created_at` | `timestamptz` | Vaut `now()` par défaut. |
| `updated_at` | `timestamptz` | Vaut `now()` par défaut. |

Les jetons révoqués et expirés sont conservés : c'est en voyant revenir un
jeton révoqué qu'on détecte un jeton volé. Rien ne les supprime pour
l'instant : la table grossit à chaque connexion et à chaque
rafraîchissement. Purgez les lignes dont `expires_at` est passée avec une
tâche planifiée quand la table devient volumineuse.

## Modèles et repositories

Les modèles GORM vivent dans `internal/models`. Chaque modèle embarque
`models.Base`, qui contient l'`id` et les horodatages ; l'identifiant est
un UUID v7 attribué juste avant l'insertion.

Les requêtes vivent dans `internal/repositories`, un repository par
modèle. Un repository traduit les erreurs de la base en ses propres
sentinelles, si bien que les services ne voient jamais une erreur de GORM
ou de la base :

| Sentinelle | Levée quand |
| --- | --- |
| `repositories.ErrNotFound` | Aucun enregistrement ne correspond. |
| `repositories.ErrDuplicate` | Un index unique rejette la valeur, comme un email déjà pris. |
| `repositories.ErrInvalid` | Une contrainte de vérification rejette la valeur. |

Les services dépendent en revanche de GORM pour les transactions :
`AuthService` ouvre une transaction sur le `*gorm.DB` et la passe aux
repositories par leur méthode `WithTx`. Voir
[switching-database.md](switching-database.md) pour ce que cela implique
au moment de changer de base.

Le repository des utilisateurs normalise les emails (sans espaces autour,
en minuscules) quand il crée un utilisateur et quand il en cherche un par
email : la casse d'un email n'a jamais d'importance.

Le repository des jetons de rafraîchissement révoque un jeton avec une mise
à jour conditionnelle qui ne touche qu'un jeton actif. Quand deux requêtes
échangent le même jeton au même moment, une seule le révoque ; l'autre
apprend qu'elle a perdu.

Les listes sont paginées avec le scope `repositories.Paginate(page, perPage)`.
La taille de page vaut 20 par défaut et plafonne à 100.

## Connexion

Au démarrage, l'API ouvre un pool de connexions et envoie un ping à la
base. Si le ping n'aboutit pas avant `DB_CONNECT_TIMEOUT`, l'API s'arrête
avec une erreur au lieu d'échouer plus tard à la première requête.

La taille du pool et la durée de vie des connexions se règlent avec les
variables `DB_*` de [configuration.md](configuration.md#base-de-données).

## Logs des requêtes

Les requêtes plus lentes que `DB_SLOW_QUERY_THRESHOLD` sont journalisées
au niveau WARN, et les requêtes en échec au niveau ERROR, même quand le
client reçoit une erreur 4xx, comme le `409` d'un email déjà pris. Les
requêtes journalisées montrent des marqueurs (`$1`, `$2`) au lieu des
valeurs liées : emails, hash de mots de passe et jetons n'arrivent jamais
dans les logs. Une requête qui ne trouve aucun enregistrement n'est pas
journalisée. Une requête SQL lancée pendant une requête HTTP porte son
`request_id`, comme les autres enregistrements de cette requête.
