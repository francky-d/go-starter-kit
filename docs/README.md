# Documentation

Ce dossier décrit l'état actuel du kit tel qu'il est implémenté dans ce
dépôt. Quand le code et cette documentation divergent, le code fait foi et
la documentation doit être corrigée dans le même changement.

La documentation est écrite en français ; une traduction anglaise suivra.
Les identifiants, les commandes, les messages de l'API et les exemples de
code restent tels qu'ils sont dans le code.

## Documents

| Document | Objet |
| --- | --- |
| [getting-started.md](getting-started.md) | Prérequis, premiers pas et commandes courantes. |
| [architecture.md](architecture.md) | Packages, démarrage et arrêt, cycle de vie d'une requête. |
| [api-reference.md](api-reference.md) | Enveloppes de réponse, format des erreurs, codes de statut et endpoints. |
| [configuration.md](configuration.md) | Chaque variable d'environnement, sa valeur par défaut et la résolution des valeurs. |
| [authentication.md](authentication.md) | Mots de passe, jetons d'accès et de rafraîchissement, sessions, requêtes authentifiées et limitation de débit. |
| [database.md](database.md) | PostgreSQL en local, migrations, schéma, pool de connexions et logs des requêtes. |
| [switching-database.md](switching-database.md) | Ce qui dépend de PostgreSQL et ce qu'il faut changer pour passer à une autre base. |
| [testing.md](testing.md) | Lancer et écrire les tests, couverture, tests de bout en bout et CI. |
| [conventions.md](conventions.md) | Organisation, nommage et tests du code. |
| [deployment.md](deployment.md) | L'image Docker, la pile en conteneurs et une check-list de mise en production. |

## Guides

Les guides construisent une fonctionnalité sur le kit. Leur code ne fait
pas partie du kit, mais il a été vérifié : lint et tests passent.

| Guide | Objet |
| --- | --- |
| [guides/todo-list.md](guides/todo-list.md) | Une to-do list partagée avec des rôles : migrations, modèles, contrôle d'accès, endpoints et tests. |
