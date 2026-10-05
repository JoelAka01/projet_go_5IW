ecommerce-cli/
├── cmd/
│ ├── client/ // binaire CLI client (communique en HTTP avec cmd/server)
│ │ └── main.go
│ ├── admin/ // binaire CLI admin, séparé du client (communique en HTTP avec cmd/server)
│ │ └── main.go
│ ├── server/ // binaire serveur HTTP (net/http, sans framework)
│ │ └── main.go
│ └── seed/ // utilitaire de dev : données de démo / promotion admin
│ └── main.go
├── internal/
│ ├── models/ // structs partagées : User, Product, Cart, Order
│ ├── store/ // couche persistence (database/sql, SQLite), utilisée uniquement par cmd/server et cmd/seed
│ │ ├── store.go
│ │ ├── users.go
│ │ ├── products.go
│ │ ├── carts.go
│ │ └── orders.go
│ ├── api/ // DTOs JSON partagés entre le serveur et les clients HTTP
│ ├── httpserver/ // handlers net/http exposant l'API (utilisés par cmd/server)
│ ├── apiclient/ // client HTTP utilisé par cmd/client et cmd/admin
│ └── auth/ // hash mot de passe, tokens, codes de confirmation
├── ui/
│ ├── client/ // écrans Bubble Tea côté utilisateur
│ └── admin/ // écrans Bubble Tea côté admin
├── migrations/ // SQL de création des tables
├── go.mod
└── docker-compose.yml // pour PostgreSQL
