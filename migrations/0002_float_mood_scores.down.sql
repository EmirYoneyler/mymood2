ALTER TABLE mood_entries
    DROP CONSTRAINT mood_entries_mood_score_check;

ALTER TABLE mood_entries
    ALTER COLUMN mood_score TYPE SMALLINT USING round(mood_score)::smallint;

ALTER TABLE mood_entries
    ADD CONSTRAINT mood_entries_mood_score_check CHECK (mood_score BETWEEN 1 AND 10);
