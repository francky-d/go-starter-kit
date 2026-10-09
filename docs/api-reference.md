# Référence de l'API

Cette page décrit le contrat HTTP de l'API : les enveloppes de réponse, le
format des erreurs et chaque endpoint.

## Enveloppe de réponse

Chaque réponse JSON de succès place son contenu dans `data` :

```json
{
  "data": { "id": "0192f1c4-7b7e-7a3e-9a51-3f1f2f1e8c11", "name": "Ada" }
}
```

Les collections paginées ajoutent un objet `meta` :

```json
{
  "data": [{ "id": "..." }, { "id": "..." }],
  "meta": { "page": 2, "per_page": 20, "total": 135, "last_page": 7 }
}
```

Les réponses `204 No Content` n'ont pas de corps.

## En-têtes

Chaque réponse porte :

| En-tête | Valeur |
| --- | --- |
| `X-Request-Id` | Identifiant de la requête, également écrit dans chaque log qui la concerne. Une valeur fournie par le client est conservée si elle compte de 1 à 64 lettres, chiffres, `.`, `_` ou `-` ; sinon une nouvelle est générée. |
| `X-Content-Type-Options` | `nosniff` |
| `X-Frame-Options` | `DENY` |
| `Content-Security-Policy` | `default-src 'none'; frame-ancestors 'none'` |
| `Referrer-Policy` | `no-referrer` |

Envoyez les corps de requête en JSON avec
`Content-Type: application/json`. Les corps plus grands que
`HTTP_BODY_LIMIT` sont rejetés.

## Erreurs

Chaque réponse d'erreur a la même forme. `errors` n'est présent que pour
les échecs de validation et associe à chaque champ invalide ses messages :

```json
{
  "message": "The given data was invalid.",
  "errors": {
    "email": ["must be a valid email address"]
  }
}
```

### Codes de statut

| Statut | Message | Quand |
| --- | --- | --- |
| 400 | `Malformed request body.` | Le corps n'est pas du JSON valide, ou une valeur a le mauvais type. |
| 401 | `Unauthenticated.` | L'authentification est absente ou invalide. La connexion répond `Invalid credentials.` à la place. |
| 403 | `This action is forbidden.` | L'appelant est authentifié mais n'a pas le droit. |
| 404 | `Resource not found.` | La route ou l'enregistrement n'existe pas, ou l'enregistrement appartient à quelqu'un d'autre. |
| 409 | selon l'erreur | La requête entre en conflit avec l'état actuel. |
| 413 | `Request Entity Too Large.` | Le corps dépasse `HTTP_BODY_LIMIT`. |
| 415 | `Unsupported Media Type.` | Le corps n'est pas envoyé en `application/json`. |
| 422 | `The given data was invalid.` | La validation a échoué ; voir `errors`. |
| 429 | `Too many requests.` | Une limite de débit est atteinte. L'en-tête `Retry-After` donne le nombre de secondes à attendre. |
| 500 | `Internal server error.` | Une erreur inattendue s'est produite. |
| 503 | `Service unavailable.` | Une dépendance est indisponible, ou la requête a dépassé `HTTP_REQUEST_TIMEOUT`. |

Les autres statuts produits par la couche HTTP (par exemple `405`) ont pour
message la phrase standard du statut, comme `Method Not Allowed.`.

### Erreurs de validation

Chaque clé de `errors` est le nom JSON d'un champ invalide. Les champs
imbriqués utilisent leur chemin, comme `address.city` ou `items[1].name`.
Un champ peut avoir plusieurs messages.

| Règle | Message |
| --- | --- |
| required | `is required` |
| email | `must be a valid email address` |
| uuid | `must be a valid UUID` |
| url | `must be a valid URL` |
| oneof | `must be one of: <values>` |
| min, gte | `must be at least <n>`, avec `characters` pour du texte et `items` pour des listes |
| max, lte | `must be at most <n>`, avec `characters` pour du texte et `items` pour des listes |
| len | `must be exactly <n>`, avec les mêmes unités |
| toute autre règle | `is invalid` |

Les messages ne répètent jamais la valeur envoyée. Les champs que
l'endpoint n'attend pas sont ignorés.

### Ce qu'une erreur ne contient jamais

Les messages d'erreur sont écrits pour être sûrs quel que soit le client :

- un `404` ne dit pas si l'enregistrement existe ;
- un `403` ne nomme ni le propriétaire ni la permission manquante ;
- un `400` ne cite jamais l'erreur du parseur JSON, qui pourrait répéter
  une partie du corps ;
- un `500` ne porte jamais de message d'erreur interne, de chemin de
  fichier ou de pile d'appels. La cause est écrite dans les logs du
  serveur.

Les requêtes `HEAD` reçoivent le code de statut sans corps.

## Endpoints

| Méthode | Chemin | Authentification | Objet |
| --- | --- | --- | --- |
| `GET` | `/health` | aucune | Indique si l'application et sa base peuvent servir des requêtes. |
| `POST` | `/api/v1/auth/register` | aucune, débit limité | Crée un compte et démarre une session. |
| `POST` | `/api/v1/auth/login` | aucune, débit limité | Démarre une session avec un email et un mot de passe. |
| `POST` | `/api/v1/auth/refresh` | aucune, débit limité | Échange un jeton de rafraîchissement contre une nouvelle session. |
| `POST` | `/api/v1/auth/logout` | aucune, débit limité | Révoque un jeton de rafraîchissement. |
| `GET` | `/api/v1/me` | jeton d'accès bearer | Renvoie l'utilisateur authentifié. |

L'inscription et la connexion acceptent ensemble
`AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE` requêtes par minute et par IP
client ; le rafraîchissement et la déconnexion acceptent ensemble
`AUTH_REFRESH_RATE_LIMIT_PER_MINUTE`. Au-delà, ils répondent `429`. Voir
[authentication.md](authentication.md) pour le fonctionnement des sessions
et des jetons.

### `GET /health`

Répond `200` quand l'application joint sa base en moins de deux secondes :

```bash
curl -s http://localhost:8080/health
```

```json
{ "data": { "status": "ok", "database": "ok" } }
```

Répond `503` avec `{"message":"Service unavailable."}` sinon ; la cause est
écrite dans les logs du serveur.

Utilisez-le comme sonde de disponibilité (**readiness**), pour que le
trafic s'arrête pendant que la base est indisponible. Ne l'utilisez pas
comme sonde de vivacité (liveness) : redémarrer l'API ne répare pas une
panne de la base.

### Sessions

L'inscription, la connexion et le rafraîchissement répondent par une
session :

```json
{
  "data": {
    "user": {
      "id": "0192f1c4-7b7e-7a3e-9a51-3f1f2f1e8c11",
      "name": "Ada Lovelace",
      "email": "ada@example.com",
      "created_at": "2026-10-09T12:00:00Z"
    },
    "token_type": "Bearer",
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 900,
    "refresh_token": "kq3V0n0t2uJx3ZbJ0yq6Q3pW8m1sT4cV5bN7aL9dE2f",
    "refresh_token_expires_at": "2026-11-08T12:00:00Z"
  }
}
```

| Champ | Contenu |
| --- | --- |
| `access_token` | À envoyer sous la forme `Authorization: Bearer <access_token>` aux routes authentifiées. |
| `expires_in` | Secondes restantes avant l'expiration du jeton d'accès. |
| `refresh_token` | À échanger sur `/api/v1/auth/refresh` avant `refresh_token_expires_at`. Chaque jeton de rafraîchissement ne sert qu'une fois. |

Conservez le jeton de rafraîchissement avec le même soin qu'un mot de
passe : qui le détient peut ouvrir une session.

### `POST /api/v1/auth/register`

```bash
curl -s http://localhost:8080/api/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"name":"Ada Lovelace","email":"ada@example.com","password":"correct horse battery staple"}'
```

| Champ | Règles |
| --- | --- |
| `name` | Obligatoire, 100 caractères au plus. |
| `email` | Obligatoire, un email valide, 254 caractères au plus. Stocké en minuscules. |
| `password` | Obligatoire, de 8 à 72 caractères, et 72 octets au plus. |

| Statut | Quand |
| --- | --- |
| `201` | Le compte est créé ; le corps est une session. |
| `409` | L'email a déjà un compte : `Unable to register with the provided details.` |
| `422` | Un champ est invalide. |

### `POST /api/v1/auth/login`

```bash
curl -s http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"ada@example.com","password":"correct horse battery staple"}'
```

| Statut | Quand |
| --- | --- |
| `200` | Le corps est une nouvelle session. |
| `401` | `Invalid credentials.`, que l'email soit inconnu ou le mot de passe faux. |
| `422` | `email` ou `password` manque, ou `email` dépasse 254 caractères. |

### `POST /api/v1/auth/refresh`

```bash
curl -s http://localhost:8080/api/v1/auth/refresh \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh_token>"}'
```

| Statut | Quand |
| --- | --- |
| `200` | Le corps est une nouvelle session ; le jeton envoyé est désormais révoqué. |
| `401` | `Unauthenticated.` : le jeton est inconnu, expiré ou révoqué. Un jeton déjà échangé révoque en plus toutes les sessions de son utilisateur. |
| `422` | `refresh_token` manque ou dépasse 128 caractères. |

### `POST /api/v1/auth/logout`

```bash
curl -s -i http://localhost:8080/api/v1/auth/logout \
  -H 'Content-Type: application/json' \
  -d '{"refresh_token":"<refresh_token>"}'
```

| Statut | Quand |
| --- | --- |
| `204` | Le jeton de rafraîchissement est révoqué. Un jeton inconnu ou déjà révoqué reçoit la même réponse. |
| `422` | `refresh_token` manque ou dépasse 128 caractères. |

Le jeton d'accès reste valide jusqu'à son expiration ; supprimez-le côté
client.

### `GET /api/v1/me`

Renvoie l'utilisateur à qui le jeton d'accès a été délivré.

```bash
curl -s http://localhost:8080/api/v1/me \
  -H 'Authorization: Bearer <access_token>'
```

```json
{
  "data": {
    "id": "0192f1c4-7b7e-7a3e-9a51-3f1f2f1e8c11",
    "name": "Ada Lovelace",
    "email": "ada@example.com",
    "created_at": "2026-10-09T12:00:00Z"
  }
}
```

| Statut | Quand |
| --- | --- |
| `200` | Le corps est l'utilisateur. |
| `401` | `Unauthenticated.` : le jeton est absent, invalide ou expiré, ou son utilisateur n'existe plus. |

Sur un `401`, l'en-tête `WWW-Authenticate` indique s'il faut rafraîchir la
session ou se reconnecter ; voir
[authentication.md](authentication.md#authentifier-les-requêtes).
