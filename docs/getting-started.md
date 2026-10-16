# Prise en main

## Prérequis

- Go 1.26 ou plus récent.
- Docker avec Compose v2, pour la base PostgreSQL locale.
- `make`.

## Premiers pas

```bash
git clone https://github.com/francky-d/go-starter-kit.git
cd go-starter-kit
cp .env.example .env
make docker-up
make test
```

Avant de lancer l'API, renseignez `JWT_SECRET` dans `.env` avec une valeur
aléatoire d'au moins 32 octets. Cette commande affiche 64 caractères
hexadécimaux, qui encodent 32 octets aléatoires :

```bash
openssl rand -hex 32
```

`make docker-up` démarre PostgreSQL 17 et attend qu'il accepte les
connexions. Voir [database.md](database.md) pour le détail.

Créez le schéma avant de lancer l'API :

```bash
make migrate-up
```

## Lancer l'API

```bash
make run
```

Le serveur écoute sur `HTTP_PORT` (`8080` par défaut). Vérifiez qu'il
répond :

```bash
curl -s http://localhost:8080/health
# {"data":{"status":"ok","database":"ok"}}
```

L'API refuse de démarrer si la base est injoignable avant
`DB_CONNECT_TIMEOUT`.

Les logs sont écrits sur la sortie standard. Arrêtez le serveur avec
`Ctrl+C` : il termine les requêtes en cours avant de s'arrêter.

## Rechargement à chaud

```bash
make dev
```

`make dev` lance l'API avec [Air](https://github.com/air-verse/air) : il
recompile et redémarre le serveur dès qu'un fichier `.go` ou `.sql`, ou le
fichier `.env`, change. Les fichiers de test, `docs/` et `tmp/` ne sont pas
surveillés. Avant chaque redémarrage, le serveur reçoit une interruption :
les requêtes en cours se terminent comme lors d'un vrai arrêt.

Tant que le code ne compile pas, le serveur reste arrêté ; la compilation
suivante qui réussit le relance.

- Une nouvelle migration est embarquée à la recompilation suivante, mais
  appliquée seulement par `make migrate-up`, ou au démarrage avec
  `DB_AUTO_MIGRATE=true`.
- Le binaire et les erreurs de compilation vont dans `tmp/`, que git
  ignore ; Air supprime ce dossier quand il s'arrête.
- Air est épinglé dans le `Makefile` et s'exécute via `go run` : rien à
  installer. Le premier lancement le télécharge.

## Commandes

`make` liste toutes les commandes.

| Commande | Effet |
| --- | --- |
| `make run` | Lance l'API avec `go run`. |
| `make dev` | Lance l'API avec rechargement à chaud : recompile et redémarre à chaque changement Go ou SQL. |
| `make build` | Compile l'API et la commande de migration dans `bin/`. |
| `make docker-up` | Démarre PostgreSQL et attend qu'il soit prêt. |
| `make docker-full` | Construit l'image de l'API et la démarre avec PostgreSQL. Nécessite `JWT_SECRET`. |
| `make docker-down` | Arrête les conteneurs en conservant les données de la base. |
| `make docker-reset` | Arrête les conteneurs et supprime les données de la base. |
| `make docker-logs` | Suit les logs des conteneurs en cours d'exécution. |
| `make migrate-up` | Applique toutes les migrations en attente. |
| `make migrate-down` | Annule la dernière migration. |
| `make migrate-version` | Affiche la version actuelle des migrations. |
| `make migrate-create name=...` | Crée une paire de migrations up/down vide. |
| `make test` | Lance la suite de tests. PostgreSQL doit tourner. |
| `make test-cover` | Lance la suite de tests avec couverture et affiche le total. |
| `make lint` | Vérifie le formatage et `go.mod`, puis lance `go vet` et `staticcheck`. Ignore les dossiers cachés. |
| `make fmt` | Formate chaque fichier Go avec `gofmt`, hors dossiers cachés. |
| `make tidy` | Synchronise `go.mod` et `go.sum` avec les imports. |

`staticcheck` est épinglé dans `go.mod` avec la directive `tool` et
s'exécute via `go tool staticcheck` : rien à installer.
