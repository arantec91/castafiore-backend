-- Test queries to check database content and search functionality

-- Check if there are artists
SELECT COUNT(*) as artist_count FROM artists;
SELECT * FROM artists LIMIT 5;

-- Check if there are albums
SELECT COUNT(*) as album_count FROM albums;
SELECT * FROM albums LIMIT 5;

-- Check if there are songs
SELECT COUNT(*) as song_count FROM songs;
SELECT * FROM songs LIMIT 5;

-- Test search for "arrolladora"
SELECT ar.id, ar.name, COUNT(al.id) as album_count
FROM artists ar
LEFT JOIN albums al ON ar.id = al.artist_id
WHERE LOWER(ar.name) LIKE '%arrolladora%'
GROUP BY ar.id, ar.name
ORDER BY ar.name;

-- Check user antonio
SELECT id, username, email, is_admin FROM users WHERE username = 'antonio';