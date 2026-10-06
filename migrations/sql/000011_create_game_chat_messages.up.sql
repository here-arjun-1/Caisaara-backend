CREATE TABLE IF NOT EXISTS game_chat_messages (
    id UUID PRIMARY KEY,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_game_chat_messages_game_id ON game_chat_messages(game_id);
CREATE INDEX idx_game_chat_messages_created_at ON game_chat_messages(created_at);
