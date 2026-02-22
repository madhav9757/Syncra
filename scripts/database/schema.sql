-- User Table Schema
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    public_key_hash VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for username for faster lookups
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
-- Index for public_key_hash
CREATE INDEX IF NOT EXISTS idx_users_pk_hash ON users(public_key_hash);
