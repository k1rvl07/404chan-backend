-- +goose Up
-- +goose StatementBegin

-- Таблица: users
-- Анонимные пользователи, один на IP
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    ip INET NOT NULL UNIQUE, -- Один IP — один пользователь
    nickname TEXT NOT NULL DEFAULT 'Аноним',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_ip ON users (ip);

-- Таблица: sessions
-- Сессии, связанные с пользователем
CREATE TABLE IF NOT EXISTS sessions (
    id BIGSERIAL PRIMARY KEY,
    session_key TEXT UNIQUE NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMPTZ,
    user_agent TEXT,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE, -- Сессия принадлежит пользователю
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_session_key ON sessions (session_key);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions (user_id);

-- Таблица: boards
CREATE TABLE IF NOT EXISTS boards (
    id BIGSERIAL PRIMARY KEY,
    slug VARCHAR(20) UNIQUE NOT NULL CHECK (slug ~ '^[a-z0-9]+$'),
    title VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_boards_slug ON boards (slug);

-- Таблица: images
CREATE TABLE IF NOT EXISTS images (
    id BIGSERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    content_type TEXT NOT NULL,
    minio_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_images_minio_key ON images (minio_key);

-- Таблица: threads
CREATE TABLE IF NOT EXISTS threads (
    id BIGSERIAL PRIMARY KEY,
    board_id BIGINT NOT NULL REFERENCES boards (id) ON DELETE CASCADE,
    title VARCHAR(200),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_bump TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_session_id BIGINT NOT NULL REFERENCES sessions (id) ON DELETE CASCADE,
    image_id BIGINT REFERENCES images (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_threads_board_id ON threads (board_id);

CREATE INDEX IF NOT EXISTS idx_threads_last_bump ON threads (last_bump DESC);

CREATE INDEX IF NOT EXISTS idx_threads_created_by_session ON threads (created_by_session_id);

CREATE INDEX IF NOT EXISTS idx_threads_image_id ON threads (image_id);

-- Таблица: messages
CREATE TABLE IF NOT EXISTS messages (
    id BIGSERIAL PRIMARY KEY,
    thread_id BIGINT NOT NULL REFERENCES threads (id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES messages (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    image_id BIGINT REFERENCES images (id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_thread_id ON messages (thread_id);

CREATE INDEX IF NOT EXISTS idx_messages_user_id ON messages (user_id);

CREATE INDEX IF NOT EXISTS idx_messages_parent_id ON messages (parent_id);

CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_messages_image_id ON messages (image_id);

-- Таблица: user_activity
CREATE TABLE IF NOT EXISTS user_activity (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    thread_count INTEGER NOT NULL DEFAULT 0,
    message_count INTEGER NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_activity_message_count ON user_activity (message_count);

CREATE INDEX IF NOT EXISTS idx_user_activity_last_message ON user_activity (last_message_at DESC);

-- === ТРИГГЕРЫ ===

-- Функция: обновление user_activity при создании треда
CREATE OR REPLACE FUNCTION update_user_activity_on_thread()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_activity (user_id, thread_count)
    VALUES (
        (SELECT user_id FROM sessions WHERE id = NEW.created_by_session_id),
        1
    )
    ON CONFLICT (user_id) DO UPDATE SET
        thread_count = user_activity.thread_count + 1,
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_activity_on_thread
    AFTER INSERT ON threads
    FOR EACH ROW
    EXECUTE FUNCTION update_user_activity_on_thread();

-- Функция: обновление user_activity при создании сообщения
CREATE OR REPLACE FUNCTION update_user_activity_on_message()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO user_activity (user_id, message_count, last_message_at)
    VALUES (NEW.user_id, 1, NOW())
    ON CONFLICT (user_id) DO UPDATE SET
        message_count = user_activity.message_count + 1,
        last_message_at = NOW(),
        updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_user_activity_on_message
    AFTER INSERT ON messages
    FOR EACH ROW
    EXECUTE FUNCTION update_user_activity_on_message();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS trigger_update_user_activity_on_thread ON threads;

DROP TRIGGER IF EXISTS trigger_update_user_activity_on_message ON messages;

DROP FUNCTION IF EXISTS update_user_activity_on_thread ();

DROP FUNCTION IF EXISTS update_user_activity_on_message ();

DROP TABLE IF EXISTS user_activity;

DROP TABLE IF EXISTS messages;

DROP TABLE IF EXISTS threads;

DROP TABLE IF EXISTS sessions;

DROP TABLE IF EXISTS users;

DROP TABLE IF EXISTS boards;

DROP TABLE IF EXISTS images;

-- +goose StatementEnd