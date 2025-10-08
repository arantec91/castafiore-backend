-- Insertar datos de ejemplo para desarrollo

-- Insertar usuario administrador por defecto
INSERT INTO users (username, email, password_hash, subsonic_password, subscription_plan, max_concurrent_streams, max_downloads_per_day, is_admin, is_active) 
VALUES (
    'admin', 
    'admin@castafiore.local', 
    '$2a$10$ij8kedkoiIMhliknzaVswexWxGVzpKXBhqK.d24.CeZk0t1q8ZM/e', -- admin123
    'admin123', -- Subsonic password (plain text for MD5 token generation)
    'premium',
    10,
    1000,
    TRUE,
    TRUE
) ON CONFLICT (username) DO UPDATE SET 
    password_hash = EXCLUDED.password_hash,
    subsonic_password = EXCLUDED.subsonic_password,
    is_admin = TRUE,
    is_active = TRUE;

-- Insertar artistas de ejemplo
INSERT INTO artists (name, bio) VALUES 
    ('The Beatles', 'Legendary British rock band'),
    ('Pink Floyd', 'Progressive rock pioneers'),
    ('Queen', 'Iconic British rock band'),
    ('Led Zeppelin', 'Hard rock legends')
ON CONFLICT DO NOTHING;

-- Insertar álbumes de ejemplo
INSERT INTO albums (name, artist_id, year, genre) VALUES 
    ('Abbey Road', 1, 1969, 'Rock'),
    ('Sgt. Pepper''s Lonely Hearts Club Band', 1, 1967, 'Rock'),
    ('The Dark Side of the Moon', 2, 1973, 'Progressive Rock'),
    ('The Wall', 2, 1979, 'Progressive Rock'),
    ('A Night at the Opera', 3, 1975, 'Rock'),
    ('Led Zeppelin IV', 4, 1971, 'Hard Rock')
ON CONFLICT DO NOTHING;

-- Insertar canciones de ejemplo (sin archivos reales)
INSERT INTO songs (title, artist_id, album_id, track_number, duration, file_path, format) VALUES 
    ('Come Together', 1, 1, 1, 259, '/music/The Beatles/Abbey Road/01 - Come Together.mp3', 'mp3'),
    ('Something', 1, 1, 2, 182, '/music/The Beatles/Abbey Road/02 - Something.mp3', 'mp3'),
    ('Money', 2, 3, 6, 382, '/music/Pink Floyd/The Dark Side of the Moon/06 - Money.mp3', 'mp3'),
    ('Time', 2, 3, 4, 413, '/music/Pink Floyd/The Dark Side of the Moon/04 - Time.mp3', 'mp3'),
    ('Bohemian Rhapsody', 3, 5, 11, 355, '/music/Queen/A Night at the Opera/11 - Bohemian Rhapsody.mp3', 'mp3'),
    ('Stairway to Heaven', 4, 6, 4, 482, '/music/Led Zeppelin/Led Zeppelin IV/04 - Stairway to Heaven.mp3', 'mp3')
ON CONFLICT (file_path) DO NOTHING;
