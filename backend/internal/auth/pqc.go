package auth

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
	"github.com/google/uuid"
)

// Post-Quantum JWT using ML-DSA-65 (FIPS 204)
// This provides quantum-resistant digital signatures

var (
	ErrInvalidPQToken = errors.New("invalid post-quantum token")
	ErrPQKeyNotLoaded = errors.New("post-quantum keys not loaded")
)

// PQKeyPair holds the ML-DSA-65 key pair
type PQKeyPair struct {
	PublicKey  *mldsa65.PublicKey
	PrivateKey *mldsa65.PrivateKey
}

// PQClaims represents claims in a PQ-JWT
type PQClaims struct {
	UserID    string `json:"sub"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Issuer    string `json:"iss"`
	TokenID   string `json:"jti"`
}

// PQAuthService provides post-quantum authentication
type PQAuthService struct {
	keys      *PQKeyPair
	issuer    string
	expiryMin int
}

// NewPQAuthService creates a new post-quantum auth service
func NewPQAuthService(expiryMinutes int) (*PQAuthService, error) {
	// Generate new ML-DSA-65 key pair
	pub, priv, err := mldsa65.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return &PQAuthService{
		keys: &PQKeyPair{
			PublicKey:  pub,
			PrivateKey: priv,
		},
		issuer:    "grants-torrent",
		expiryMin: expiryMinutes,
	}, nil
}

// GeneratePQToken creates a post-quantum signed JWT
func (s *PQAuthService) GeneratePQToken(userID uuid.UUID, email, role string) (string, error) {
	if s.keys == nil || s.keys.PrivateKey == nil {
		return "", ErrPQKeyNotLoaded
	}

	now := time.Now()
	claims := PQClaims{
		UserID:    userID.String(),
		Email:     email,
		Role:      role,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Duration(s.expiryMin) * time.Minute).Unix(),
		Issuer:    s.issuer,
		TokenID:   uuid.New().String(),
	}

	// Create header
	header := map[string]string{
		"alg": "ML-DSA-65",
		"typ": "JWT",
	}

	// Encode header and claims
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signing input
	signingInput := headerB64 + "." + claimsB64

	// Sign with ML-DSA-65
	signature := make([]byte, mldsa65.SignatureSize)
	if err := mldsa65.SignTo(s.keys.PrivateKey, []byte(signingInput), nil, false, signature); err != nil {
		return "", err
	}

	// Encode signature
	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + signatureB64, nil
}

// ValidatePQToken verifies a post-quantum signed JWT
func (s *PQAuthService) ValidatePQToken(token string) (*PQClaims, error) {
	if s.keys == nil || s.keys.PublicKey == nil {
		return nil, ErrPQKeyNotLoaded
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidPQToken
	}

	headerB64, claimsB64, signatureB64 := parts[0], parts[1], parts[2]

	// Decode and verify header
	headerJSON, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return nil, ErrInvalidPQToken
	}

	var header map[string]string
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, ErrInvalidPQToken
	}

	if header["alg"] != "ML-DSA-65" {
		return nil, ErrInvalidPQToken
	}

	// Decode signature
	signature, err := base64.RawURLEncoding.DecodeString(signatureB64)
	if err != nil {
		return nil, ErrInvalidPQToken
	}

	// Verify signature
	signingInput := headerB64 + "." + claimsB64
	if !mldsa65.Verify(s.keys.PublicKey, []byte(signingInput), nil, signature) {
		return nil, ErrInvalidPQToken
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return nil, ErrInvalidPQToken
	}

	var claims PQClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, ErrInvalidPQToken
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

// GetPublicKeyPEM returns the public key for verification by other services
func (s *PQAuthService) GetPublicKeyBytes() []byte {
	if s.keys == nil || s.keys.PublicKey == nil {
		return nil
	}
	bytes, _ := s.keys.PublicKey.MarshalBinary()
	return bytes
}

// HybridAuthService combines classical JWT with PQ signatures
// This provides security even if one algorithm is broken
type HybridAuthService struct {
	classical *AuthService
	pq        *PQAuthService
}

// NewHybridAuthService creates a hybrid auth service
func NewHybridAuthService(classical *AuthService, pqExpiryMinutes int) (*HybridAuthService, error) {
	pq, err := NewPQAuthService(pqExpiryMinutes)
	if err != nil {
		return nil, err
	}

	return &HybridAuthService{
		classical: classical,
		pq:        pq,
	}, nil
}

// HybridToken contains both classical and PQ tokens
type HybridToken struct {
	Classical string `json:"classical"` // Standard HS256 JWT
	PQ        string `json:"pq"`        // ML-DSA-65 signed JWT
}

// GenerateHybridTokens creates both classical and PQ tokens
func (h *HybridAuthService) GenerateHybridTokens(userID uuid.UUID, email, role string) (*HybridToken, error) {
	classical, err := h.classical.GenerateAccessToken(userID, email, role)
	if err != nil {
		return nil, err
	}

	pq, err := h.pq.GeneratePQToken(userID, email, role)
	if err != nil {
		return nil, err
	}

	return &HybridToken{
		Classical: classical,
		PQ:        pq,
	}, nil
}

// ValidateHybridToken validates both tokens (both must be valid)
func (h *HybridAuthService) ValidateHybridToken(token *HybridToken) (*Claims, error) {
	// Validate classical token
	classicalClaims, err := h.classical.ValidateAccessToken(token.Classical)
	if err != nil {
		return nil, err
	}

	// Validate PQ token
	pqClaims, err := h.pq.ValidatePQToken(token.PQ)
	if err != nil {
		return nil, err
	}

	// Ensure both tokens refer to the same user
	if classicalClaims.UserID != pqClaims.UserID {
		return nil, ErrInvalidToken
	}

	return classicalClaims, nil
}
