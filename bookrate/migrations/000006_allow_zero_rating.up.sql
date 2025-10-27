ALTER TABLE ratings DROP CONSTRAINT ratings_rating_check;
ALTER TABLE ratings ADD CONSTRAINT ratings_rating_check CHECK (rating >= 0 AND rating <= 10);