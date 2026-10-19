ALTER TABLE game_analyses
DROP COLUMN IF EXISTS opening_classification_white,
DROP COLUMN IF EXISTS opening_classification_black,
DROP COLUMN IF EXISTS middlegame_classification_white,
DROP COLUMN IF EXISTS middlegame_classification_black,
DROP COLUMN IF EXISTS endgame_classification_white,
DROP COLUMN IF EXISTS endgame_classification_black,
DROP COLUMN IF EXISTS opening_symbol_white,
DROP COLUMN IF EXISTS opening_symbol_black,
DROP COLUMN IF EXISTS middlegame_symbol_white,
DROP COLUMN IF EXISTS middlegame_symbol_black,
DROP COLUMN IF EXISTS endgame_symbol_white,
DROP COLUMN IF EXISTS endgame_symbol_black;
