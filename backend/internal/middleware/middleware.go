package middleware

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/freetorrent/freetorrent/internal/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
	UserRoleKey  contextKey = "user_role"
)

// AuthMiddleware validates JWT tokens
func AuthMiddleware(authService *auth.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		token := parts[1]
		claims, err := authService.ValidateAccessToken(token)
		if err != nil {
			if err == auth.ErrExpiredToken {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
					"error": "token expired",
					"code":  "TOKEN_EXPIRED",
				})
			}
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		// Store user info in context
		c.Locals(string(UserIDKey), claims.UserID)
		c.Locals(string(UserEmailKey), claims.Email)
		c.Locals(string(UserRoleKey), claims.Role)

		return c.Next()
	}
}

// AdminMiddleware ensures the user has admin role
func AdminMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		role := c.Locals(string(UserRoleKey))
		if role != "admin" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "admin access required",
			})
		}
		return c.Next()
	}
}

// GetUserID extracts user ID from context
func GetUserID(c *fiber.Ctx) (uuid.UUID, error) {
	userIDStr := c.Locals(string(UserIDKey)).(string)
	return uuid.Parse(userIDStr)
}

// GetUserRole extracts user role from context
func GetUserRole(c *fiber.Ctx) string {
	role := c.Locals(string(UserRoleKey))
	if role == nil {
		return ""
	}
	return role.(string)
}

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     int           // requests per window
	window   time.Duration // time window
	cleanup  time.Duration // cleanup interval
}

type bucket struct {
	tokens    int
	lastReset time.Time
}

func NewRateLimiter(rate int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		window:  window,
		cleanup: window * 2,
	}

	// Start cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, b := range rl.buckets {
			if now.Sub(b.lastReset) > rl.cleanup {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]
	
	if !exists || now.Sub(b.lastReset) >= rl.window {
		rl.buckets[key] = &bucket{
			tokens:    rl.rate - 1,
			lastReset: now,
		}
		return true
	}

	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

func (rl *RateLimiter) Remaining(key string) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, exists := rl.buckets[key]
	if !exists {
		return rl.rate
	}

	if time.Since(b.lastReset) >= rl.window {
		return rl.rate
	}

	return b.tokens
}

// RateLimitMiddleware applies rate limiting
func RateLimitMiddleware(rl *RateLimiter) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Use user ID if authenticated, otherwise IP
		key := c.IP()
		if userID := c.Locals(string(UserIDKey)); userID != nil {
			key = userID.(string)
		}

		if !rl.Allow(key) {
			c.Set("X-RateLimit-Limit", "100")
			c.Set("X-RateLimit-Remaining", "0")
			c.Set("Retry-After", "60")
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "rate limit exceeded",
				"code":  "RATE_LIMITED",
			})
		}

		remaining := rl.Remaining(key)
		c.Set("X-RateLimit-Limit", "100")
		c.Set("X-RateLimit-Remaining", string(rune(remaining)))

		return c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Set("Access-Control-Max-Age", "86400")

		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}

		return c.Next()
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("X-Request-ID", requestID)
		c.Locals("request_id", requestID)
		return c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-XSS-Protection", "1; mode=block")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Content-Security-Policy", "default-src 'self'")
		return c.Next()
	}
}

// ContextWithUser creates a context with user information
func ContextWithUser(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}
