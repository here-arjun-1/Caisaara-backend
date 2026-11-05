CREATE TABLE IF NOT EXISTS mini_mates_levels (
    level_number INT PRIMARY KEY,
    puzzle_id VARCHAR(16) NOT NULL REFERENCES puzzles(id) ON DELETE CASCADE,
    chapter INT NOT NULL,
    target_moves INT NOT NULL,
    title VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_mini_mates_levels_chapter ON mini_mates_levels(chapter);
CREATE INDEX IF NOT EXISTS idx_mini_mates_levels_puzzle ON mini_mates_levels(puzzle_id);

CREATE TABLE IF NOT EXISTS user_mini_mates_progress (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current_level INT NOT NULL DEFAULT 1,
    total_stars INT NOT NULL DEFAULT 0,
    completed_levels INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_mini_mates_progress_stars ON user_mini_mates_progress(total_stars DESC);

CREATE TABLE IF NOT EXISTS user_mini_mates_level_history (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    level_number INT NOT NULL REFERENCES mini_mates_levels(level_number) ON DELETE CASCADE,
    stars INT NOT NULL DEFAULT 0,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    hint_used BOOLEAN NOT NULL DEFAULT FALSE,
    time_taken_ms INT NOT NULL DEFAULT 0,
    attempts INT NOT NULL DEFAULT 1,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, level_number)
);

CREATE INDEX IF NOT EXISTS idx_user_mini_mates_level_history_user_id ON user_mini_mates_level_history(user_id);
CREATE INDEX IF NOT EXISTS idx_user_mini_mates_level_history_level ON user_mini_mates_level_history(level_number);
ALTER TABLE user_mini_mates_level_history ADD COLUMN IF NOT EXISTS hint_used BOOLEAN NOT NULL DEFAULT FALSE;

INSERT INTO mini_mates_levels (level_number, puzzle_id, chapter, target_moves, title)
WITH t1_pool AS (
    SELECT id, rating, popularity
    FROM puzzles
    WHERE 'mateIn1' = ANY(themes)
    ORDER BY CASE WHEN rating >= 600 AND rating <= 1050 THEN 0 ELSE 1 END, popularity DESC, rating ASC
    LIMIT 35
),
t2_pool AS (
    SELECT id, rating, popularity
    FROM puzzles
    WHERE 'mateIn2' = ANY(themes)
      AND id NOT IN (SELECT id FROM t1_pool)
    ORDER BY CASE WHEN rating >= 1000 AND rating <= 1450 THEN 0 ELSE 1 END, popularity DESC, rating ASC
    LIMIT 50
),
t3_pool AS (
    SELECT id, rating, popularity
    FROM puzzles
    WHERE 'mateIn3' = ANY(themes)
      AND id NOT IN (SELECT id FROM t1_pool)
      AND id NOT IN (SELECT id FROM t2_pool)
    ORDER BY CASE WHEN rating >= 1400 AND rating <= 1850 THEN 0 ELSE 1 END, popularity DESC, rating ASC
    LIMIT 50
),
t4_pool AS (
    SELECT id, rating, popularity
    FROM puzzles
    WHERE 'mateIn4' = ANY(themes)
      AND id NOT IN (SELECT id FROM t1_pool)
      AND id NOT IN (SELECT id FROM t2_pool)
      AND id NOT IN (SELECT id FROM t3_pool)
    ORDER BY CASE WHEN rating >= 1800 AND rating <= 2150 THEN 0 ELSE 1 END, popularity DESC, rating ASC
    LIMIT 15
),
all_selected AS (
    SELECT id AS puzzle_id, 1 AS chapter, 1 AS target_moves, rating, popularity FROM t1_pool
    UNION ALL
    SELECT id AS puzzle_id, 2 AS chapter, 2 AS target_moves, rating, popularity FROM t2_pool
    UNION ALL
    SELECT id AS puzzle_id, 3 AS chapter, 3 AS target_moves, rating, popularity FROM t3_pool
    UNION ALL
    SELECT id AS puzzle_id, 4 AS chapter, 4 AS target_moves, rating, popularity FROM t4_pool
),
sequenced AS (
    SELECT 
        ROW_NUMBER() OVER (ORDER BY chapter ASC, rating ASC, popularity DESC)::INT AS level_number,
        puzzle_id,
        chapter,
        target_moves
    FROM all_selected
)
SELECT 
    level_number,
    puzzle_id,
    chapter,
    target_moves,
    'Level ' || level_number::TEXT AS title
FROM sequenced
ON CONFLICT (level_number) DO NOTHING;

