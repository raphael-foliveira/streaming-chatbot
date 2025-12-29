-- +goose Up
-- +goose StatementBegin
CREATE TABLE
  IF NOT EXISTS chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW ()
  );

CREATE TABLE
  IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    role VARCHAR(50) NOT NULL,
    content TEXT NOT NULL,
    chat_session_id UUID NOT NULL REFERENCES chats (id) ON DELETE CASCADE,
    reasoning_summary TEXT,
    name VARCHAR(255),
    args TEXT,
    call_id VARCHAR(255),
    result TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW ()
  );

CREATE INDEX idx_chat_messages_session ON chat_messages (chat_session_id);

CREATE INDEX idx_chat_messages_created_at ON chat_messages (created_at);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS chat_messages;

DROP TABLE IF EXISTS chats;

-- +goose StatementEnd