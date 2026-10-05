# Agents.md

Instructions à destination des agents (humains ou IA) travaillant sur ce projet.

## Règles de code

- **Interdiction d'utiliser `panic` et `recover`** dans tout le code du projet (`cmd/`, `internal/`, `ui/`).
  - Toute erreur doit être gérée explicitement via le mécanisme standard de Go : retour d'une valeur `error`, propagée à l'appelant avec `fmt.Errorf("...: %w", err)` si besoin de contexte.
  - Un agent/développeur ne doit jamais introduire de `panic(...)`, de `recover()`, ni de fonction qui en dépend indirectement (ex: certaines libs tierces), sauf si cela est strictement imposé par une dépendance externe et impossible à contourner — dans ce cas, documenter clairement pourquoi.
  - Les erreurs attendues (ressource introuvable, validation invalide, etc.) doivent utiliser des erreurs sentinelles (`errors.New`, `errors.Is`) comme c'est déjà fait dans `internal/store` (`ErrUserNotFound`, `ErrUserExists`, `ErrProductNotFound`, etc.).
  - En cas d'erreur irrécupérable (ex: échec d'ouverture de la base au démarrage), terminer proprement le programme avec `os.Exit(1)` après avoir affiché un message clair, plutôt que de laisser un `panic` remonter.

## Pourquoi cette règle

`panic`/`recover` casse le flux normal de gestion d'erreurs de Go, rend le comportement du programme moins prévisible, et complique les tests et la lecture du code. Toutes les fonctions du projet doivent rester prévisibles et testables via leurs valeurs de retour `error`.
