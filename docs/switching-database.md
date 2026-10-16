# Changer de base de données

Le kit cible PostgreSQL. Cette page indique ce qui dépend de PostgreSQL, ce
qui n'en dépend pas, et ce qu'il faut changer pour passer à une autre base.

## En bref

- **Vers une autre base SQL** que GORM et golang-migrate savent piloter
  (MySQL, MariaDB, SQLite, SQL Server) : faisable, et le travail reste dans
  l'infrastructure (connexion, migrations, helpers de test, outillage). Les
  contrôleurs, les services, les routes et l'authentification ne changent
  pas.
- **Vers une base NoSQL** (MongoDB par exemple) : ce n'est plus un
  changement de configuration. GORM ne pilote pas ces bases ; les
  repositories, la transaction du service d'authentification, les
  migrations et les helpers de test sont à réécrire.
- **Le plus tôt est le mieux.** Changer de base avant d'avoir des données
  en production revient à réécrire des fichiers ; après, il faut en plus
  migrer les données, ce que le kit ne prévoit pas.

## Ce qui ne dépend pas de PostgreSQL

| Couche | Pourquoi |
| --- | --- |
| Contrôleurs, requests, resources, routes, middleware | Ils ne voient que des services et des modèles. |
| `internal/auth` | Hash des mots de passe et jetons, sans accès à la base. |
| Services | Ils ne connaissent que les sentinelles des repositories (`ErrNotFound`, `ErrDuplicate`, `ErrInvalid`), jamais une erreur de la base. |
| Requêtes des repositories | Elles passent par le query builder de GORM (`Where`, `Take`, `Updates`, `Offset`, `Limit`), qui génère le SQL du dialecte. |
| Exécution des migrations | Les fichiers sont embarqués avec `embed` et lus par golang-migrate ; seul le driver de la base change. |

Les services ne dépendent pas de PostgreSQL, mais ils dépendent de **GORM**
pour les transactions : `AuthService` reçoit le `*gorm.DB`, ouvre la
transaction de rotation des jetons avec `Transaction`, et la passe aux
repositories par leur méthode `WithTx(tx *gorm.DB)`. Changer de base sans
changer d'ORM ne touche pas ce code ; abandonner GORM le touche.

## Ce qui dépend de PostgreSQL

| Endroit | Dépendance | À faire |
| --- | --- | --- |
| `internal/database/database.go` | Le dialecte `gorm.io/driver/postgres`, et le commentaire du package qui nomme PostgreSQL. | Ouvrir la connexion avec le dialecte de la nouvelle base. |
| `internal/database/migrator.go` | Le driver golang-migrate `database/pgx/v5`, le driver `database/sql` `pgx` et le nom `"postgres"`. | Utiliser le driver golang-migrate de la nouvelle base. |
| `database/migrations/*.sql` | Les types `uuid` et `timestamptz`, la fonction `now()`, la contrainte `CHECK (email = lower(email))`. | Réécrire les migrations dans le SQL de la nouvelle base (voir [Types et SQL](#types-et-sql)). |
| `internal/models/base.go`, `internal/models/refresh_token.go` | Le tag `gorm:"type:uuid"`. | L'adapter au type de colonne choisi pour les identifiants. |
| `internal/testutil/db.go` | `Truncate` lit `pg_tables`, appelle `quote_ident` et lance `TRUNCATE ... CASCADE`. | Réécrire `Truncate` pour la nouvelle base. |
| `internal/testutil/database.go` | L'URL par défaut `postgres://...`. | Mettre l'URL de la base de test locale. |
| `go.mod` | `gorm.io/driver/postgres` et le driver pgx de golang-migrate. | Remplacer ces dépendances. |
| `docker-compose.yml`, `database/init/`, `.github/workflows/ci.yml`, `.gitlab-ci.yml`, `.env.example` | Le service PostgreSQL, la création de la base de test, les services des deux CI, l'exemple de `DATABASE_URL`. | Les remplacer par leurs équivalents. |
| `docs/` | Les pages qui décrivent PostgreSQL, dont [database.md](database.md). | Les mettre à jour dans le même commit. |

## Les pièges

### Traduction des erreurs

La connexion active `TranslateError` : le dialecte GORM traduit les erreurs
de la base en erreurs GORM, que les repositories traduisent à leur tour en
sentinelles. Tous les dialectes ne traduisent pas les mêmes erreurs :

| Dialecte (version lue) | Doublon → `ErrDuplicate` | Contrainte `CHECK` → `ErrInvalid` |
| --- | --- | --- |
| `gorm.io/driver/postgres` v1.6.3 | oui | oui |
| `gorm.io/driver/mysql` v1.6.0 | oui | non |
| `gorm.io/driver/sqlite` v1.6.0 | oui | non |
| `gorm.io/driver/sqlserver` v1.6.4 | oui | non |

Sans traduction, le repository ne reconnaît plus une contrainte `CHECK`
violée : il renvoie l'erreur brute au lieu de la sentinelle `ErrInvalid`.
Aucun service ne s'appuie aujourd'hui sur `ErrInvalid` (une contrainte
violée donne un `500`, avec PostgreSQL aussi), et la seule contrainte
`CHECK`, `users_email_lowercase`, ne sert que de filet de sécurité : le
repository des utilisateurs met les emails en minuscules avant d'écrire.
Mais un service qui s'appuierait sur `ErrInvalid` cesserait de fonctionner.

Deux tests de `internal/database/migrator_test.go` vérifient la traduction
en insérant directement en SQL : `TestUsersTable_RejectsUppercaseEmails`
attend `gorm.ErrCheckConstraintViolated` et
`TestUsersTable_RejectsDuplicateEmails` attend `gorm.ErrDuplicatedKey`.
Sur un dialecte qui ne traduit pas les contraintes `CHECK`, le premier
échoue : c'est le signal attendu. Ne le supprimez qu'après avoir décidé
comment traiter ces violations. De plus, MySQL avant 8.0.16 analyse les
contraintes `CHECK` sans les appliquer.

Le dialecte PostgreSQL garde l'erreur d'origine dans l'erreur traduite ;
les trois autres la remplacent, et les logs perdent le message de la base.

### Lignes affectées

Deux repositories testent le nombre de lignes touchées par un `UPDATE` :
`UpdatePasswordHash` répond `ErrNotFound` quand il vaut 0, et
`RevokeIfActive` ne réussit que s'il vaut 1. MySQL compte par défaut les
lignes **modifiées**, pas les lignes **trouvées** : un `UPDATE` qui réécrit
les mêmes valeurs compte 0. Les deux requêtes changent toujours une valeur,
mais ajoutez `clientFoundRows=true` à l'URL de connexion MySQL pour garder
le sens « lignes trouvées » dans tout nouveau repository.

`RevokeIfActive` repose aussi sur le verrouillage de la ligne pendant un
`UPDATE` : quand deux requêtes échangent le même jeton de rafraîchissement
en même temps, une seule doit réussir. Le test
`TestAuthService_ConcurrentRefreshesWithOneTokenRevokeEverySession` le
vérifie : il doit passer sur la nouvelle base, de préférence avec le
détecteur de courses (`go test -race`).

### Types et SQL

Correspondances pour MySQL, MariaDB et SQLite ; pour une autre base,
cherchez l'équivalent de chaque ligne.

| PostgreSQL | MySQL / MariaDB | SQLite |
| --- | --- | --- |
| `uuid` | `char(36)` : `uuid.UUID` s'enregistre comme une chaîne, et l'ordre des chaînes d'un UUID v7 suit l'ordre de création. | `text` |
| `timestamptz` | `datetime(6)`, en UTC : ajoutez `parseTime=true` et `loc=UTC` à l'URL. | `datetime` |
| `now()` | `CURRENT_TIMESTAMP(6)` | `CURRENT_TIMESTAMP` |
| `text` sous un index unique (`email`, `token_hash`) | `varchar(254)` pour l'email, `char(64)` pour le hash : MySQL n'indexe pas un `text` sans longueur de préfixe. | `text` |
| `ON DELETE CASCADE`, `ON DELETE SET NULL` | supportés par InnoDB | ignorés tant que les clés étrangères ne sont pas activées sur chaque connexion (`PRAGMA foreign_keys = ON`) |

Le tag `gorm:"type:uuid"` des modèles doit suivre le type choisi pour les
identifiants, par exemple `gorm:"type:char(36)"`.

### Verrou des migrations

Avec `DB_AUTO_MIGRATE=true`, plusieurs instances qui démarrent ensemble
comptent sur le verrou que prend golang-migrate pour n'appliquer chaque
migration qu'une fois. Dans golang-migrate v4.20.1 :

| Driver | Verrou |
| --- | --- |
| `pgx/v5` (PostgreSQL) | Verrou consultatif, entre processus. |
| `mysql` | `GET_LOCK`, entre processus. |
| `sqlserver` | `sp_getapplock`, entre processus. |
| `sqlite`, `sqlite3` | Un simple drapeau en mémoire : il ne protège qu'un processus. |

Avec SQLite, appliquez donc les migrations dans un job séparé (voir
[deployment.md](deployment.md#migrations-en-production)), jamais au
démarrage de plusieurs instances.

### Logs des requêtes

Les requêtes journalisées montrent les marqueurs du dialecte : `$1` avec
PostgreSQL, `?` avec MySQL et SQLite. Les valeurs liées restent hors des
logs dans tous les cas.

## Étapes, vers une autre base SQL

1. Remplacez les dépendances de `go.mod` : le dialecte GORM et le driver
   golang-migrate de la nouvelle base.
2. Ouvrez la connexion avec le nouveau dialecte dans
   `internal/database/database.go`, en gardant `TranslateError`.
3. Branchez le driver golang-migrate dans `internal/database/migrator.go`.
4. Réécrivez les migrations. Sans données en production, repartez de
   `000001`.
5. Adaptez les tags `type:uuid` des modèles.
6. Réécrivez `testutil.Truncate` et l'URL de test par défaut.
7. Remplacez le service de base de données de Docker Compose, le script
   de `database/init/`, les services des deux CI et `.env.example`.
8. Lancez `make test` : les tests des repositories, le test de
   rafraîchissement concurrent et les tests des migrations valident le
   changement, dont les deux tests de traduction des contraintes décrits
   dans [Traduction des erreurs](#traduction-des-erreurs).
9. Mettez à jour `docs/`, en commençant par [database.md](database.md) et
   [configuration.md](configuration.md#base-de-données).

## Vers une base NoSQL

| À réécrire | Pourquoi |
| --- | --- |
| `internal/repositories` | Toutes les requêtes passent par GORM. |
| `AuthService.rotate` et les méthodes `WithTx` | La transaction est une transaction GORM. MongoDB n'offre des transactions multi-documents que sur un replica set ou un cluster shardé. |
| `internal/database` et `database/migrations` | Plus de schéma SQL : les index se créent autrement. |
| `internal/models` | Les tags GORM n'ont plus de sens. |
| `internal/testutil` | Vider la base de test se fait autrement. |

Les règles métier des services, la couche HTTP et `internal/auth` restent.
Pour rendre ce passage possible sans toucher aux services, il faudrait que
chaque service dépende d'interfaces qu'il déclare (ses repositories et une
abstraction de transaction) au lieu des types concrets. Le kit ne le fait
pas : ses [conventions](conventions.md) n'ajoutent une interface que
lorsqu'un test ou une seconde implémentation en a besoin.

## Recommandation

Restez sur PostgreSQL tant qu'une contrainte concrète ne l'interdit pas :
hébergement imposé, base existante à réutiliser, compétences de l'équipe.
Si vous devez changer, faites-le avant la mise en production, et gardez le
test de rafraîchissement concurrent vert : c'est lui qui garantit qu'un
jeton volé ne peut pas être échangé deux fois.
