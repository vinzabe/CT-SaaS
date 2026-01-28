package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/freetorrent/freetorrent/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrExpiredToken     = errors.New("token has expired")
	ErrInvalidPassword  = errors.New("invalid password")
)

// Argon2 parameters (OWASP recommended)
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type AuthService struct {
	cfg *config.Config
}

func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// HashPassword creates an Argon2id hash of the password
func (a *AuthService) HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	
	// Encode as: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	
	return "$argon2id$v=19$m=65536,t=3,p=4$" + b64Salt + "$" + b64Hash, nil
}

// VerifyPassword checks if the provided password matches the hash
func (a *AuthService) VerifyPassword(password, encodedHash string) bool {
	// Parse the encoded hash
	// Format: $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	if len(encodedHash) < 40 {
		return false
	}
	
	// Find the salt and hash parts
	parts := splitArgon2Hash(encodedHash)
	if parts == nil {
		return false
	}
	
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	
	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	
	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	
	// Constant-time comparison
	if len(computedHash) != len(expectedHash) {
		return false
	}
	
	var diff byte
	for i := 0; i < len(computedHash); i++ {
		diff |= computedHash[i] ^ expectedHash[i]
	}
	
	return diff == 0
}

func splitArgon2Hash(encoded string) []string {
	// Simple parser for $argon2id$v=19$m=65536,t=3,p=4$<salt>$<hash>
	if len(encoded) < 30 {
		return nil
	}
	
	// Find the last two $ separators
	lastDollar := -1
	secondLastDollar := -1
	for i := len(encoded) - 1; i >= 0; i-- {
		if encoded[i] == '$' {
			if lastDollar == -1 {
				lastDollar = i
			} else {
				secondLastDollar = i
				break
			}
		}
	}
	
	if secondLastDollar == -1 || lastDollar == -1 {
		return nil
	}
	
	salt := encoded[secondLastDollar+1 : lastDollar]
	hash := encoded[lastDollar+1:]
	
	return []string{salt, hash}
}

// GenerateAccessToken creates a new JWT access token
func (a *AuthService) GenerateAccessToken(userID uuid.UUID, email, role string) (string, error) {
	claims := &Claims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(a.cfg.JWTAccessExpiry) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "ct-saas",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.cfg.JWTSecret))
}

// GenerateRefreshToken creates a secure random refresh token
func (a *AuthService) GenerateRefreshToken() (string, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}
	
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	hash := sha256.Sum256([]byte(token))
	tokenHash := hex.EncodeToString(hash[:])
	
	return token, tokenHash, nil
}

// ValidateAccessToken validates and parses a JWT access token
func (a *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(a.cfg.JWTSecret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// HashRefreshToken creates a SHA-256 hash of the refresh token for storage
func (a *AuthService) HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// GenerateDownloadToken creates a secure random download token
func GenerateDownloadToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// ValidatePassword checks password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	
	var hasUpper, hasLower, hasNumber bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		}
	}
	
	if !hasUpper || !hasLower || !hasNumber {
		return errors.New("password must contain uppercase, lowercase, and numbers")
	}
	
	return nil
}
