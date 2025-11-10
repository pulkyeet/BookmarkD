CREATE TYPE reading_status AS ENUM ('to_read', 'currently_reading', 'finished_reading');
ALTER TABLE ratings ADD COLUMN status reading_status DEFAULT 'finished_reading';
CREATE INDEX idx_reading_status ON ratings(status);