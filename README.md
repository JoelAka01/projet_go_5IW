# ecommerce-cli

Application e-commerce en ligne de commande (Go) organisée en architecture
client/serveur :

- un **serveur HTTP** (`cmd/server`) qui expose l'API (JSON, `net/http` seul,
  sans framework) et qui est le seul composant à accéder à la base de données
- un **client CLI** pour les utilisateurs (`cmd/client`), 100% ligne de
  commande, qui communique avec le serveur via `internal/apiclient`
- un **client CLI admin** pour la gestion (`cmd/admin`), séparé du client
  utilisateur, qui communique également avec le serveur via `internal/apiclient`
- une commande utilitaire de **seed** (données de démo / promotion admin)
  (`cmd/seed`), qui accède directement à la base (outil de développement)

Les CLI client et admin ne communiquent jamais directement avec la base de
données : toutes leurs actions passent par des requêtes HTTP vers `cmd/server`.

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

3. Démarrer le serveur HTTP (requis avant d'utiliser les CLI client/admin) :

   ```bash
   go run ./cmd/server
   ```

   Par défaut, le serveur utilise le fichier `ecommerce.db` (SQLite) à la
   racine du projet et écoute sur `:8080`.

4. Dans un autre terminal, lancer le client CLI (utilisateur) :

   ```bash
   go run ./cmd/client
   ```

5. Pour l'interface admin (dans un autre terminal également) :
   ```bash
   go run ./cmd/admin
   ```

### Variables d'environnement

| Variable      | Utilisé par                                 | Défaut                                            | Description                      |
| ------------- | ------------------------------------------- | ------------------------------------------------- | -------------------------------- |
| `DB_PATH`     | `server`, `seed`                            | `ecommerce.db`                                    | Chemin du fichier de base SQLite |
| `SERVER_ADDR` | `server` (écoute), `client`/`admin` (cible) | `:8080` (serveur) / `http://localhost:8080` (CLI) | Adresse du serveur HTTP          |

Exemple (serveur sur un port personnalisé, CLI pointant dessus) :

```bash
SERVER_ADDR=:9090 go run ./cmd/server
SERVER_ADDR=http://localhost:9090 go run ./cmd/client
SERVER_ADDR=http://localhost:9090 go run ./cmd/admin
```

## Lancer le projet avec Docker

Le `compose.yml` fourni monte le code source dans un conteneur Go et ouvre un shell interactif :

```bash
docker compose run --rm go sh
```

Une fois dans le conteneur, vous pouvez utiliser les mêmes commandes que ci-dessus (`go run ./cmd/server`, puis `go run ./cmd/client` / `go run ./cmd/admin` dans d'autres sessions).

## Tests

Pour exécuter les tests du projet :

```bash
go test ./...
```

## Structure du projet

Voir [`structure.md`](./structure.md) pour le détail de l'arborescence et le rôle de chaque dossier.
