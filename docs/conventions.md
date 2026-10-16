# Conventions

Le code de ce kit privilégie des noms explicites et des appels chaînés là
où Go le permet, tout en restant du Go idiomatique. Cette page résume les
règles. La version complète, qui fait référence, est le
[skill conventions](../.claude/skills/conventions/SKILL.md) : les agents IA
le chargent avant d'écrire du code, et les humains peuvent le lire comme un
guide de style.

## Ordre de lecture d'une requête

```text
router -> middleware -> controller -> service -> repository -> database
```

| Couche | Responsabilité |
| --- | --- |
| Contrôleur | Lie l'entrée, appelle une méthode de service, renvoie une réponse. |
| Request | Déclare les champs d'entrée et leurs règles de validation. |
| Resource | Met en forme un modèle dans le JSON que voit le client. |
| Service | Porte les règles métier et les transactions ; renvoie des erreurs applicatives. |
| Repository | Exécute les requêtes ; renvoie des modèles ou des erreurs sentinelles. |
| Modèle | Représente une table ; de petits helpers sur un seul enregistrement. |

## Règles principales

- **Une fonction, une tâche.** Une fonction fait exactement ce que dit son
  nom. Les étapes sont extraites dans des fonctions privées nommées ; visez
  20 à 25 lignes.
- **De petites interfaces, déclarées par leur consommateur**, et seulement
  quand un test ou une seconde implémentation en a besoin.
- **Des commentaires seulement quand c'est nécessaire.** Le code s'explique
  lui-même ; un commentaire explique un concept, une contrainte ou un choix
  délibéré, jamais ce qu'un nom dit déjà.
- **Les noms disent ce que font les choses.**
  `users.FindByEmail(ctx, email)`,
  `respond.Created(c, resources.NewUser(user))`.
- **Des dépendances explicites.** Chaque type est construit par un
  constructeur `NewXxx` qui reçoit ses dépendances. Il n'y a pas de
  conteneur global.
- **Des contrôleurs minces.** Ils ne construisent jamais de JSON d'erreur
  et n'interrogent jamais la base.
- **Les erreurs ne divulguent rien.** Les clients reçoivent des messages
  génériques pour les 401, 403, 404 et 500. Les causes internes sont
  journalisées, jamais renvoyées. La seule divulgation acceptée est un 409
  à l'inscription d'un email déjà pris.
- **Les modèles ne sont jamais sérialisés directement.** Une resource
  décide quels champs quittent le serveur.
- **La configuration est typée.** Seul `internal/config` lit les variables
  d'environnement.
- **Le contexte partout.** Chaque fonction qui fait des I/O prend `ctx` en
  premier. Pendant une requête, journalisez avec les méthodes `...Context`
  du logger (`WarnContext(ctx, ...)`) : il y lit l'identifiant de requête.
- **Les migrations ne font qu'ajouter.** Ne modifiez jamais une migration
  appliquée ailleurs que sur votre machine ; écrivez-en une nouvelle.
- **Chaque changement est testé**, contre une vraie base PostgreSQL quand
  la base est concernée.
- **Deux langues.** Le code, les commentaires et les messages de commit
  sont en anglais ; la documentation (`docs/` et `README.md`) est en
  français, y compris les commentaires du code des guides.
