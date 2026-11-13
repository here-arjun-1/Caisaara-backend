ALTER TABLE tournament_pairings DROP CONSTRAINT IF EXISTS tournament_pairings_result_check;

ALTER TABLE tournament_pairings
    ADD CONSTRAINT tournament_pairings_result_check
    CHECK (result IS NULL OR result IN ('white_win', 'black_win', 'draw', 'bye', 'aborted'));
