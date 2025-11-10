CREATE TABLE genres (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE book_genres (
    book_id INT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    genre_id INT NOT NULL REFERENCES genres(id) ON DELETE CASCADE,
    PRIMARY KEY (book_id, genre_id)
);

CREATE INDEX idx_book_genres_book_id ON book_genres(book_id);
CREATE INDEX idx_book_genres_genre_id ON book_genres(genre_id);

INSERT INTO genres(name) VALUES
                             ('Fiction'),
                             ('Non-Fiction'),
                             ('Mystery'),
                             ('Thriller'),
                             ('Science Fiction'),
                             ('Fantasy'),
                             ('Romance'),
                             ('Horror'),
                             ('Biography'),
                             ('History'),
                             ('Self-Help'),
                             ('Business'),
                             ('Poetry'),
                             ('Young Adult'),
                             ('Classics');
