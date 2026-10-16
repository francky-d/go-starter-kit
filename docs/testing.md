# Tests

Chaque changement est couvert par un test. Les tests utilisent le package
standard `testing` avec [testify](https://github.com/stretchr/testify), et
s'exécutent contre une vraie base PostgreSQL : la base n'est jamais
simulée.

## Lancer les tests

```bash
make docker-up     # une fois : démarre PostgreSQL
make test          # toute la suite
make test-cover    # toute la suite, avec couverture
go test ./internal/repositories/...   # un seul package pendant le développement
```

`make test` lance `go test -p 1 ./...`. Tous les packages partagent la même
base de test : les packages s'exécutent donc un par un ; les tests d'un
même package s'exécutent toujours dans l'ordre.

Les tests se connectent à `app_test` sur `localhost:5432`. Réglez
`TEST_DATABASE_URL` pour utiliser une autre base (voir
[database.md](database.md#base-de-test)).

## Couverture

`make test-cover` lance la suite avec la couverture de chaque package sous
`internal/`, y compris le code atteint depuis les tests d'un autre
package : les tests de bout en bout comptent pour les contrôleurs et les
services qu'ils traversent. Il affiche le total et écrit deux fichiers,
tous deux ignorés par git :

| Fichier | Contenu |
| --- | --- |
| `coverage.out` | Le profil brut, pour `go tool cover`. |
| `coverage.html` | Une page qui colore chaque ligne selon sa couverture. |

Lisez la ligne `total:` et la page HTML. Pendant l'exécution, un package
sans fichier de test affiche `coverage: 0.0% of statements` même quand les
tests d'autres packages le couvrent ; le profil compte bien cette
couverture.

## Tests de bout en bout

`internal/app/auth_flow_test.go` pilote l'API comme le ferait une
application cliente : inscription, lecture de l'utilisateur courant,
rafraîchissement, déconnexion, nouvelle connexion, puis présentation d'un
jeton de rafraîchissement déjà échangé pour vérifier que toutes les
sessions sont révoquées. Il couvre la façon dont les endpoints
fonctionnent ensemble, ce que ne font pas les tests de chaque endpoint.

## Intégration continue

Le kit fournit la même CI pour deux hébergeurs : GitHub Actions
(`.github/workflows/ci.yml`) et GitLab CI (`.gitlab-ci.yml`). Chacune a
deux jobs.

**Lint and test** (`lint-and-test` sur GitLab) lance les mêmes commandes
qu'une exécution locale, avec la dernière version corrective de Go 1.26 :

1. `go build ./...`
2. `make lint`
3. `go test -race -p 1 ./...`, contre un conteneur de service
   PostgreSQL 17 joint via `TEST_DATABASE_URL`. Le détecteur de courses
   vérifie aussi les tests concurrents, comme deux rafraîchissements du
   même jeton.

**Docker image** (`docker-image` sur GitLab) construit le `Dockerfile` :
un changement qui casse l'image échoue comme un test.

| | GitHub Actions | GitLab CI |
| --- | --- | --- |
| Déclencheurs | Chaque push sur `main` et chaque pull request. | Chaque merge request et chaque push sur la branche par défaut. |
| Annulation | Un nouveau push sur une pull request annule l'exécution en cours. | Un nouveau push sur une merge request annule le pipeline en cours. |
| Branche principale | Chaque exécution va à son terme. | Chaque pipeline va à son terme. |
| Cache Go | Géré par `actions/setup-go`. | Modules et compilation dans `.go/`, clé calculée sur `go.sum`. |
| Image Docker | Construite par le runner. | Construite par Docker-in-Docker (`docker:29-dind`) : le runner doit autoriser les conteneurs privilégiés et partager `/certs/client` entre le service et le job, comme les runners partagés de GitLab.com. |
| Version minimale | — | GitLab 16.10, pour `workflow:auto_cancel`. |

GitLab ne met en cache que des chemins situés dans le projet, d'où le
dossier `.go/`. `make lint` et `make fmt` ignorent les dossiers cachés,
comme les commandes `go` : sans cela, `gofmt` analyserait aussi les
modules téléchargés.

Un commit dont le message contient `[skip ci]` n'est vérifié ni par
GitHub ni par GitLab : le skill de commit ne l'ajoute que quand tous les
fichiers modifiés sont en Markdown. Si le dépôt exige une CI verte avant
de fusionner (règle de protection de branche sur GitHub, « Pipelines must
succeed » sur GitLab), une pull ou merge request dont le dernier commit a
sauté la CI ne peut pas être fusionnée tant qu'un nouveau commit ne l'a
pas lancée.

## Helpers

Le package `internal/testutil` fournit :

| Helper | Fournit |
| --- | --- |
| `testutil.DB(t)` | Une connexion à la base de test migrée. Chaque table est vidée avant le test et à sa fin. |
| `testutil.Truncate(t, db)` | Vide chaque table sauf l'historique des migrations. |
| `testutil.Config()` | Une configuration complète et valide pour les tests, pointant vers la base de test. |
| `testutil.DatabaseConfig()` | La partie base de données de cette configuration. |
| `testutil.DatabaseURL()` | `TEST_DATABASE_URL`, ou la base de test locale. |
| `testutil.NewClient(t, handler)` | Un client qui envoie des requêtes JSON à un handler en mémoire : `Get`, `Post`, et `WithToken` pour un jeton bearer. |

Une réponse du client est un `httptest.ResponseRecorder` avec deux
décodeurs : `Data(&target)` lit le membre `data` d'une réponse de succès,
et `ErrorBody()` renvoie le `message` et les `errors` d'une réponse
d'erreur.

Les migrations sont appliquées une fois par binaire de test, au premier
appel de `testutil.DB`.

## Écrire un test

```go
func TestUserRepository_ReportsATakenEmailAsDuplicate(t *testing.T) {
    users := repositories.NewUserRepository(testutil.DB(t))
    require.NoError(t, users.Create(t.Context(), newUser("ada@example.com")))

    err := users.Create(t.Context(), newUser("ada@example.com"))

    assert.ErrorIs(t, err, repositories.ErrDuplicate)
}
```

### Tester un endpoint

Les tests d'endpoint passent par la vraie application, avec chaque
middleware, sur la base de test :

```go
func newClient(t *testing.T) *testutil.Client {
    t.Helper()
    testutil.DB(t) // empties the tables around the test
    application, err := app.New(t.Context(), testutil.Config(), logging.Discard())
    require.NoError(t, err)
    t.Cleanup(func() { assert.NoError(t, application.Close()) })

    return testutil.NewClient(t, application.Handler())
}

func TestLogin_AnswersTheSameForAnUnknownEmailAndAWrongPassword(t *testing.T) {
    client := newClient(t)
    register(t, client, "ada@example.com")

    unknownEmail := client.Post("/api/v1/auth/login", map[string]string{"email": "grace@example.com", "password": password})
    wrongPassword := client.Post("/api/v1/auth/login", map[string]string{"email": "ada@example.com", "password": "wrong"})

    assert.Equal(t, http.StatusUnauthorized, unknownEmail.Code)
    assert.Equal(t, unknownEmail.Body.String(), wrongPassword.Body.String())
}
```

La configuration de test fixe des limites de débit très hautes pour
l'authentification : seul un test qui en abaisse une l'atteint.

### Règles

- Nommez le test d'après ce qu'il prouve : `Test<Subject>_<Behaviour>`.
- Utilisez `require` pour les préconditions et `assert` pour les
  vérifications.
- Placez le fichier de test à côté du code, dans le package `_test`, pour
  que le test n'utilise que l'API exportée.
- Utilisez `t.Context()` pour les contextes ; il est annulé à la fin du
  test.
