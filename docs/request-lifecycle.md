# Cycle de vie d'une requête

Cette page suit une requête HTTP de bout en bout, de la connexion acceptée
par le serveur jusqu'à la réponse que reçoit le client. Elle décrit chaque
couche traversée, dans l'ordre : ce dont elle est responsable, ce qu'elle
peut renvoyer, et comment une erreur remonte jusqu'à devenir la réponse
d'erreur. Lisez-la avant d'ajouter une fonctionnalité.

## Vue d'ensemble

```text
client
 │  connexion TCP
 ▼
http.Server (net/http)    délais de lecture, d'écriture et d'inactivité
 │
 ▼
Echo                      choisit le handler de la route, ou 404 / 405
 │
 ├─ middleware de base     Request ID, Request logger, Recover,
 │                         Secure headers, CORS, Body limit, Request timeout,
 │                         Client disconnect
 ├─ middleware de route    limitation de débit, Authenticated
 ▼
contrôleur                requests.Bind, un appel de service, respond.*
 ▼
service                   règles métier, transactions, *apperror.Error
 ▼
repository                GORM avec le contexte de la requête, sentinelles
 ▼
PostgreSQL
```

La réponse refait le chemin en sens inverse. Une réponse de succès est
écrite par le contrôleur, avec `respond`. Une réponse d'erreur est écrite
par un seul composant, `respond.ErrorHandler`, quelle que soit la couche
qui a échoué.

Chaque couche a un contrat court :

| Couche | Reçoit | Renvoie |
| --- | --- | --- |
| Middleware de route | `echo.Context` | Appelle la suite, ou renvoie une `*apperror.Error` (`401`, `429`). |
| Contrôleur | `echo.Context` | Une réponse écrite par `respond`, ou l'erreur reçue, inchangée. |
| `requests.Bind` | `echo.Context` et un pointeur vers la struct d'entrée | `nil`, ou une `*apperror.Error` (`400`, `413`, `415`, `422`). |
| Service | `ctx` et des valeurs Go, jamais `echo.Context` | Des modèles ou ses propres structs, ou une `*apperror.Error`. |
| Repository | `ctx` et des valeurs Go | Des modèles, ou une sentinelle (`ErrNotFound`, `ErrDuplicate`, `ErrInvalid`) qui enveloppe l'erreur de GORM, ou l'erreur brute. |

Toute la requête s'exécute dans la goroutine que `net/http` attribue à la
connexion : chaque couche appelle la suivante et attend son retour.

## Le serveur HTTP

[app.go](../internal/app/app.go) écoute sur `HTTP_PORT` et sert les
connexions avec un `http.Server` de la bibliothèque standard. Echo n'est
que le `Handler` de ce serveur ; le serveur intégré d'Echo n'est pas
utilisé.

Le serveur borne la durée de chaque connexion avec la
[configuration](configuration.md#serveur-http) :

| Délai | Variable | Borne |
| --- | --- | --- |
| `ReadHeaderTimeout` | `HTTP_READ_HEADER_TIMEOUT` | La lecture des en-têtes. |
| `ReadTimeout` | `HTTP_READ_TIMEOUT` | La lecture de toute la requête, corps compris. |
| `WriteTimeout` | `HTTP_WRITE_TIMEOUT` | Le traitement et l'écriture de la réponse, à partir de la lecture des en-têtes. |
| `IdleTimeout` | `HTTP_IDLE_TIMEOUT` | L'attente de la requête suivante sur une connexion keep-alive. |

La configuration exige que `HTTP_WRITE_TIMEOUT` dépasse
`HTTP_REQUEST_TIMEOUT` : une requête qui atteint son échéance a encore le
temps d'écrire sa réponse. Les erreurs que `net/http` journalise lui-même
passent par le logger `slog`, au niveau WARN.

**IP client.** `middleware.ClientIP` fixe la politique d'IP d'Echo :
l'adresse du pair TCP par défaut, ou `X-Forwarded-For` quand
`HTTP_TRUST_PROXY` est activée (voir
[Derrière un reverse proxy](configuration.md#derrière-un-reverse-proxy)).
Cette IP est celle que renvoie `c.RealIP()` : le champ `remote_ip` des logs
et la clé de la limitation de débit.

**Arrêt.** À la réception de SIGINT ou SIGTERM, le serveur cesse
d'accepter des connexions et laisse aux requêtes en cours jusqu'à
`HTTP_SHUTDOWN_TIMEOUT` pour finir, puis ferme les connexions restantes
(voir [Démarrage et arrêt](architecture.md#démarrage-et-arrêt)). Le
contexte d'une requête ne dérive pas de celui du signal : l'arrêt
n'annule pas une requête en cours.

## Les middleware de base

[Baseline](../internal/http/middleware/baseline.go) renvoie les middleware
que traverse chaque requête, installés avec `e.Use`. Ils enveloppent tous
les handlers, y compris ceux des réponses `404` et `405`. Chacun exécute
du code avant d'appeler la suite, puis au retour : l'aller se fait dans
l'ordre du tableau, le retour dans l'ordre inverse.

| Ordre | Middleware | Rôle | Statut produit |
| --- | --- | --- | --- |
| 1 | Request ID | Conserve un `X-Request-Id` client de 1 à 64 lettres, chiffres, `.`, `_` ou `-` ; sinon en génère un de 32 caractères. L'écrit dans l'en-tête `X-Request-Id` de la réponse. | aucun |
| 2 | Request logger | Au retour, confie une éventuelle erreur à `respond.ErrorHandler`, puis écrit un enregistrement `request` avec le statut final (voir [Ce qui est journalisé](#ce-qui-est-journalisé)). | aucun |
| 3 | Recover | Rattrape un panic des couches inférieures, journalise `panic recovered` avec la pile d'appels, puis appelle `respond.ErrorHandler`. | `500` |
| 4 | Secure headers | Pose `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'`, `Referrer-Policy: no-referrer` et `X-XSS-Protection: 0`, avant d'appeler la suite : les réponses d'erreur les portent aussi. | aucun |
| 5 | CORS | Seulement si `CORS_ALLOWED_ORIGINS` est renseignée. Ajoute les en-têtes CORS pour les origines autorisées et répond lui-même à toute requête `OPTIONS`, qui n'atteint donc jamais la route. | `204` |
| 6 | Body limit | Refuse un `Content-Length` supérieur à `HTTP_BODY_LIMIT`. Sans `Content-Length`, la limite s'applique pendant la lecture du corps (voir [`requests.Bind`](#requestsbind)). | `413` |
| 7 | Request timeout | [request_timeout.go](../internal/http/middleware/request_timeout.go). Remplace le contexte de la requête par un contexte à échéance de `HTTP_REQUEST_TIMEOUT`. Au retour, si cette échéance est dépassée, une erreur qui enveloppe `context.DeadlineExceeded` devient `503 Service unavailable.` ; une échéance que le handler a fixée lui-même garde son statut (voir [Contexte, échéance et annulation](#contexte-échéance-et-annulation)). | `503` |
| 8 | Client disconnect | [client_disconnect.go](../internal/http/middleware/client_disconnect.go). Au retour, si le client a fermé la connexion, une erreur qui enveloppe `context.Canceled` devient `499 Client closed request.` ; une annulation que le handler a faite lui-même garde son statut. Placé à l'intérieur de Request timeout, qui annule le contexte en rendant la main. | `499` |

Le request logger est placé au-dessus de Recover pour journaliser le
statut `500` d'un panic rattrapé.

## Le routage

[routes.go](../internal/routes/routes.go) déclare chaque route avec la
[surcouche du routeur](../internal/http/router/router.go) :

| Méthode | Chemin | Middleware de route | Handler |
| --- | --- | --- | --- |
| `GET` | `/health` | aucun | `HealthController.Show` |
| `POST` | `/api/v1/auth/register` | `CredentialsRateLimit` | `AuthController.Register` |
| `POST` | `/api/v1/auth/login` | `CredentialsRateLimit` | `AuthController.Login` |
| `POST` | `/api/v1/auth/refresh` | `RefreshRateLimit` | `AuthController.Refresh` |
| `POST` | `/api/v1/auth/logout` | `RefreshRateLimit` | `AuthController.Logout` |
| `GET` | `/api/v1/me` | `Authenticated` | `UserController.Me` |

Echo cherche la route d'après la méthode et le chemin dès l'arrivée de la
requête, puis exécute la chaîne : middleware de base, middleware de la
route, handler.

- **Route trouvée.** La surcouche enregistre chaque route avec son chemin
  complet et ses middleware ; elle n'utilise pas les groupes d'Echo. Les
  middleware de groupe s'exécutent avant ceux de la route, dans l'ordre de
  déclaration.
- **Aucune route pour ce chemin.** Echo exécute son handler `404`,
  enveloppé par les seuls middleware de base : la réponse est
  `404 Resource not found.`. Une URL inconnue sous `/api/v1` répond donc
  `404`, jamais `401`.
- **Chemin connu, mauvaise méthode.** La réponse est
  `405 Method Not Allowed.`, avec un en-tête `Allow` qui liste les
  méthodes du chemin, par exemple `Allow: OPTIONS, GET`. Quand CORS est
  désactivé, une requête `OPTIONS` sur un chemin connu répond `204` avec le
  même en-tête.

### Middleware de route

Le kit en a deux, construits dans
[dependencies.go](../internal/app/dependencies.go) et passés aux routes
par `routes.Dependencies`.

**`RateLimit(n)`** ([rate_limit.go](../internal/http/middleware/rate_limit.go))
tient un seau de jetons en mémoire par IP client : `n` requêtes par minute,
avec des rafales de `n`. Chaque appel à `RateLimit` crée ses propres
compteurs : l'inscription et la connexion partagent un budget, le
rafraîchissement et la déconnexion un autre. Toute requête qui atteint la
route consomme un jeton, qu'elle réussisse ou non. Un refus répond
`429 Too many requests.` avec `Retry-After: 60`. Voir
[Limitation de débit](authentication.md#limitation-de-débit).

**`Authenticated(tokens)`** ([authenticate.go](../internal/http/middleware/authenticate.go))
lit l'en-tête `Authorization: Bearer <jeton>` et vérifie le jeton d'accès
(voir [Authentifier les requêtes](authentication.md#authentifier-les-requêtes)).
S'il est valide, il enregistre l'identifiant de l'utilisateur, un
`uuid.UUID`, dans le contexte Echo avec `c.Set`, et appelle la suite.
Le contrôleur le relit avec `middleware.CurrentUserID(c)` et le passe au
service en argument : le `context.Context` de la requête ne le porte pas.
Un refus répond
`401 Unauthenticated.` avec l'en-tête `WWW-Authenticate: Bearer` quand
aucun jeton n'est envoyé, ou `WWW-Authenticate: Bearer error="invalid_token"`
quand le jeton est refusé.

`CurrentUserID` panique sur une route déclarée sans `Authenticated` : c'est
une erreur de câblage, que Recover transforme en `500`.

Un middleware qui refuse une requête renvoie une erreur sans appeler la
suite. Le contrôleur ne s'exécute pas, et l'erreur suit
[le chemin d'une erreur](#le-chemin-dune-erreur).

## Le contrôleur

Un contrôleur ([controllers](../internal/http/controllers/)) est une
méthode `func(c echo.Context) error` sur une struct qui reçoit son service
par son constructeur. Il fait trois choses, toujours dans cet ordre :

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

1. Il lie et valide l'entrée avec `requests.Bind`.
2. Il appelle une méthode de service avec `c.Request().Context()`, qui
   porte l'échéance de la requête, et des valeurs Go. Le service ne voit
   jamais `echo.Context`.
3. Il répond avec `respond`, ou renvoie l'erreur telle quelle.

Un contrôleur n'applique pas de règle métier, n'interroge pas la base,
n'écrit jamais de corps d'erreur et n'appelle pas `c.JSON` directement.

`HealthController` est la seule exception : il n'a pas de service. Il
envoie un ping au pool de connexions à travers une petite interface,
`DatabasePinger`, avec un délai de deux secondes dérivé du contexte de la
requête, et répond `503 Service unavailable.` si le ping échoue.

### `requests.Bind`

[bind.go](../internal/http/requests/bind.go) vérifie que le corps est du
JSON, appelle le binder d'Echo, lit le reste du corps, puis valide la
struct d'après ses tags `validate`. Chaque endpoint déclare sa
propre struct dans [requests](../internal/http/requests/) : un client ne
peut pas remplir un champ que l'endpoint n'attend pas.

Le binder lit d'abord les paramètres de chemin (tags `param`), puis les
paramètres de query string (tags `query`) pour `GET`, `DELETE` et `HEAD`
seulement, puis le corps JSON (tags `json`).

| Situation | Réponse |
| --- | --- |
| Corps vide (`Content-Length: 0`) | Rien n'est lié ; la validation signale les champs obligatoires : `422`. |
| JSON invalide, ou valeur du mauvais type | `400 Malformed request body.` Le message du parseur, qui peut citer le corps, ne va que dans les logs. |
| Corps sans `Content-Length` qui dépasse `HTTP_BODY_LIMIT` | `413 Request Entity Too Large.` Body limit compte les octets pendant la lecture, et `requests.Bind` lit le corps jusqu'au bout, même quand la valeur JSON tient dans la limite. |
| Corps non vide envoyé sans `Content-Type` `application/json` (un paramètre comme `charset` est accepté) | `415 Unsupported Media Type.` |
| Une règle `validate` n'est pas respectée | `422 The given data was invalid.`, avec les messages par champ dans `errors`. |
| L'entrée n'est pas une struct | `500` : c'est une erreur de programmation. |

Le binder d'Echo accepterait aussi XML, lié d'après les noms des champs Go,
et les formulaires : `requests.Bind` les refuse avant de l'appeler. Un
corps vide n'est pas vérifié, si bien qu'une requête `GET` qui ne lie que
sa query string n'a pas besoin de `Content-Type`.

Les clés de `errors` sont les noms JSON des champs, et les messages ne
répètent jamais la valeur envoyée. Pour `POST /api/v1/auth/login` avec le
corps `{}` :

```json
{
  "message": "The given data was invalid.",
  "errors": {
    "email": ["is required"],
    "password": ["is required"]
  }
}
```

La liste des messages est dans la
[référence de l'API](api-reference.md#erreurs-de-validation).

## Le service

Un service ([services](../internal/services/)) porte les règles métier. Il
reçoit `ctx` et des valeurs Go (`services.Registration`, un email, un
`uuid.UUID`), et ne sait rien de HTTP. Exemples de règles :

- `AuthService.Login` donne la même réponse, après le même travail bcrypt,
  que l'email soit inconnu ou le mot de passe faux ;
- `AuthService.Register` refuse un mot de passe de plus de 72 octets,
  hache le mot de passe, crée l'utilisateur et démarre une session ;
- `AuthService.Logout` ignore un jeton inconnu.

Un service traduit chaque erreur qu'il reçoit en `*apperror.Error`. C'est
lui, et lui seul, qui décide du statut HTTP d'une erreur de la base :

| Erreur reçue | Traduction dans le kit |
| --- | --- |
| `repositories.ErrNotFound` | Dépend de la règle métier : `401 Unauthenticated.` dans `UserService.CurrentUser` et `AuthService.Refresh`, `401 Invalid credentials.` dans `AuthService.Login`, aucune erreur dans `AuthService.Logout`. |
| `repositories.ErrDuplicate` | `409 Unable to register with the provided details.` dans `AuthService.Register`. |
| `auth.ErrPasswordTooLong` | `422`, avec `must be at most 72 bytes` sur le champ `password`. |
| `auth.ErrPasswordMismatch` | `401 Invalid credentials.` |
| Toute autre erreur | `apperror.Internal(err)` : `500 Internal server error.`, la cause allant aux logs. |

[apperror.go](../internal/apperror/apperror.go) fournit les constructeurs,
dont `BadRequest`, `Unauthorized`, `Forbidden`, `NotFound`, `Conflict`,
`Validation`, `Internal` et `New` pour un statut et un message libres. Une
`*apperror.Error` porte un statut, un message sûr pour n'importe quel
client, des erreurs par champ pour un `422`, et une cause facultative,
ajoutée avec `Wrap(err)`, qui n'est jamais sérialisée.

**Transactions.** Un service ouvre une transaction quand plusieurs
écritures doivent réussir ensemble. Dans le kit, seul
`AuthService.Refresh` le fait : il ouvre
`s.db.WithContext(ctx).Transaction(...)` et passe la transaction au
repository par `WithTx(tx)`. Dans la transaction, il enregistre le nouveau
jeton de rafraîchissement, puis révoque l'ancien seulement s'il est
encore actif. Si une autre requête l'a révoqué entre-temps, la transaction
est annulée, toutes les sessions de l'utilisateur sont révoquées, et la
réponse est `401`. Toute autre erreur annule aussi la transaction et
répond `500`. Voir
[Rafraîchir une session](authentication.md#rafraîchir-une-session).

Un échec qui ne doit pas faire échouer la requête est journalisé, pas
renvoyé : `AuthService.Login` écrit `password hash upgrade failed` au
niveau WARN quand la mise à niveau d'un hash échoue, et répond quand même
`200`.

## Le repository et la base

Un repository ([repositories](../internal/repositories/)) exécute les
requêtes d'un modèle avec GORM. Chaque méthode prend `ctx` en premier et
commence par `r.db.WithContext(ctx)` : le contexte de la requête, avec son
échéance, atteint le pool de `database/sql` et le driver PostgreSQL. Quand
ce contexte est annulé, l'attente d'une connexion libre ou la requête SQL
en cours s'arrête avec une erreur.

Un repository renvoie des modèles, jamais `nil, nil` : un finder qui ne
trouve rien renvoie `ErrNotFound`. Il traduit les erreurs de GORM avec
`translate` ([errors.go](../internal/repositories/errors.go)) :

| Erreur de GORM | Sentinelle |
| --- | --- |
| `gorm.ErrRecordNotFound` | `repositories.ErrNotFound` |
| `gorm.ErrDuplicatedKey` | `repositories.ErrDuplicate` |
| `gorm.ErrCheckConstraintViolated` | `repositories.ErrInvalid` |
| Toute autre erreur | Renvoyée telle quelle. |

La sentinelle enveloppe l'erreur d'origine : `errors.Is` la reconnaît, et
la cause reste lisible dans les logs. GORM ne produit
`ErrDuplicatedKey` et `ErrCheckConstraintViolated` que parce que la
connexion est ouverte avec `TranslateError: true`
([database.go](../internal/database/database.go)). Certaines méthodes
produisent elles-mêmes leur résultat : `UpdatePasswordHash` renvoie
`ErrNotFound` quand aucune ligne n'est modifiée, et `RevokeIfActive`
indique par un booléen s'il a révoqué le jeton.

Aucun service du kit ne traduit aujourd'hui `ErrInvalid` : une violation
de contrainte de vérification devient un `500`, comme une violation de clé
étrangère ou une base injoignable. Un service qui
s'appuie sur une contrainte de vérification doit traduire `ErrInvalid`
lui-même.

Les modèles ([models](../internal/models/)) embarquent `models.Base`, qui
attribue un UUID v7 juste avant l'insertion ; GORM remplit `CreatedAt` et
`UpdatedAt`. Voir
[Modèles et repositories](database.md#modèles-et-repositories).

## La réponse

Un modèle ne quitte jamais le serveur tel quel. Le contrôleur le passe à
une resource ([resources](../internal/http/resources/)), une struct aux
tags `json` explicites qui choisit les champs envoyés (jamais
`password_hash`), convertit les types (`ID.String()`) et calcule les champs
dérivés : `resources.NewSession` fixe `token_type` à `Bearer` et calcule
`expires_in`, le nombre de secondes avant l'expiration du jeton d'accès.

La resource est ensuite écrite par un helper de
[respond.go](../internal/http/respond/respond.go) :

| Helper | Statut | Corps |
| --- | --- | --- |
| `respond.OK(c, data)` | `200` | `{"data": ...}` |
| `respond.Created(c, data)` | `201` | `{"data": ...}` |
| `respond.NoContent(c)` | `204` | aucun |
| `respond.Paginated(c, data, page)` | `200` | `{"data": [...], "meta": {"page": ..., "per_page": ..., "total": ..., "last_page": ...}}` |

`meta` n'apparaît que lorsqu'il n'est pas vide. Le corps est écrit avec
`Content-Type: application/json`. Voir
[Enveloppe de réponse](api-reference.md#enveloppe-de-réponse).

Une fois écrite, la réponse est engagée : une erreur renvoyée après coup
est ignorée par `respond.ErrorHandler`, ni écrite ni journalisée. Un
contrôleur renvoie donc soit une réponse, soit une erreur, jamais les
deux.

La réponse remonte ensuite les middleware de base : le request logger
écrit son enregistrement `request`, et les en-têtes posés à l'aller
(`X-Request-Id`, en-têtes de sécurité, en-têtes CORS) font partie de la
réponse.

## Le chemin d'une erreur

### Remonter la chaîne

Aucune couche n'écrit de corps d'erreur : chacune renvoie une valeur
d'erreur à la couche du dessus.

```text
repository        return translate(err)     sentinelle, ou erreur brute
service           return apperror.Xxx(...)  *apperror.Error : statut, message, cause
contrôleur        return err                inchangée
middleware        return err                Client disconnect enveloppe une erreur
                                            Canceled dans un 499, Request timeout
                                            une erreur DeadlineExceeded dans un 503
Request logger    c.Error(err)              respond.ErrorHandler écrit la réponse,
                                            puis le logger écrit « request »
Echo              reçoit encore l'erreur    respond.ErrorHandler : réponse déjà
                                            écrite, rien à faire
```

Un panic suit un autre chemin : Recover le rattrape, journalise
`panic recovered` avec la pile d'appels au niveau ERROR, et appelle
lui-même `respond.ErrorHandler`, qui répond `500`. Le request logger ne
voit alors aucune erreur, mais journalise le statut `500` déjà écrit.

### `respond.ErrorHandler`

[errors.go](../internal/http/respond/errors.go) est le seul endroit où
s'écrivent les corps d'erreur :

1. Si la réponse est déjà engagée, il s'arrête, sans rien écrire ni
   journaliser.
2. Il convertit l'erreur en `*apperror.Error` :

   | Erreur | Réponse |
   | --- | --- |
   | Contient une `*apperror.Error`, même enveloppée par `fmt.Errorf("...: %w", err)` ou placée dans le champ `Internal` d'une `*echo.HTTPError` | Son statut, son message et ses erreurs par champ. |
   | `*echo.HTTPError` (routeur, Body limit) | Son statut seulement. Le message d'Echo, qui peut contenir des détails du parseur, est remplacé : message du kit pour `401`, `403`, `404`, `429` et `503`, phrase standard suivie d'un point pour les autres (`Method Not Allowed.`) ; un statut hors 4xx et 5xx devient `500`. |
   | Toute autre erreur | `500 Internal server error.` |

3. Il journalise l'erreur d'origine complète, cause comprise.
4. Il écrit `{"message": "...", "errors": {...}}`, où `errors` n'apparaît
   que pour un échec de validation. Une requête `HEAD` reçoit le statut
   sans corps.

Le client ne reçoit jamais que le statut, le message et les erreurs par
champ. La cause d'une `*apperror.Error`, le message d'une erreur Go ou
d'Echo, la pile d'un panic ne vont que dans les logs. Voir
[Erreurs](api-reference.md#erreurs).

### Ce qui est journalisé

| Enregistrement | Écrit par | Niveau | Attributs |
| --- | --- | --- | --- |
| `request` | Request logger, une fois par requête | INFO ; DEBUG pour une réponse de `/health` sous `400` | `method`, `path` (sans query string), `route` (vide pour une route inconnue), `status`, `latency`, `remote_ip`, `request_id` |
| `request rejected` | `respond.ErrorHandler`, statut 4xx | INFO pour `401`, `403` et `429` ; DEBUG pour les autres | `method`, `path`, `status`, `error`, `request_id` |
| `request failed` | `respond.ErrorHandler`, statut 5xx | ERROR | `method`, `path`, `status`, `error`, `request_id` |
| `panic recovered` | Recover | ERROR | `error`, `stack`, `request_id` |
| `SQL executed` | GORM | ERROR pour une requête en échec, sauf « aucun enregistrement » ; WARN au-delà de `DB_SLOW_QUERY_THRESHOLD` | `trace` : `duration`, `sql` avec des marqueurs à la place des valeurs, `rows`, `error` |
| Messages des services | `AuthService` | WARN | `user_id`, et `error` pour un échec |

Chaque enregistrement porte aussi `app` et `environment`. Avec
`LOG_LEVEL=info`, la valeur par défaut, les rejets de niveau DEBUG (`400`,
`404`, `405`, `409`, `413`, `415`, `422`, `499`) ne sont pas écrits. Une requête
SQL en échec est journalisée par GORM au niveau ERROR même quand le client
reçoit une erreur 4xx, comme le `409` d'un email déjà pris.

**Identifiant de requête.** `X-Request-Id` figure dans la réponse et dans
les enregistrements `request`, `request rejected`, `request failed` et
`panic recovered`. Il ne figure pas dans les enregistrements de GORM ni
dans ceux des services : le logger ne lit rien dans le contexte. Pour les
relier à une requête, utilisez l'horodatage et `user_id`.

### Qui produit quel statut

| Statut | Message | Produit par |
| --- | --- | --- |
| `400` | `Malformed request body.` | `requests.Bind` |
| `401` | `Unauthenticated.` | `Authenticated` ; `UserService.CurrentUser` et `AuthService.Refresh` |
| `401` | `Invalid credentials.` | `AuthService.Login` |
| `404` | `Resource not found.` | Le routeur d'Echo, pour une route inconnue |
| `405` | `Method Not Allowed.` | Le routeur d'Echo, avec l'en-tête `Allow` |
| `409` | `Unable to register with the provided details.` | `AuthService.Register` |
| `413` | `Request Entity Too Large.` | Body limit ; `requests.Bind` pour un corps sans `Content-Length` |
| `415` | `Unsupported Media Type.` | `requests.Bind` |
| `422` | `The given data was invalid.` | `requests.Bind` ; `AuthService.Register` pour un mot de passe de plus de 72 octets |
| `429` | `Too many requests.` | `RateLimit`, avec `Retry-After: 60` |
| `499` | `Client closed request.` | Client disconnect, quand le client a fermé la connexion ; ce statut non standard, repris de nginx, n'apparaît que dans les logs |
| `500` | `Internal server error.` | Recover ; tout service, par `apperror.Internal` ; `respond.ErrorHandler` pour une erreur qu'il ne reconnaît pas |
| `503` | `Service unavailable.` | `HealthController` ; Request timeout, quand l'échéance coupe le travail de la requête |

`apperror.Forbidden` (`403 This action is forbidden.`) et
`apperror.NotFound` existent pour les services, mais aucun service du kit
ne les renvoie aujourd'hui ; le `404` d'une route inconnue vient du
routeur.

## Contexte, échéance et annulation

`net/http` crée le contexte de chaque requête. Ce contexte est annulé
quand le client ferme la connexion, ou quand le handler a fini. Request
timeout le remplace par un contexte enfant à échéance de
`HTTP_REQUEST_TIMEOUT`. Tout ce qui s'exécute ensuite le voit :
middleware de route, contrôleur, service par `ctx`, repository par
`db.WithContext(ctx)`.

**L'échéance n'arrête pas le handler.** Elle annule seulement les
opérations qui surveillent le contexte : l'attente d'une connexion du
pool, une requête SQL, le ping de `/health`. Le calcul d'un hash bcrypt ou
la vérification d'un jeton vont jusqu'au bout. Un handler qui ignore le
contexte et répond après l'échéance envoie sa réponse normalement.

**Quand l'échéance coupe une requête SQL, la réponse est `503`.** Le
driver renvoie une erreur qui enveloppe `context.DeadlineExceeded`, et le
service l'enveloppe à son tour dans `apperror.Internal`. Au retour,
Request timeout constate que l'échéance de la requête est dépassée, trouve
`context.DeadlineExceeded` dans la chaîne et place l'erreur dans
`apperror.ServiceUnavailable()`. Ce `503` est la première
`*apperror.Error` de la chaîne : `respond.ErrorHandler` répond
`503 Service unavailable.`, et l'enregistrement `request failed`, au
niveau ERROR, garde la cause complète dans son attribut `error`.

**Déconnexion du client.** Le contexte est annulé avec
`context.Canceled`, et une requête SQL en cours s'arrête. Le service
enveloppe l'erreur dans `apperror.Internal`, mais au retour, Client
disconnect constate que le client a fermé la connexion et la place dans
`apperror.ClientClosedRequest()`. Le statut `499` n'est pas une panne du
serveur : `request rejected` est écrit au niveau DEBUG, l'enregistrement
`request` montre `499`, et la réponse ne parvient à personne.

## Deux requêtes pas à pas

Les valeurs des exemples (identifiants, jetons, dates) sont
illustratives ; la forme des réponses est celle que produit le code. Les
enregistrements de log sont montrés sans `time`, `app` ni `environment`.

### `POST /api/v1/auth/login`

```bash
curl -s http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"Ada@Example.com","password":"correct horse battery staple"}'
```

1. **Serveur et routage.** `net/http` lit la requête ; Echo trouve
   `POST /api/v1/auth/login`, dont le handler est `AuthController.Login`
   derrière `CredentialsRateLimit`.
2. **Middleware de base.** Un `X-Request-Id` est attribué, les en-têtes de
   sécurité sont posés, le `Content-Length` est comparé à
   `HTTP_BODY_LIMIT`, et le contexte reçoit son échéance.
3. **Limitation de débit.** `CredentialsRateLimit` prend un jeton dans le
   seau de l'IP client, partagé avec `/api/v1/auth/register`.
4. **Contrôleur.** `requests.Bind` remplit `requests.Login` et vérifie que
   `email` et `password` sont présents et que `email` ne dépasse pas 254
   caractères. `AuthController.Login` appelle
   `AuthService.Login(ctx, email, password)`.
5. **Service et repositories.**
   - `UserRepository.FindByEmail` normalise l'email en `ada@example.com`
     et charge l'utilisateur.
   - `PasswordHasher.Verify` compare le mot de passe au hash bcrypt.
   - Si le hash a été calculé avec un autre `BCRYPT_COST`, il est
     recalculé et enregistré.
   - `startSession` émet un jeton d'accès JWT, génère un jeton de
     rafraîchissement, et `RefreshTokenRepository.Create` enregistre son
     hash SHA-256 avec sa date d'expiration.
6. **Réponse.** `resources.NewSession` met la session en forme, et
   `respond.OK` l'écrit avec le statut `200` :

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

Le request logger écrit enfin un enregistrement `request` au niveau INFO,
avec `"status":200`.

**Avec un mauvais mot de passe**, les étapes 1 à 4 sont identiques, et le
jeton de limitation de débit est consommé de la même façon.

- `PasswordHasher.Verify` renvoie `auth.ErrPasswordMismatch`, que le
  service traduit en `apperror.New(401, "Invalid credentials.")`.
- Le contrôleur renvoie l'erreur telle quelle ; elle remonte les
  middleware jusqu'au request logger, qui la confie à
  `respond.ErrorHandler`.
- Le client reçoit `401`, sans en-tête `WWW-Authenticate`, avec :

```json
{"message":"Invalid credentials."}
```

Un email inconnu donne exactement la même réponse : le service exécute
une comparaison bcrypt factice, puis renvoie la même erreur. Dans les deux
cas, les logs contiennent :

```json
{"level":"INFO","msg":"request rejected","method":"POST","path":"/api/v1/auth/login","status":401,"error":"401 Invalid credentials.","request_id":"VzxpGgaABjHxqhvfZMYOQcQscFfmcPGT"}
{"level":"INFO","msg":"request","method":"POST","path":"/api/v1/auth/login","route":"/api/v1/auth/login","status":401,"latency":251934250,"remote_ip":"172.18.0.1","request_id":"VzxpGgaABjHxqhvfZMYOQcQscFfmcPGT"}
```

### `GET /api/v1/me`

```bash
curl -s http://localhost:8080/api/v1/me \
  -H 'Authorization: Bearer <access_token>'
```

1. **Serveur, routage et middleware de base**, comme ci-dessus. Le
   handler est `UserController.Me` derrière `Authenticated`.
2. **Authentification.** `Authenticated` extrait le jeton de l'en-tête,
   `TokenIssuer.UserID` vérifie sa signature `HS256`, son émetteur, son
   expiration et sa date d'émission, puis lit l'identifiant de
   l'utilisateur dans `sub`. Le middleware l'enregistre dans le contexte
   Echo et appelle le contrôleur.
3. **Contrôleur.** `UserController.Me` lit l'identifiant avec
   `middleware.CurrentUserID(c)` et appelle
   `UserService.CurrentUser(ctx, userID)`. Aucune entrée n'est à lier.
4. **Service et repository.** `UserRepository.FindByID` charge
   l'utilisateur. S'il a été supprimé depuis l'émission du jeton, le
   repository renvoie `ErrNotFound`, que le service traduit en
   `401 Unauthenticated.`, sans en-tête `WWW-Authenticate` : seul le
   middleware pose cet en-tête.
5. **Réponse.** `respond.OK(c, resources.NewUser(user))`, statut `200` :

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

**Sans jeton**, la requête s'arrête à l'étape 2 : le contrôleur et la base
ne sont jamais atteints.

- `Authenticated` ne trouve pas d'en-tête `Authorization`, pose
  `WWW-Authenticate: Bearer` et renvoie `apperror.Unauthorized()`.
- `respond.ErrorHandler` répond `401` :

```json
{"message":"Unauthenticated."}
```

Les logs contiennent :

```json
{"level":"INFO","msg":"request rejected","method":"GET","path":"/api/v1/me","status":401,"error":"401 Unauthenticated.","request_id":"kbvDIuJkTkfVfaGhYLBfkoIjqmDTIwkt"}
{"level":"INFO","msg":"request","method":"GET","path":"/api/v1/me","route":"/api/v1/me","status":401,"latency":7834,"remote_ip":"172.18.0.1","request_id":"kbvDIuJkTkfVfaGhYLBfkoIjqmDTIwkt"}
```

Avec un jeton refusé (expiré, forgé, mal formé), la réponse est la même,
mais l'en-tête devient `WWW-Authenticate: Bearer error="invalid_token"`, et
la cause n'apparaît que dans les logs, par exemple
`"error":"401 Unauthenticated.: auth: invalid token: token is malformed: token contains an invalid number of segments"`.

## Où ajouter quoi

Pour une nouvelle fonctionnalité, chaque morceau de code a sa couche :

| Besoin | Où | Règle |
| --- | --- | --- |
| Une table ou une colonne | `database/migrations/`, avec `make migrate-create name=...` | Une migration ne fait qu'ajouter ; chaque `up` a son `down`. |
| Le modèle | `internal/models/` | Embarque `models.Base`. |
| Les requêtes SQL | `internal/repositories/` | `ctx` en premier, `r.db.WithContext(ctx)`, erreurs passées par `translate`. |
| Les règles métier et les transactions | `internal/services/` | Ne lit jamais `echo.Context` ; traduit chaque erreur en `*apperror.Error`. |
| Les champs d'entrée et leur validation | `internal/http/requests/` | Une struct par endpoint, tags `json` et `validate`, chaque chaîne bornée par `max=`. |
| Le JSON de la réponse | `internal/http/resources/` | Une struct et son constructeur `NewXxx` par forme de réponse ; jamais de modèle sérialisé directement. |
| Le handler | `internal/http/controllers/` | `requests.Bind`, un appel de service, `respond.*` ou `return err`. |
| La route et ses middleware | `internal/routes/routes.go` et `routes.Dependencies` | Derrière `d.Authenticated` dès que la route lit l'utilisateur courant. |
| Le câblage | `internal/app/dependencies.go` | Chaque dépendance construite par son constructeur. |
| Un middleware | `internal/http/middleware/` | Ajouté à `Baseline` s'il concerne chaque requête, sinon passé aux routes par `routes.Dependencies`. |
| Une variable de configuration | `internal/config/`, `.env.example`, [configuration.md](configuration.md) | Seul `internal/config` lit l'environnement. |
| Les tests | À côté du code | Endpoints testés à travers l'application réelle avec `testutil.NewClient` ; voir [Tester un endpoint](testing.md#tester-un-endpoint). |

Les [conventions](conventions.md) détaillent ces règles. Le
[skill scaffold](../.claude/skills/scaffold/SKILL.md) déroule l'ajout d'une
ressource étape par étape, et le guide
[to-do list](guides/todo-list.md) l'applique sur un cas complet.
