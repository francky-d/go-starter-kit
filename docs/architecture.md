# Architecture

Cette page explique comment l'application démarre, comment une requête
traverse le code, et où vit chaque responsabilité.

## Packages

| Package | Responsabilité |
| --- | --- |
| `cmd/api` | Point d'entrée : charge la configuration, construit le logger et l'application, l'exécute jusqu'à la réception d'un signal. |
| `cmd/migrate` | Applique, annule et crée les migrations. |
| `database/migrations` | Les migrations SQL, embarquées dans les binaires. |
| `internal/config` | Lit et valide les variables d'environnement dans des structs typées. Le seul package qui lit l'environnement. |
| `internal/logging` | Construit le logger `slog` à partir de la configuration des logs. |
| `internal/app` | Câble chaque dépendance avec des constructeurs explicites et exécute le serveur HTTP. |
| `internal/database` | Ouvre le pool de connexions PostgreSQL utilisé via GORM, et exécute les migrations embarquées. |
| `internal/routes` | Déclare toutes les routes en un seul endroit. |
| `internal/http/middleware` | Les middleware : la pile de base que traverse chaque requête, la politique d'IP client, l'authentification bearer et la limitation de débit. |
| `internal/http/router` | Surcouche fluide du routage d'Echo : groupes, middleware et routes déclarés en chaîne. |
| `internal/http/requests` | Les structs de requête avec leurs règles de validation, et `Bind`, qui transforme une entrée invalide en `400`, `413`, `415` ou `422`. |
| `internal/http/resources` | Mettent en forme les modèles dans le JSON que reçoivent les clients ; un modèle n'est jamais sérialisé directement, si bien qu'une nouvelle colonne ne fuit jamais par accident. |
| `internal/http/controllers` | Les handlers HTTP. Un contrôleur renvoie une réponse ou une erreur, jamais un corps d'erreur. |
| `internal/http/respond` | Les enveloppes de succès et le gestionnaire d'erreurs central. |
| `internal/models` | Les modèles GORM. Chaque modèle embarque `Base` : identifiant UUID v7 et horodatages. |
| `internal/repositories` | Les requêtes, un repository par modèle ; les erreurs de la base deviennent `ErrNotFound`, `ErrDuplicate` ou `ErrInvalid`. |
| `internal/services` | Les règles métier, comme l'inscription, la connexion et la rotation des jetons de rafraîchissement. Renvoient des erreurs applicatives et ignorent tout de HTTP. |
| `internal/auth` | Les primitives d'authentification : hash des mots de passe, jetons d'accès et jetons de rafraîchissement. Ignorent tout de HTTP et de la base. |
| `internal/apperror` | Des erreurs qui portent un statut HTTP et un message sûr pour le client. |
| `internal/testutil` | Les helpers de test : configuration de test, base de test migrée et vidée autour de chaque test, client HTTP en mémoire. |

Les dépendances pointent vers l'intérieur : `cmd/api` dépend de
`internal/app`, qui dépend de tout le reste. Aucun package de production ne
dépend de `internal/app` ; seuls les tests l'importent.

## Démarrage et arrêt

```text
main
 ├─ config.Load()          lit .env (facultatif) et l'environnement, valide
 ├─ logging.New()          logger JSON ou texte sur stdout
 ├─ app.New(ctx, cfg, log)
 │   ├─ database.Connect   ouvre le pool, échoue après DB_CONNECT_TIMEOUT si la base est injoignable
 │   ├─ migrations         appliquées ici seulement si DB_AUTO_MIGRATE=true
 │   ├─ dependencies       repositories, services, contrôleurs et middleware,
 │   │                     chacun construit par son constructeur (internal/app/dependencies.go)
 │   └─ routes             construit Echo, installe le gestionnaire d'erreurs, enregistre les routes
 ├─ app.Run(ctx)           écoute sur HTTP_PORT et sert les requêtes
 │   └─ SIGINT / SIGTERM   cesse d'accepter des connexions,
 │                         attend jusqu'à HTTP_SHUTDOWN_TIMEOUT la fin des requêtes en cours
 └─ app.Close()            ferme le pool de connexions
```

Toute erreur de démarrage (configuration invalide, base injoignable, port
déjà utilisé) est affichée sur la sortie d'erreur et le processus se
termine avec le statut 1.

Chaque enregistrement de log porte les attributs `app` et `environment`.

Le serveur applique les délais de lecture, d'écriture et d'inactivité de la
[configuration](configuration.md#serveur-http) : un client lent ou inactif
ne peut pas garder une connexion indéfiniment.

## Routes

Chaque route est déclarée dans `internal/routes/routes.go` via la surcouche
du routeur, si bien que la table de routage se lit comme une liste :

```go
api := r.Group("/api/v1")

authentication := api.Group("/auth")

authentication.Middleware(d.CredentialsRateLimit).
    Post("/register", d.Auth.Register).
    Post("/login", d.Auth.Login)

authentication.Middleware(d.RefreshRateLimit).
    Post("/refresh", d.Auth.Refresh).
    Post("/logout", d.Auth.Logout)

api.Middleware(d.Authenticated).
    Get("/me", d.Users.Me)
```

`d` contient les contrôleurs et les middleware construits dans
`internal/app`.

- `Group(prefix, middleware...)` et `Middleware(middleware...)` renvoient
  un nouveau routeur et laissent le receveur inchangé : l'ordre de
  déclaration des groupes et des middleware n'a donc jamais d'importance.
- Les middleware de groupe s'exécutent avant ceux de la route, dans l'ordre
  de déclaration.
- Un middleware ne s'exécute que sur les routes qui existent : une URL
  inconnue sous un préfixe protégé répond `404`, pas `401`.

## Cycle de vie d'une requête

Echo choisit la route, puis la requête traverse les middleware de base,
ceux de sa route, le contrôleur, le service et le repository. Une réponse
de succès est écrite par le contrôleur ; toute erreur remonte jusqu'à
`respond.ErrorHandler`, le seul endroit où s'écrivent les corps d'erreur,
qui n'envoie jamais de détail interne au client.

[request-lifecycle.md](request-lifecycle.md) suit ce trajet couche par
couche : l'ordre et l'effet des middleware, la traduction des erreurs à
chaque couche, l'échéance des requêtes, ce qui est journalisé, et deux
requêtes pas à pas.
