CREATE TABLE year_ratings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    year SMALLINT NOT NULL CHECK (year BETWEEN 1900 AND 2200),
    score NUMERIC(3,1) NOT NULL CHECK (score >= 1 AND score <= 10),
    note VARCHAR(280),
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (user_id, year)
);
