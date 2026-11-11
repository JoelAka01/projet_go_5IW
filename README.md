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

| Variable          | Utilisé par                                              | Défaut                                                | Description                                |
| ----------------- | -------------------------------------------------------- | ----------------------------------------------------- | ------------------------------------------ |
| `DB_PATH`         | `server`, `seed`                                         | `ecommerce.db`                                        | Chemin du fichier de base SQLite           |
| `SERVER_ADDR`     | `server` (écoute), `client`/`admin`/`ssh-server` (cible) | `:8080` (serveur) / `http://localhost:8080` (CLI/SSH) | Adresse du serveur HTTP                    |
| `SSH_CLIENT_ADDR` | `ssh-server`                                             | `:2222`                                               | Adresse d'écoute SSH de l'interface client |
| `SSH_ADMIN_ADDR`  | `ssh-server`                                             | `:2223`                                               | Adresse d'écoute SSH de l'interface admin  |

Exemple (serveur sur un port personnalisé, CLI pointant dessus) :

```bash
SERVER_ADDR=:9090 go run ./cmd/server
SERVER_ADDR=http://localhost:9090 go run ./cmd/client
SERVER_ADDR=http://localhost:9090 go run ./cmd/admin
```

## Accès via SSH (wish)

1. Le serveur HTTP (`cmd/server`) doit être démarré au préalable (voir
   ci-dessus).

2. Démarrer le serveur SSH :

   ```bash
   go run ./cmd/ssh-server
   ```

   Par défaut, il écoute sur `:2222` (interface client) et `:2223`
   (interface admin), et cible l'API sur `http://localhost:8080`. Une clé
   d'hôte SSH est générée automatiquement au premier lancement dans
   `.ssh/id_ed25519` (dossier ignoré par git).

3. Se connecter avec n'importe quel client SSH :

   ```bash
   ssh -p 2222 localhost   # interface client
   ssh -p 2223 localhost   # interface admin
   ```

Options disponibles (flags, avec variables d'environnement équivalentes) :

```bash
go run ./cmd/ssh-server \
  --api http://localhost:8080 \
  --client-addr :2222 \
  --admin-addr :2223
```

## Lancer le projet avec Docker

Le `compose.yml` fourni monte le code source dans un conteneur Go. Le
serveur et les CLI doivent impérativement tourner **dans le même conteneur**
pour partager le même réseau (`localhost`) : chaque `docker compose run`
crée un nouveau conteneur isolé, donc lancer le serveur avec `run` dans un
terminal et le client avec `run` dans un autre terminal ne fonctionnera pas
(`connection refused`).

1. Démarrer un conteneur persistant en arrière-plan :

   ```bash
   docker compose up -d
   ```

2. Ouvrir un premier shell dans ce conteneur et lancer le serveur :

   ```bash
   docker compose exec go sh
   go run ./cmd/server
   ```

3. Dans un **autre terminal**, rejoindre le **même** conteneur (avec `exec`,
   pas `run`) pour lancer le client ou l'admin :

   ```bash
   docker compose exec go sh
   go run ./cmd/client
   ```

   ```bash
   docker compose exec go sh
   go run ./cmd/admin
   ```

4. (Optionnel) Pour exposer les TUI via SSH depuis le conteneur, ouvrir un
   autre shell dans le **même** conteneur :

   ```bash
   docker compose exec go sh
   go run ./cmd/ssh-server
   ```

Pour arrêter le conteneur :

```bash
docker compose down
```

## Image de production (Dockerfile.prod)

`Dockerfile` et `compose.yml` sont destinés au **développement** (montage du
code source, compilation à la volée avec `go run`). Pour un déploiement
(Azure, VM, etc.), utiliser `Dockerfile.prod` : un build multi-stage qui
compile des binaires statiques (`server`, `ssh-server`, `seed`) et produit
une image Alpine minimale, sans dépendre du code source monté.

Construire et tester l'image en local :

```bash
docker build -f Dockerfile.prod -t ecommerce-cli:prod .
docker compose -f compose.prod.yml up -d
```

Le conteneur lance `cmd/server` (API HTTP sur `:8080`) et `cmd/ssh-server`
(SSH sur `:2222`/`:2223`) via `docker/entrypoint.sh` ; si l'un des deux
process s'arrête, le conteneur s'arrête (pour que l'orchestrateur le
redémarre). La base SQLite est persistée dans un volume (`/home/app/data`),
de même que la clé d'hôte SSH (`/home/app/.ssh`) afin de ne pas regénérer
une nouvelle clé (et donc une alerte "host key changed" côté clients) à
chaque redémarrage du conteneur.

Variables d'environnement spécifiques à cette image :

| Variable  | Défaut                  | Description                                                                                                                         |
| --------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `API_URL` | `http://127.0.0.1:8080` | URL interne utilisée par `ssh-server` pour joindre l'API (distincte de `SERVER_ADDR`, qui reste l'adresse d'écoute de `cmd/server`) |

### Déployer sur Azure

Cette image Docker peut être hébergée sur **Azure Container Apps** ou
**Azure Container Instances** (le SSH nécessite un ingress TCP, pas
seulement HTTP) :

1. Pousser l'image vers un registre (Azure Container Registry) :

   ```bash
   az acr create --name <registry> --resource-group <rg> --sku Basic
   az acr build --registry <registry> --image ecommerce-cli:prod -f Dockerfile.prod .
   ```

2. Déployer un Container App avec :
   - un ingress TCP exposant les ports `2222` et `2223` (SSH),
   - le port `8080` non exposé publiquement (interne au conteneur),
   - un stockage persistant (Azure Files) monté sur `/home/app/data` pour
     que la base SQLite survive aux redémarrages/mises à l'échelle,
   - les variables d'environnement `DB_PATH`, `SERVER_ADDR`, `API_URL`,
     `SSH_CLIENT_ADDR`, `SSH_ADMIN_ADDR` si les valeurs par défaut ne
     conviennent pas.

Alternative plus simple pour un usage ponctuel/démo : une **Azure VM**
(Ubuntu) avec Docker installé, sur laquelle on exécute simplement
`docker compose -f compose.prod.yml up -d`, en ouvrant les ports `2222` et
`2223` dans le groupe de sécurité réseau (NSG).

## Tests

Pour exécuter les tests du projet :

```bash
go test ./...
```

## Structure du projet

Voir [`structure.md`](./structure.md) pour le détail de l'arborescence et le rôle de chaque dossier.
