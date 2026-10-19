ALTER TABLE tournaments
ADD COLUMN IF NOT EXISTS min_rating INT NOT NULL DEFAULT 0 CHECK (min_rating >= 0);
