# Authentification

Cette page décrit comment le kit protège les identifiants des utilisateurs,
comment il délivre et vérifie les jetons d'accès et les jetons de
rafraîchissement, et comment les sessions démarrent, se renouvellent et
se terminent.

## Mots de passe

Les mots de passe sont hachés avec bcrypt avant d'être stockés ; le mot de
passe en clair n'est jamais stocké ni journalisé.

- **Facteur de coût :** `BCRYPT_COST` (`12` par défaut). Chaque cran double
  le temps de hachage et de vérification, ce qui ralentit un attaquant qui
  obtient les hash. Voir
  [configuration.md](configuration.md#authentification). Après un
  changement, chaque hash existant est remplacé à la connexion réussie
  suivante de son utilisateur.
- **Sel :** chaque hash a son propre sel aléatoire, si bien que deux
  utilisateurs avec le même mot de passe obtiennent des hash différents.
- **Longueur :** bcrypt ne lit que les 72 premiers octets d'un mot de
  passe. Les mots de passe plus longs sont rejetés plutôt que tronqués en
  silence : l'inscription les refuse, et la connexion les traite comme un
  mauvais mot de passe. Les caractères hors ASCII prennent plusieurs
  octets : 72 octets peuvent donc faire moins de 72 caractères.
- **Comptes inconnus :** quand une connexion nomme un email qui n'existe
  pas, le hacheur exécute quand même une comparaison bcrypt complète contre
  un hash factice : le temps de réponse ne révèle pas quels emails ont un
  compte. Tant que tous les hash n'ont pas été mis à niveau après un
  changement de `BCRYPT_COST`, les anciens hash se vérifient à leur ancien
  coût, ce qu'une mesure de temps précise pourrait distinguer.

## Jetons d'accès

Un jeton d'accès est un JSON Web Token signé en HMAC-SHA256 (`HS256`) avec
la clé `JWT_SECRET`. Il porte :

| Claim | Contenu |
| --- | --- |
| `sub` | L'identifiant de l'utilisateur. |
| `iss` | `JWT_ISSUER`. |
| `iat` | La date d'émission du jeton. |
| `exp` | La date d'expiration du jeton : `ACCESS_TOKEN_LIFETIME` après `iat`. |
| `jti` | Un identifiant unique du jeton. |

Un jeton n'est accepté que si toutes ces conditions sont réunies :

- il est signé en `HS256` avec le secret configuré. Les jetons signés avec
  un autre algorithme, `none` compris, sont refusés avant même la
  vérification de leur signature ;
- son émetteur est `JWT_ISSUER` ;
- il a une date d'expiration, et elle n'est pas dépassée ;
- il n'a pas été émis dans le futur ;
- son sujet est un identifiant d'utilisateur.

Les horloges peuvent différer de 30 secondes au plus entre le serveur qui
a émis un jeton et celui qui le vérifie.

Tout jeton refusé produit le même résultat, quelle que soit la raison : un
client n'apprend rien sur le motif du refus.

Un jeton d'accès ne peut pas être révoqué : il reste valide jusqu'à son
expiration. Gardez `ACCESS_TOKEN_LIFETIME` courte. Changer `JWT_SECRET`
invalide d'un coup tous les jetons d'accès.

`HS256` utilise le même secret pour signer et pour vérifier, ce qui
convient à une API qui délivre et vérifie elle-même ses jetons. Dès qu'un
autre service doit les vérifier, partager le secret lui permettrait aussi
d'en forger : passez à un algorithme asymétrique comme `EdDSA` ou `RS256`,
gardez la clé privée dans cette API et ne publiez que la clé publique.

## Authentifier les requêtes

Les routes protégées attendent le jeton d'accès dans l'en-tête
`Authorization` :

```text
Authorization: Bearer <access_token>
```

Le schéma est insensible à la casse. Une requête est refusée avec `401
Unauthenticated.` quand l'en-tête manque, utilise un autre schéma, ou porte
un jeton qui échoue à l'une des vérifications des
[jetons d'accès](#jetons-daccès). Un jeton de rafraîchissement n'est pas un
jeton d'accès et il est refusé lui aussi.

L'en-tête `WWW-Authenticate` du refus indique au client quoi faire :

| En-tête | Signification | Ce que doit faire le client |
| --- | --- | --- |
| `Bearer` | Aucun jeton bearer n'a été envoyé. | Se connecter. |
| `Bearer error="invalid_token"` | Le jeton envoyé a été refusé. Un jeton expiré et un jeton forgé reçoivent la même réponse. | Rafraîchir la session, puis se reconnecter si cela échoue. |

Un jeton d'accès reste valide après la suppression de son utilisateur,
jusqu'à son expiration. Les endpoints qui chargent l'utilisateur courant
répondent alors `401`.

Dans un contrôleur, `middleware.CurrentUserID(c)` renvoie l'identifiant de
l'utilisateur authentifié. Il n'est disponible que sur les routes placées
derrière le middleware `Authenticated` ; l'appeler ailleurs est une erreur
de programmation et provoque un panic.

## Jetons de rafraîchissement

Un jeton de rafraîchissement permet à un client d'obtenir un nouveau jeton
d'accès sans renvoyer le mot de passe. C'est une chaîne opaque, pas un
JWT : 32 octets aléatoires encodés en 43 caractères base64 compatibles URL.

- **Stockage :** seul le hash SHA-256 du jeton est stocké, dans la table
  `refresh_tokens`. Une copie de la table ne donne aucun jeton utilisable.
  Un simple hash suffit ici, contrairement aux mots de passe : 256 bits
  aléatoires ne se devinent pas à partir de leur hash.
- **Durée de vie :** `REFRESH_TOKEN_LIFETIME` (30 jours par défaut).
- **Révocation :** un jeton de rafraîchissement peut être révoqué à tout
  moment. Les jetons révoqués sont conservés, pour reconnaître un jeton
  volé quand il est présenté à nouveau (voir
  [Rafraîchir une session](#rafraîchir-une-session)).
- **Rotation du secret :** les jetons de rafraîchissement ne dépendent pas
  de `JWT_SECRET` ; changer le secret ne les révoque donc pas.

## Sessions

Une session est une paire de jetons : un jeton d'accès de courte durée et
le jeton de rafraîchissement qui le renouvelle. Chaque utilisateur peut
avoir plusieurs sessions à la fois, une par appareil.

### Inscription

L'inscription crée l'utilisateur et démarre une session. L'email est
stocké sans espaces autour et en minuscules : `Ada@Example.com` et
`ada@example.com` désignent le même compte.

Un email qui a déjà un compte est refusé avec
`Unable to register with the provided details.`. Ce message révèle bien
que l'email est pris : le masquer exigerait un parcours de vérification de
l'email, que le kit n'a pas. La limitation de débit des routes
d'authentification ralentit un attaquant qui teste des emails de cette
façon.

### Connexion

La connexion démarre une nouvelle session. Un email inconnu, un mauvais
mot de passe et un mot de passe de plus de 72 octets reçoivent tous la même
erreur `Invalid credentials.`, après le même travail bcrypt.

### Rafraîchir une session

Échanger un jeton de rafraîchissement renvoie une nouvelle session et
révoque l'ancien jeton, qui enregistre le nouveau comme son remplaçant.
Chaque jeton de rafraîchissement ne sert donc qu'une fois.

Un jeton présenté alors qu'il a déjà été échangé signifie que deux parties
le détiennent : le client et quelqu'un qui l'a volé. L'API ne peut pas
savoir laquelle est légitime : elle révoque **tous** les jetons de
rafraîchissement de l'utilisateur, et chaque appareil doit se reconnecter.
L'événement est journalisé en avertissement avec l'identifiant de
l'utilisateur.

La même chose se produit quand un client envoie deux fois le même jeton :
deux onglets qui rafraîchissent en même temps, ou un rafraîchissement dont
la réponse s'est perdue sur le réseau et que le client rejoue avec l'ancien
jeton. Un seul échange peut réussir ; l'autre compte comme une
réutilisation. Les clients devraient sérialiser leurs appels de
rafraîchissement, et se reconnecter quand un rafraîchissement reste sans
réponse, plutôt que de réessayer avec le même jeton : la nouvelle tentative
déconnecterait aussi tous les autres appareils.

Un jeton expiré, un jeton révoqué par une déconnexion et un jeton inconnu
sont refusés sans rien révoquer d'autre.

### Déconnexion

La déconnexion révoque le jeton de rafraîchissement qu'elle reçoit ; les
autres sessions de l'utilisateur restent ouvertes. Détenir le jeton de
rafraîchissement suffit : aucun jeton d'accès n'est demandé, si bien qu'un
client peut se déconnecter après l'expiration de son jeton d'accès. Un
jeton inconnu ou déjà révoqué est accepté sans erreur : se déconnecter deux
fois n'est pas une erreur.

Le jeton d'accès de la session reste valide jusqu'à son expiration, au plus
`ACCESS_TOKEN_LIFETIME`. Les clients doivent supprimer les deux jetons à la
déconnexion.

## Limitation de débit

Les routes d'authentification sont limitées par IP client, avec deux
budgets distincts :

| Routes | Variable | Défaut | Pourquoi |
| --- | --- | --- | --- |
| register, login | `AUTH_CREDENTIALS_RATE_LIMIT_PER_MINUTE` | `10` | Elles reçoivent des identifiants : une limite serrée ralentit la recherche de mots de passe et l'énumération des emails. |
| refresh, logout | `AUTH_REFRESH_RATE_LIMIT_PER_MINUTE` | `60` | Les clients les appellent au fil normal d'une session ; un jeton de rafraîchissement ne se devine pas, la limite ne protège donc que la base. |

Chaque limite autorise des rafales de sa propre taille. Au-delà, une route
répond `429` avec `Retry-After: 60`. Avec des budgets séparés, les
rafraîchissements de routine n'empêchent jamais une connexion, et un
client qui a atteint la limite de connexion peut encore se déconnecter.

- Les compteurs vivent dans la mémoire de chaque instance. Avec plusieurs
  instances, un client dispose de la limite une fois par instance ; placez
  une limite partagée dans le load balancer ou l'API gateway si cela
  compte.
- L'IP client vient de la connexion, ou de `X-Forwarded-For` quand
  `HTTP_TRUST_PROXY` est activée (voir
  [configuration.md](configuration.md#serveur-http)). Derrière un proxy
  sans ce réglage, tous les clients partagent l'IP du proxy et la même
  limite.
- Les utilisateurs derrière un même NAT, comme le réseau d'un bureau,
  partagent les deux limites. Relevez d'abord la limite de
  rafraîchissement si des utilisateurs légitimes l'atteignent ; relever la
  limite des identifiants aide aussi un attaquant.
