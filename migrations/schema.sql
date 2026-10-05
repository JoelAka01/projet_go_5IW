-- Active les clés étrangères (désactivées par défaut en SQLite)
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    is_admin        BOOLEAN NOT NULL DEFAULT 0,
    confirmed       BOOLEAN NOT NULL DEFAULT 0,
    confirm_code    TEXT,               -- NULL une fois le compte confirmé
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    token           TEXT NOT NULL UNIQUE,
    user_id         INTEGER NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at      DATETIME NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);

CREATE TABLE IF NOT EXISTS products (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    category        TEXT NOT NULL DEFAULT '',
    price_cents     INTEGER NOT NULL,
    tax_rate_percent INTEGER NOT NULL DEFAULT 20,
    stock           INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_products_name ON products(name);
CREATE INDEX IF NOT EXISTS idx_products_category ON products(category);

CREATE TABLE IF NOT EXISTS cart_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    product_id      INTEGER NOT NULL,
    quantity        INTEGER NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE,
    UNIQUE (user_id, product_id)
);

-- Un panier "métier" identifié par une référence lisible (ex: BSK-1KH8E7),
-- distinct des lignes de cart_items (qui restent indexées par user_id pour
-- ne pas casser le fonctionnement existant). Un seul panier est "open" à la
-- fois par utilisateur : c'est celui que l'on sauvegarde / que l'on paie.
-- Une fois payé, son statut passe à 'paid' et un nouveau panier "open" est
-- créé pour les achats suivants.
CREATE TABLE IF NOT EXISTS carts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    reference       TEXT NOT NULL UNIQUE,
    status          TEXT NOT NULL DEFAULT 'open', -- open | paid
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    paid_at         DATETIME,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_carts_user_status ON carts(user_id, status);

CREATE TABLE IF NOT EXISTS orders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    total_cents     INTEGER NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS order_items (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    order_id        INTEGER NOT NULL,
    product_id      INTEGER NOT NULL,
    quantity        INTEGER NOT NULL,
    price_cents     INTEGER NOT NULL,

    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
);

-- Trace du paiement d'un panier : aucune donnée bancaire sensible n'est
-- stockée (ni numéro complet, ni CVC) conformément aux bonnes pratiques,
-- seulement les 4 derniers chiffres de la carte à titre de justificatif.
CREATE TABLE IF NOT EXISTS payments (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    cart_id         INTEGER NOT NULL,
    order_id        INTEGER NOT NULL,
    amount_cents    INTEGER NOT NULL,
    card_last4      TEXT NOT NULL,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (cart_id) REFERENCES carts(id) ON DELETE CASCADE,
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);
