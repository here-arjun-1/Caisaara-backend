ALTER TABLE users
ADD COLUMN IF NOT EXISTS start_rating INTEGER NOT NULL DEFAULT 1200;

UPDATE users
SET start_rating = CASE skill_level
    WHEN 'new' THEN 400
    WHEN 'beginner' THEN 800
    WHEN 'intermediate' THEN 1200
    WHEN 'advanced' THEN 1600
    ELSE 1200
END;

UPDATE users u
SET rating = pr.rating,
    rating_deviation = pr.rating_deviation
FROM (
    SELECT DISTINCT ON (user_id) user_id, rating, rating_deviation
    FROM player_ratings
    ORDER BY user_id, games_played DESC, updated_at DESC
) pr
WHERE pr.user_id = u.id;
