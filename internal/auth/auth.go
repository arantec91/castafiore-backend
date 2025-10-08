package auth

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	jwtSecret []byte
}

type User struct {
	ID                   int    `json:"id"`
	Username             string `json:"username"`
	Email                string `json:"email"`
	SubscriptionPlan     string `json:"subscription_plan"`
	MaxConcurrentStreams int    `json:"max_concurrent_streams"`
	MaxDownloadsPerDay   int    `json:"max_downloads_per_day"`
}

type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewService(jwtSecret string) *Service {
	return &Service{
		jwtSecret: []byte(jwtSecret),
	}
}

// ValidateAdminAuth validates admin login credentials (simple bcrypt comparison)
func (s *Service) ValidateAdminAuth(db *sql.DB, username, password string) (*User, error) {
	// Get user from database
	user := &User{}
	var passwordHash string

	query := `
		SELECT id, username, email, password_hash, subscription_plan, 
		       max_concurrent_streams, max_downloads_per_day 
		FROM users WHERE username = $1`

	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash,
		&user.SubscriptionPlan, &user.MaxConcurrentStreams, &user.MaxDownloadsPerDay,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Direct bcrypt password comparison
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// ValidateSubsonicAuth validates Subsonic-style authentication
func (s *Service) ValidateSubsonicAuth(db *sql.DB, username, token, salt string) (*User, error) {
	// Get user from database
	user := &User{}
	var passwordHash string
	var subsonicPassword sql.NullString

	query := `
		SELECT id, username, email, password_hash, subsonic_password, subscription_plan, 
		       max_concurrent_streams, max_downloads_per_day 
		FROM users WHERE username = $1`

	err := db.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &passwordHash, &subsonicPassword,
		&user.SubscriptionPlan, &user.MaxConcurrentStreams, &user.MaxDownloadsPerDay,
	)

	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if subsonic_password is configured
	if !subsonicPassword.Valid || subsonicPassword.String == "" {
		return nil, fmt.Errorf("subsonic authentication not configured for this user")
	}

	// Validate using Subsonic token-based authentication: token = MD5(password + salt)
	if salt != "" && token != "" {
		// Calculate expected token: MD5(password + salt)
		hasher := md5.New()
		hasher.Write([]byte(subsonicPassword.String + salt))
		expectedToken := hex.EncodeToString(hasher.Sum(nil))

		// Compare tokens
		if token != expectedToken {
			return nil, fmt.Errorf("invalid token")
		}
	} else if token != "" {
		// Plain password authentication (p parameter instead of t/s)
		// This is when client sends password directly (enc:)
		if token != subsonicPassword.String {
			return nil, fmt.Errorf("invalid password")
		}
	} else {
		return nil, fmt.Errorf("no authentication credentials provided")
	}

	return user, nil
}

func (s *Service) GenerateJWT(user *User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return s.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// HashPasswordMD5 generates MD5 hash for Subsonic authentication
func HashPasswordMD5(password string) string {
	hasher := md5.New()
	hasher.Write([]byte(password))
	return hex.EncodeToString(hasher.Sum(nil))
}
