CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE IF NOT EXISTS player_locations (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    nearby_enabled BOOLEAN NOT NULL DEFAULT true,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_player_locations_location
    ON player_locations USING GIST (location);


