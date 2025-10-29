CREATE TABLE IF NOT EXISTS user_invitations (
    id UUID PRIMARY KEY NOT NULL,
    user_id BIGINT NOT NULL,
    expires_at TIMESTAMP NOT NULL
);