-- Add scan metadata table for incremental scanning
CREATE TABLE IF NOT EXISTS scan_metadata (
    id INTEGER PRIMARY KEY DEFAULT 1,
    last_scan_time TIMESTAMP,
    total_files_last_scan INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT single_row CHECK (id = 1)
);

-- Insert initial record
INSERT INTO scan_metadata (id, last_scan_time) 
VALUES (1, NOW()) 
ON CONFLICT (id) DO NOTHING;

-- Add indexes for better performance on large libraries
CREATE INDEX IF NOT EXISTS idx_songs_file_path_btree ON songs USING btree(file_path);
CREATE INDEX IF NOT EXISTS idx_songs_artist_album ON songs(artist_id, album_id);
CREATE INDEX IF NOT EXISTS idx_albums_artist_name ON albums(artist_id, name);
CREATE INDEX IF NOT EXISTS idx_artists_name_btree ON artists USING btree(name);

-- Add partial indexes for better performance
CREATE INDEX IF NOT EXISTS idx_songs_format ON songs(format) WHERE format IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_songs_duration ON songs(duration) WHERE duration > 0;
CREATE INDEX IF NOT EXISTS idx_songs_bitrate ON songs(bitrate) WHERE bitrate > 0;
