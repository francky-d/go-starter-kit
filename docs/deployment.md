# Déploiement

Cette page décrit l'image Docker, comment exécuter toute la pile en
conteneurs, et ce qu'il faut vérifier avant la mise en production.

## Image Docker

Le `Dockerfile` construit l'image en deux étapes :

1. `golang:1.26-alpine` compile des binaires statiques pour l'API et la
   commande de migration.
2. `gcr.io/distroless/static-debian13:nonroot` les exécute. L'image n'a ni
   shell ni gestionnaire de paquets, et s'exécute avec un utilisateur non
   privilégié.

```bash
docker build -t go-starter-kit .
```

| Chemin dans l'image | Contenu |
| --- | --- |
| `/usr/local/bin/api` | L'API, point d'entrée de l'image. Elle écoute sur `HTTP_PORT` (`8080`). |
| `/usr/local/bin/migrate` | La commande de migration, avec les migrations embarquées. |

L'image ne contient aucune configuration : passez chaque variable de
[configuration.md](configuration.md) par l'environnement. Les fichiers
`.env` sont exclus du contexte de build et jamais copiés dans l'image.

## Toute la pile en conteneurs

`make docker-full` construit l'image et démarre PostgreSQL et l'API
ensemble :

```bash
make docker-full
curl -s http://localhost:8080/health
```

Le conteneur de l'API applique les migrations au démarrage
(`DB_AUTO_MIGRATE`) et lit `JWT_SECRET` dans `.env` ou le shell ; sans
lui, l'API s'arrête avec une erreur de configuration. `make docker-full`
crée `.env` s'il n'existe pas, avec un `JWT_SECRET` aléatoire. L'API
écoute sur `localhost:8080` ; réglez `API_FORWARD_PORT` pour utiliser un
autre port.
`make docker-logs` suit les logs des deux conteneurs, et `make docker-down`
les arrête.

L'image n'a pas de health check : `make docker-full` rend la main dès que
le processus de l'API a démarré, pas dès qu'il sert des requêtes. Vérifiez
`/health` comme ci-dessus.

Au quotidien, préférez `make dev` : l'API tourne avec rechargement à chaud
(voir [getting-started.md](getting-started.md#rechargement-à-chaud)).

## Migrations en production

Appliquez les migrations avant que la nouvelle version reçoive du trafic,
de l'une de ces deux façons :

- **Un job ponctuel** avant le déploiement, avec la même image :

  ```bash
  docker run --rm -e DATABASE_URL="$DATABASE_URL" \
    --entrypoint /usr/local/bin/migrate go-starter-kit up
  ```

- **Au démarrage**, avec `DB_AUTO_MIGRATE=true`. Un verrou consultatif
  PostgreSQL fait que des instances concurrentes appliquent chaque
  migration une seule fois, mais chaque instance attend le verrou pendant
  son démarrage.

Écrivez des migrations avec lesquelles la version précédente de l'API
fonctionne encore : pendant un déploiement, anciennes et nouvelles
instances partagent la base.

## Avant la mise en production

- Réglez `APP_ENV=production`. Chaque enregistrement de log le porte.
- Générez un `JWT_SECRET` propre à cet environnement
  (`openssl rand -hex 32`) et gardez-le dans un gestionnaire de secrets,
  jamais dans l'image ni dans le dépôt.
- Utilisez `sslmode=require` ou plus strict dans `DATABASE_URL`.
- Gardez `LOG_FORMAT=json` pour les collecteurs de logs.
- Terminez le TLS devant l'API. N'activez `HTTP_TRUST_PROXY` que si un
  proxy que vous contrôlez renseigne `X-Forwarded-For`.
- Réglez `CORS_ALLOWED_ORIGINS` sur les origines exactes de vos clients
  web.
- Utilisez `/health` comme sonde de disponibilité (readiness), pas comme
  sonde de vivacité (liveness) : voir
  [api-reference.md](api-reference.md#get-health).
- Avec plusieurs instances, les limites de débit de l'authentification
  comptent par instance ; ajoutez une limite partagée dans le load
  balancer ou la gateway si besoin.
- Gardez `BCRYPT_COST` à 12 ou plus.
