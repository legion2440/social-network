CREATE TABLE auth_identities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    provider TEXT NOT NULL,
    provider_user_id TEXT NOT NULL,
    provider_email TEXT NOT NULL,
    provider_email_verified INTEGER NOT NULL CHECK (provider_email_verified IN (0, 1)),
    provider_username TEXT NOT NULL,
    provider_display_name TEXT NOT NULL,
    linked_at INTEGER NOT NULL,
    last_login_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (provider, provider_user_id),
    UNIQUE (provider, user_id)
);

CREATE INDEX idx_auth_identities_provider_email
ON auth_identities(provider, provider_email);
