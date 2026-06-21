ALTER TABLE mood_entries
    DROP CONSTRAINT mood_entries_mood_score_check;

ALTER TABLE mood_entries
    ALTER COLUMN mood_score TYPE NUMERIC(3,1) USING mood_score::numeric(3,1);

ALTER TABLE mood_entries
    ADD CONSTRAINT mood_entries_mood_score_check CHECK (mood_score >= 1 AND mood_score <= 10);
