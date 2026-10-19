ALTER TABLE game_analyses
ADD COLUMN IF NOT EXISTS opening_classification_white VARCHAR(20) DEFAULT '',
ADD COLUMN IF NOT EXISTS opening_classification_black VARCHAR(20) DEFAULT '',
ADD COLUMN IF NOT EXISTS middlegame_classification_white VARCHAR(20) DEFAULT '',
ADD COLUMN IF NOT EXISTS middlegame_classification_black VARCHAR(20) DEFAULT '',
ADD COLUMN IF NOT EXISTS endgame_classification_white VARCHAR(20) DEFAULT '',
ADD COLUMN IF NOT EXISTS endgame_classification_black VARCHAR(20) DEFAULT '',
ADD COLUMN IF NOT EXISTS opening_symbol_white VARCHAR(10) DEFAULT '',
ADD COLUMN IF NOT EXISTS opening_symbol_black VARCHAR(10) DEFAULT '',
ADD COLUMN IF NOT EXISTS middlegame_symbol_white VARCHAR(10) DEFAULT '',
ADD COLUMN IF NOT EXISTS middlegame_symbol_black VARCHAR(10) DEFAULT '',
ADD COLUMN IF NOT EXISTS endgame_symbol_white VARCHAR(10) DEFAULT '',
ADD COLUMN IF NOT EXISTS endgame_symbol_black VARCHAR(10) DEFAULT '';
