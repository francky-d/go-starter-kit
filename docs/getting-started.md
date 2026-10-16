# Prise en main

## Prérequis

- Docker avec Compose v2 (testé avec Compose 2.40).
- `make`.

Go n'a pas besoin d'être installé : l'API, les tests et les autres
commandes Go s'exécutent dans des conteneurs.

## Premiers pas

```bash
git clone https://github.com/francky-d/go-starter-kit.git
cd go-starter-kit
make dev
```

`make dev` lance l'environnement de développement :

1. crée `.env` à partir de `.env.example` s'il n'existe pas encore, avec
   un `JWT_SECRET` aléatoire de 32 octets ;
2. démarre PostgreSQL 17 et attend qu'il accepte les connexions ;
3. démarre le conteneur `api-dev`, qui applique les migrations en attente,
   puis lance l'API avec rechargement à chaud.

Le premier lancement télécharge l'image Go, Air et les modules ; les
suivants démarrent en quelques secondes. Dans un autre terminal, vérifiez
que l'API répond, puis lancez les tests :

```bash
curl -s http://localhost:8080/health
# {"data":{"status":"ok","database":"ok"}}

make test
```

L'API refuse de démarrer si la base est injoignable avant
`DB_CONNECT_TIMEOUT`.

`Ctrl+C` dans le terminal de `make dev` arrête l'API ; le serveur termine
les requêtes en cours avant de s'arrêter. PostgreSQL continue de tourner
pour les tests. `make docker-down` arrête tous les conteneurs.

## Rechargement à chaud

[Air](https://github.com/air-verse/air) tourne dans le conteneur `api-dev`,
sur les sources montées depuis votre machine : il recompile et redémarre
le serveur dès qu'un fichier `.go` ou `.sql`, ou le fichier `.env`,
change. Les fichiers de test, `docs/` et `tmp/` ne sont pas surveillés.
Avant chaque redémarrage, le serveur reçoit une interruption : les
requêtes en cours se terminent comme lors d'un vrai arrêt.

Tant que le code ne compile pas, le serveur reste arrêté ; la compilation
suivante qui réussit le relance.

- Les migrations en attente sont appliquées au démarrage du conteneur,
  jamais lors d'un rechargement : une migration en cours d'écriture n'est
  pas appliquée avant d'être finie. Appliquez-la avec `make migrate-up`.
- Le binaire et les erreurs de compilation vont dans `tmp/`, que git
  ignore ; Air supprime ce dossier quand il s'arrête.
- Air est épinglé dans `docker-compose.yml` et installé dans le conteneur
  à son démarrage.

## Configuration dans les conteneurs

L'API du conteneur `api-dev` lit `.env`, monté avec les sources, comme
elle le ferait sur l'hôte. `docker-compose.yml` fixe trois variables, qui
l'emportent sur `.env` :

| Variable | Valeur dans le conteneur | Raison |
| --- | --- | --- |
| `DATABASE_URL` | La base `app` du service `postgres`. | `localhost` désigne le conteneur lui-même. |
| `HTTP_PORT` | `8080` | Le port publié sur l'hôte pointe vers lui. |
| `DB_AUTO_MIGRATE` | `false` | Les migrations s'appliquent au démarrage du conteneur, pas à chaque rechargement. |

Sur l'hôte, l'API répond sur `localhost:8080` et PostgreSQL sur
`localhost:5432`, joignables depuis cette machine seulement. Réglez
`API_FORWARD_PORT` ou `DB_FORWARD_PORT` (dans `.env` ou le shell) pour
utiliser d'autres ports.

## Commandes Go dans un conteneur

Les cibles du Makefile qui utilisent Go (`test`, `lint`, `build`,
`migrate-*`, ...) s'exécutent dans le conteneur `go` de
`docker-compose.yml` : `make test` relance `make test` dans ce conteneur
et démarre PostgreSQL si besoin. Ce conteneur :

- tourne avec votre utilisateur (`HOST_UID` et `HOST_GID`, fixés par le
  Makefile) : les fichiers qu'il crée, comme une nouvelle migration, vous
  appartiennent ;
- garde le cache des modules et de la compilation dans le volume `go`, que
  `make docker-reset` supprime.

Pour une commande qui n'a pas de cible, ouvrez un shell dans ce
conteneur ; `make` y fonctionne aussi :

```bash
make shell
go test ./internal/repositories/...
```

Passez par `make` plutôt que par `docker compose run` directement : sans
le Makefile, `HOST_UID` et `HOST_GID` valent `1000`, et le conteneur risque
de ne plus pouvoir écrire dans le cache créé avec votre utilisateur.

Avec Docker rootless ou Podman, l'utilisateur root du conteneur correspond
déjà au vôtre : ajoutez `HOST_UID=0 HOST_GID=0` aux commandes `make`.

Si Go 1.26 est installé sur votre machine, `GO_IN_DOCKER=false` exécute
les cibles Go directement, comme le fait la CI. Elles utilisent alors
`.env` tel quel, avec PostgreSQL sur `localhost:5432` (`make docker-up`) :

```bash
GO_IN_DOCKER=false make test
```

## Commandes

`make` liste toutes les commandes.

| Commande | Effet |
| --- | --- |
| `make dev` | Démarre PostgreSQL et l'API avec rechargement à chaud, dans des conteneurs. Crée `.env` au premier lancement. |
| `make shell` | Ouvre un shell dans le conteneur `go`, pour lancer n'importe quelle commande Go. |
| `make run` | Applique les migrations en attente, puis lance l'API sans rechargement à chaud. |
| `make build` | Compile l'API et la commande de migration dans `bin/`. |
| `make docker-up` | Démarre PostgreSQL et attend qu'il soit prêt. |
| `make docker-full` | Construit l'image de production de l'API et la démarre avec PostgreSQL. Crée `.env` au besoin. |
| `make docker-down` | Arrête les conteneurs en conservant les données de la base. |
| `make docker-reset` | Arrête les conteneurs et supprime les données de la base et les caches Go. |
| `make docker-logs` | Suit les logs des conteneurs en cours d'exécution. |
| `make migrate-up` | Applique toutes les migrations en attente. |
| `make migrate-down` | Annule la dernière migration. |
| `make migrate-version` | Affiche la version actuelle des migrations. |
| `make migrate-create name=...` | Crée une paire de migrations up/down vide. |
| `make test` | Lance la suite de tests. Démarre PostgreSQL si besoin. |
| `make test-cover` | Lance la suite de tests avec couverture et affiche le total. |
| `make lint` | Vérifie le formatage et `go.mod`, puis lance `go vet` et `staticcheck`. Ignore les dossiers cachés. |
| `make fmt` | Formate chaque fichier Go avec `gofmt`, hors dossiers cachés. |
| `make tidy` | Synchronise `go.mod` et `go.sum` avec les imports. |

Les cibles Go s'exécutent dans un conteneur (voir
[ci-dessus](#commandes-go-dans-un-conteneur)). `staticcheck` est épinglé
dans `go.mod` avec la directive `tool` et s'exécute via
`go tool staticcheck` : rien à installer.
