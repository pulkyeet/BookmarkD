CREATE TABLE review_likes (
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating_id INT NOT NULL REFERENCES ratings(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, rating_id)
);

CREATE INDEX idx_review_like_rating_id ON review_likes(rating_id);