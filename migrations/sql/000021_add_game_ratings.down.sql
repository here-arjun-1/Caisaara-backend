ALTER TABLE games
DROP COLUMN IF EXISTS white_rating_before,
DROP COLUMN IF EXISTS white_rating_after,
DROP COLUMN IF EXISTS white_provisional,
DROP COLUMN IF EXISTS black_rating_before,
DROP COLUMN IF EXISTS black_rating_after,
DROP COLUMN IF EXISTS black_provisional;
