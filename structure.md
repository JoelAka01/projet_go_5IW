ecommerce-cli/
├── cmd/
│   ├── server/        // binaire serveur HTTP
│   │   └── main.go
│   ├── client/         // binaire CLI client (Bubble Tea)
│   │   └── main.go
│   ├── admin/          // binaire CLI admin (Bubble Tea)
│   │   └── main.go
│   └── ssh-client/     // (optionnel, ou fusionné avec client/admin via wish)
├── internal/
│   ├── models/         // structs partagées : User, Product, Cart, Order
│   ├── store/           // couche persistence (database/sql)
│   │   ├── postgres.go / sqlite.go
│   │   ├── users.go
│   │   ├── products.go
│   │   ├── carts.go
│   │   └── orders.go
│   ├── httpapi/         // handlers net/http + routing manuel
│   │   ├── router.go
│   │   ├── auth.go
│   │   ├── products.go
│   │   ├── cart.go
│   │   └── orders.go
│   ├── apiclient/       // client HTTP utilisé par les TUI (appelle httpapi)
│   └── auth/             // hash mot de passe, tokens, codes de confirmation
├── ui/
│   ├── client/           // écrans Bubble Tea côté utilisateur
│   └── admin/            // écrans Bubble Tea côté admin
├── migrations/           // SQL de création des tables
├── go.mod
└── docker-compose.yml    // pour PostgreSQL