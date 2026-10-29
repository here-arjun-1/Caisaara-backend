ALTER TABLE post_reactions DROP CONSTRAINT IF EXISTS post_reactions_reaction_check;
ALTER TABLE post_reactions ADD CONSTRAINT post_reactions_reaction_check
    CHECK (reaction IN ('like', 'love', 'fire', 'wow', 'dislike'));
