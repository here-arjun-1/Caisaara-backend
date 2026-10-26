CREATE TABLE IF NOT EXISTS user_privacy_settings (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    direct_message VARCHAR(20) NOT NULL DEFAULT 'everyone' CHECK (direct_message IN ('everyone', 'only_friends', 'nobody')),
    club VARCHAR(20) NOT NULL DEFAULT 'everyone' CHECK (club IN ('everyone', 'only_friends', 'nobody')),
    tournament VARCHAR(20) NOT NULL DEFAULT 'everyone' CHECK (tournament IN ('everyone', 'only_friends', 'nobody')),
    invite_friend VARCHAR(20) NOT NULL DEFAULT 'everyone' CHECK (invite_friend IN ('everyone', 'only_friends', 'nobody')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
