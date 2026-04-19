CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    client_id TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE,
    mobile TEXT UNIQUE,
    password_hash TEXT,
    mpin_hash TEXT,
    has_mpin BOOLEAN DEFAULT FALSE,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);