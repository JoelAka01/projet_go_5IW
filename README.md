# ecommerce-cli

Application e-commerce en ligne de commande (Go), 100% CLI :

- un **client CLI** pour les utilisateurs (`cmd/client`)
- un **client CLI admin** pour la gestion (`cmd/admin`)
- une commande utilitaire de **seed** (données de démo / promotion admin) (`cmd/seed`)

Il n'y a pas de serveur HTTP ni d'API réseau : chaque CLI accède directement à la base de données locale via le package `internal/store`.

La persistance se fait via **SQLite** (`modernc.org/sqlite`), aucune base externe n'est requise.

## Prérequis

- Go 1.27+ installé localement, **ou**
- Docker (le projet fournit un `Dockerfile` et un `compose.yml` basés sur `golang:1.27-alpine`)

## Lancer le projet en local (sans Docker)

1. Installer les dépendances :

   ```bash
   go mod tidy
   ```

2. (Optionnel) Initialiser/alimenter la base avec des données de démo :

   ```bash
   go run ./cmd/seed
   ```

   Pour rendre un utilisateur existant administrateur :

   ```bash
   go run ./cmd/seed make-admin <email>
   ```

3. Lancer le client CLI (utilisateur) :

   ```bash
   go run ./cmd/client
   ```

   Par défaut, la CLI utilise le fichier `ecommerce.db` (SQLite) à la racine du projet.

4. Pour l'interface admin :
   ```bash
   go run ./cmd/admin
   ```

### Variables d'environnement

| Variable  | Utilisé par               | Défaut         | Description                      |
| --------- | ------------------------- | -------------- | -------------------------------- |
| `DB_PATH` | `client`, `admin`, `seed` | `ecommerce.db` | Chemin du fichier de base SQLite |

Exemple (pour que client et admin partagent la même base) :

```bash
DB_PATH=mabase.db go run ./cmd/client
DB_PATH=mabase.db go run ./cmd/admin
```

## Lancer le projet avec Docker

Le `compose.yml` fourni monte le code source dans un conteneur Go et ouvre un shell interactif :

```bash
docker compose run --rm go sh
```

Une fois dans le conteneur, vous pouvez utiliser les mêmes commandes que ci-dessus (`go run ./cmd/client`, etc.).

## Tests

Pour exécuter les tests du projet :

```bash
go test ./...
```

## Structure du projet

Voir [`structure.md`](./structure.md) pour le détail de l'arborescence et le rôle de chaque dossier.
