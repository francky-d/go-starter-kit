# Configuration

L'application lit sa configuration dans les variables d'environnement au
démarrage. Le package `internal/config` est le seul code qui les lit ; tout
autre package reçoit les valeurs dont il a besoin par son constructeur.

## Résolution des valeurs

1. Une variable définie dans l'environnement du processus l'emporte.
2. Sinon, la valeur d'un fichier `.env` du répertoire de travail est
   utilisée. Le fichier est facultatif.
3. Sinon, la valeur par défaut indiquée ci-dessous s'applique.

Partez du modèle :

```bash
cp .env.example .env
```

`.env` est ignoré par git. Ne commitez jamais de vrais secrets.

## Validation

La configuration est validée une fois, au démarrage. Une valeur invalide
arrête l'application avec une erreur qui liste **tous** les problèmes
trouvés, pour qu'un déploiement se corrige en une seule passe :

```text
config: invalid configuration: APP_ENV must be one of [development test production], got "staging"
HTTP_PORT must be between 1 and 65535, got 0
```

Une valeur du mauvais type (par exemple `HTTP_PORT=http`) est signalée avec
le nom du champ de configuration et la valeur rejetée.

## Variables

Les durées suivent la syntaxe Go : `500ms`, `10s`, `5m`, `1h30m`.

### Application

| Variable | Défaut | Description |
| --- | --- | --- |
| `APP_NAME` | `go-starter-kit` | Nom de l'application. |
| `APP_ENV` | `development` | `development`, `test` ou `production`. |

### Serveur HTTP

| Variable | Défaut | Description |
| --- | --- | --- |
| `HTTP_PORT` | `8080` | Port d'écoute du serveur (de 1 à 65535). |
| `HTTP_READ_HEADER_TIMEOUT` | `5s` | Durée maximale de lecture des en-têtes de la requête. |
| `HTTP_READ_TIMEOUT` | `10s` | Durée maximale de lecture de toute la requête, corps compris. |
| `HTTP_WRITE_TIMEOUT` | `30s` | Durée maximale d'écriture de la réponse. |
| `HTTP_IDLE_TIMEOUT` | `60s` | Durée maximale pendant laquelle une connexion keep-alive attend la requête suivante. |
| `HTTP_SHUTDOWN_TIMEOUT` | `10s` | Durée maximale pour terminer les requêtes en cours à l'arrêt. |
| `HTTP_REQUEST_TIMEOUT` | `15s` | Échéance du contexte de chaque requête ; une requête dont le travail est annulé par l'échéance répond `503`. Doit être inférieure à `HTTP_WRITE_TIMEOUT`. |
| `HTTP_BODY_LIMIT` | `1M` | Taille maximale du corps de requête, comme `512K` ou `2M`. Les corps plus grands répondent `413`. |
| `HTTP_TRUST_PROXY` | `false` | Lit l'IP client dans `X-Forwarded-For`. Voir ci-dessous. |

Tous les délais doivent être positifs.

#### Derrière un reverse proxy

Par défaut, l'IP client est l'adresse du pair TCP. Derrière un load
balancer ou un reverse proxy, c'est l'adresse du proxy : réglez
`HTTP_TRUST_PROXY=true` pour lire l'IP dans `X-Forwarded-For`. L'en-tête
n'est alors pris en compte que si le pair est sur une boucle locale ou un
réseau privé. Ne l'activez jamais quand les clients joignent l'API
directement : ils pourraient choisir l'IP que voient les logs et les
limites de débit.

L'API n'envoie pas d'en-tête `Strict-Transport-Security` : elle s'attend à
ce que le TLS soit terminé par le proxy, qui doit l'envoyer. Si l'API
termine elle-même le TLS, ajoutez un `HSTSMaxAge` au middleware des
en-têtes de sécurité.

### CORS

| Variable | Défaut | Description |
| --- | --- | --- |
| `CORS_ALLOWED_ORIGINS` | *(vide)* | Origines navigateur autorisées à appeler l'API, séparées par des virgules, comme `https://app.example.com, http://localhost:3000`. Chaque entrée vaut `scheme://host[:port]` ou `*`. |

Sans origine configurée, l'API n'envoie aucun en-tête CORS et les
navigateurs sur d'autres origines ne peuvent pas l'appeler. Les
identifiants de navigateur (cookies) ne sont jamais autorisés : les clients
s'authentifient avec un jeton bearer.

### Base de données

| Variable | Défaut | Description |
| --- | --- | --- |
| `DATABASE_URL` | **obligatoire** | URL de connexion PostgreSQL, par exemple `postgres://app:app@localhost:5432/app?sslmode=disable`. |
| `DB_CONNECT_TIMEOUT` | `5s` | Durée maximale pour joindre la base au démarrage. |
| `DB_MAX_OPEN_CONNECTIONS` | `25` | Nombre maximal de connexions ouvertes (au moins 1). |
| `DB_MAX_IDLE_CONNECTIONS` | `25` | Nombre maximal de connexions inactives gardées dans le pool (de 0 à `DB_MAX_OPEN_CONNECTIONS`). |
| `DB_CONNECTION_MAX_LIFETIME` | `30m` | Une connexion est remplacée après cette durée. |
| `DB_CONNECTION_MAX_IDLE_TIME` | `5m` | Une connexion inactive est fermée après cette durée. |
| `DB_SLOW_QUERY_THRESHOLD` | `200ms` | Les requêtes plus lentes sont journalisées en avertissement. |
| `DB_AUTO_MIGRATE` | `false` | Applique les migrations en attente au démarrage de l'API. Voir [database.md](database.md#migrations). |

Utilisez `sslmode=require` (ou plus strict) en production.

### Authentification

| Variable | Défaut | Description |
| --- | --- | --- |
| `BCRYPT_COST` | `12` | Facteur de coût bcrypt des hash de mots de passe, de 4 à 16. Chaque cran double le temps de hachage et de vérification d'un mot de passe ; gardez au moins 12 en production. |
| `JWT_SECRET` | aucun, obligatoire | Clé qui signe les jetons d'accès, d'au moins 32 octets et aléatoire. Générez-la avec `openssl rand -hex 32` et gardez les 64 caractères hexadécimaux ; utilisez-en une différente par environnement et ne la commitez jamais. La changer invalide tous les jetons d'accès déjà délivrés. |
| `JWT_ISSUER` | `go-starter-kit` | Écrit dans chaque jeton d'accès ; un jeton d'un autre émetteur est refusé. |
| `ACCESS_TOKEN_LIFETIME` | `15m` | Durée de validité d'un jeton d'accès. Gardez-la courte : un jeton d'accès ne peut pas être révoqué avant son expiration. |
| `AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE` | `10` | Requêtes par minute que chaque IP client peut envoyer à l'inscription et à la connexion réunies. Voir [authentication.md](authentication.md#limitation-de-débit). |
| `AUTH_REFRESH_RATE_LIMIT_PER_MINUTE` | `60` | Requêtes par minute que chaque IP client peut envoyer au rafraîchissement et à la déconnexion réunis. |
| `REFRESH_TOKEN_LIFETIME` | `720h` | Durée de validité d'un jeton de rafraîchissement (30 jours par défaut). Doit être plus longue que `ACCESS_TOKEN_LIFETIME`. Les durées Go n'ont pas d'unité jour : exprimez les jours en heures. |

### Journalisation

| Variable | Défaut | Description |
| --- | --- | --- |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn` ou `error` (insensible à la casse). |
| `LOG_FORMAT` | `json` | `json` pour les machines, `text` pour les humains. `.env.example` utilise `text`. |

## Ajouter une variable

1. Ajoutez un champ à la bonne struct de `internal/config/config.go`, avec
   un tag `env` et soit un tag `envDefault`, soit l'option `required`
   (`env:"NAME,required"`).
2. Ajoutez une vérification à `Validate` quand certaines valeurs sont
   invalides.
3. Couvrez-la dans `internal/config/config_test.go`, y compris dans la
   liste des variables connues qui sert à vider l'environnement.
4. Ajoutez-la à `.env.example` et à cette page.
