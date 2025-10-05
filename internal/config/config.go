package config

import (
	"os"
	"strconv"
)

type Config struct {
	Host                 string // Dirección IP para bind (0.0.0.0 permite acceso externo)
	Port                 string
	DatabaseURL          string
	JWTSecret            string
	MusicPath            string
	MaxConcurrentStreams int
	MaxDownloadsPerDay   int
	LastFMAPIKey         string
}

func Load() *Config {
	return &Config{
		Host:                 getEnv("HOST", "0.0.0.0"), // 0.0.0.0 permite acceso desde cualquier IP
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          getEnv("DATABASE_URL", "postgres://castafiore_user:castafiore_password@localhost/castafiore?sslmode=disable"),
		JWTSecret:            getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
		MusicPath:            getEnv("MUSIC_PATH", "./music"),
		MaxConcurrentStreams: getEnvInt("MAX_CONCURRENT_STREAMS", 3),
		MaxDownloadsPerDay:   getEnvInt("MAX_DOWNLOADS_PER_DAY", 50),
		LastFMAPIKey:         getEnv("LASTFM_API_KEY", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
