CREATE TABLE COMMENTS (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating_id INT NOT NULL REFERENCES ratings(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_rating_id ON ratings(id);
CREATE INDEX idc_comments_user_id ON users(id);