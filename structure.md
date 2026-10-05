ecommerce-cli/
├── cmd/
│ ├── client/ // binaire CLI client (Bubble Tea)
│ │ └── main.go
│ ├── admin/ // binaire CLI admin (Bubble Tea)
│ │ └── main.go
│ └── seed/ // utilitaire de dev : données de démo / promotion admin
│ └── main.go
├── internal/
│ ├── models/ // structs partagées : User, Product, Cart, Order
│ ├── store/ // couche persistence (database/sql, SQLite)
│ │ ├── store.go
│ │ ├── users.go
│ │ ├── products.go
│ │ ├── carts.go
│ │ └── orders.go
│ └── auth/ // hash mot de passe, tokens, codes de confirmation
├── ui/
│ ├── client/ // écrans Bubble Tea côté utilisateur
│ └── admin/ // écrans Bubble Tea côté admin
├── migrations/ // SQL de création des tables
├── go.mod
└── docker-compose.yml // pour PostgreSQL
