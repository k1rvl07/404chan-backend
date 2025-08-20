-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    ip INET NOT NULL UNIQUE,
    nickname TEXT NOT NULL DEFAULT 'Аноним' CHECK (LENGTH(nickname) >= 1 AND LENGTH(nickname) <= 16),
    last_nickname_change TIMESTAMPTZ, 
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_ip ON users (ip);

CREATE TABLE IF NOT EXISTS sessions (
    id BIGSERIAL PRIMARY KEY,
    session_key TEXT UNIQUE NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    user_agent TEXT,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_sessions_session_key ON sessions (session_key);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);

CREATE TABLE IF NOT EXISTS boards (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(20) UNIQUE NOT NULL CHECK (slug ~ '^[a-z0-9]+$'),
    title VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_boards_slug ON boards (slug);

CREATE TABLE IF NOT EXISTS threads (
    id BIGSERIAL PRIMARY KEY,
    board_id BIGINT NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL CHECK (LENGTH(title) >= 3 AND LENGTH(title) <= 99),
    content TEXT CHECK (LENGTH(content) >= 3 AND LENGTH(content) <= 999),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_session_id BIGINT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    author_nickname VARCHAR(16) NOT NULL DEFAULT 'Аноним' CHECK (LENGTH(author_nickname) >= 1 AND LENGTH(author_nickname) <= 16)
);
CREATE INDEX IF NOT EXISTS idx_threads_board_id ON threads (board_id);
CREATE INDEX IF NOT EXISTS idx_threads_created_by_session ON threads (created_by_session_id);

CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    thread_id BIGINT NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES messages (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages (thread_id);
CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages (user_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent_id ON messages (parent_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages (created_at DESC);

CREATE TABLE IF NOT EXISTS threads_activity (
    thread_id BIGINT PRIMARY KEY REFERENCES threads (id) ON DELETE CASCADE,
    message_count INTEGER NOT NULL DEFAULT 0,
    bump_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_threads_activity_bump_at ON threads_activity (bump_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_activity_message_count ON threads_activity (message_count DESC);

CREATE TABLE IF NOT EXISTS user_activities (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    thread_count INTEGER NOT NULL DEFAULT 0,
    message_count INTEGER NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    last_thread_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_activities_message_count ON user_activities (message_count);
CREATE INDEX IF NOT EXISTS idx_user_activities_last_message ON user_activities (last_message_at DESC);
CREATE INDEX IF NOT EXISTS idx_user_activities_last_thread ON user_activities (last_thread_at DESC);

CREATE OR REPLACE FUNCTION create_user_activity_on_user()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_activities (user_id, thread_count, message_count, last_message_at, last_thread_at, created_at, updated_at)
    VALUES (NEW.id, 0, 0, NULL, NULL, NOW(), NOW())
    ON CONFLICT (user_id) DO NOTHING;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_user_activity_on_user
    AFTER INSERT ON users
    FOR EACH ROW
    EXECUTE FUNCTION create_user_activity_on_user();

CREATE OR REPLACE FUNCTION update_user_and_thread_activity_on_message()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_activities (user_id, message_count, last_message_at)
    VALUES (NEW.user_id, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        message_count = user_activities.message_count + 1,
        last_message_at = NOW(),
        updated_at = NOW();

    INSERT INTO threads_activity (thread_id, message_count, bump_at)
    VALUES (NEW.thread_id, 1, NOW())
    ON CONFLICT (thread_id) DO UPDATE SET
        message_count = threads_activity.message_count + 1,
        bump_at = NOW(),
        updated_at = NOW();

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_and_thread_activity_on_message
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_user_and_thread_activity_on_message();

CREATE OR REPLACE FUNCTION create_threads_activity_on_thread()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO threads_activity (thread_id, message_count, bump_at)
    VALUES (NEW.id, 0, NOW())
    ON CONFLICT (thread_id) DO NOTHING;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_create_threads_activity_on_thread
    AFTER INSERT ON threads
    FOR EACH ROW
    EXECUTE FUNCTION create_threads_activity_on_thread();
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trigger_update_user_and_thread_activity_on_message ON messages;
DROP TRIGGER IF EXISTS trigger_create_threads_activity_on_thread ON threads;
DROP FUNCTION IF EXISTS update_user_and_thread_activity_on_message ();
DROP FUNCTION IF EXISTS create_threads_activity_on_thread ();
DROP FUNCTION IF EXISTS create_user_activity_on_user ();

DROP TABLE IF EXISTS threads_activity;
DROP TABLE IF EXISTS user_activities;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS threads;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS boards;
-- +goose StatementEnd