CREATE TABLE list_bookmarks (
       user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
       list_id INT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
       PRIMARY KEY (user_id, list_id)
);

CREATE INDEX idx_list_bookmarks_user_id ON list_bookmarks(user_id);
CREATE INDEX idx_list_bookmarks_list_id ON list_bookmarks(list_id);