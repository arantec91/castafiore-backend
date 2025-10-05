package auth

import (
	"crypto/sha256"
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
func (s *Service) ValidateSubsonicAuth(db *sql.DB, username, password, salt string) (*User, error) {
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

	// Validate password using Subsonic method: MD5(password + salt)
	if salt != "" {
		// Create MD5 hash of password + salt
		hasher := sha256.New()
		hasher.Write([]byte(password + salt))
		hashedPassword := hex.EncodeToString(hasher.Sum(nil))

		// Compare with stored hash
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(hashedPassword)); err != nil {
			return nil, fmt.Errorf("invalid password")
		}
	} else {
		// Direct password comparison (for backwards compatibility)
		if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
			return nil, fmt.Errorf("invalid password")
		}
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
