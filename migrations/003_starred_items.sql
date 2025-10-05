-- Crear tabla de artistas favoritos (starred)
CREATE TABLE IF NOT EXISTS starred_artists (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    artist_id INTEGER REFERENCES artists(id) ON DELETE CASCADE,
    starred_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, artist_id)
);

-- Crear tabla de álbumes favoritos (starred)
CREATE TABLE IF NOT EXISTS starred_albums (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    album_id INTEGER REFERENCES albums(id) ON DELETE CASCADE,
    starred_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, album_id)
);

-- Renombrar tabla favorites a starred_songs para consistencia
ALTER TABLE favorites RENAME TO starred_songs;

-- Agregar columna starred_at si no existe
ALTER TABLE starred_songs ADD COLUMN IF NOT EXISTS starred_at TIMESTAMP DEFAULT NOW();

-- Actualizar starred_at con created_at para registros existentes
UPDATE starred_songs SET starred_at = created_at WHERE starred_at IS NULL;

-- Crear índices para mejorar el rendimiento
CREATE INDEX IF NOT EXISTS idx_starred_artists_user_id ON starred_artists(user_id);
CREATE INDEX IF NOT EXISTS idx_starred_artists_artist_id ON starred_artists(artist_id);
CREATE INDEX IF NOT EXISTS idx_starred_albums_user_id ON starred_albums(user_id);
CREATE INDEX IF NOT EXISTS idx_starred_albums_album_id ON starred_albums(album_id);
CREATE INDEX IF NOT EXISTS idx_starred_songs_user_id ON starred_songs(user_id);
CREATE INDEX IF NOT EXISTS idx_starred_songs_song_id ON starred_songs(song_id);

-- Crear tabla de "now playing" para rastrear lo que los usuarios están escuchando actualmente
CREATE TABLE IF NOT EXISTS now_playing (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    song_id INTEGER REFERENCES songs(id) ON DELETE CASCADE,
    player_id VARCHAR(255),
    started_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, player_id)
);

CREATE INDEX IF NOT EXISTS idx_now_playing_user_id ON now_playing(user_id);
CREATE INDEX IF NOT EXISTS idx_now_playing_updated_at ON now_playing(updated_at);