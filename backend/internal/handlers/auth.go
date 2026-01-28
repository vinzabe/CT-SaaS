package handlers

import (
	"regexp"
	"time"

	"github.com/freetorrent/freetorrent/internal/auth"
	"github.com/freetorrent/freetorrent/internal/config"
	"github.com/freetorrent/freetorrent/internal/database"
	"github.com/freetorrent/freetorrent/internal/middleware"
	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AuthHandler struct {
	db   *database.Database
	auth *auth.AuthService
	cfg  *config.Config
}

func NewAuthHandler(db *database.Database, authService *auth.AuthService, cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		db:   db,
		auth: authService,
		cfg:  cfg,
	}
}

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Register creates a new user account
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	// Validate email
	if !emailRegex.MatchString(req.Email) {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid email format",
		})
	}

	// Validate password
	if err := auth.ValidatePassword(req.Password); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "weak password",
			Details: err.Error(),
		})
	}

	// Check if user exists
	existing, err := h.db.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "database error",
		})
	}
	if existing != nil {
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error: "email already registered",
			Code:  "EMAIL_EXISTS",
		})
	}

	// Hash password
	passwordHash, err := h.auth.HashPassword(req.Password)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to hash password",
		})
	}

	// Create user
	user, err := h.db.CreateUser(c.Context(), req.Email, passwordHash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to create user",
		})
	}

	// Generate tokens
	accessToken, err := h.auth.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate access token",
		})
	}

	refreshToken, tokenHash, err := h.auth.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate refresh token",
		})
	}

	// Save refresh token
	expiresAt := time.Now().AddDate(0, 0, h.cfg.JWTRefreshExpiry)
	if err := h.db.SaveRefreshToken(c.Context(), user.ID, tokenHash, expiresAt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to save refresh token",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    h.cfg.JWTAccessExpiry * 60,
		User:         user,
	})
}

// Login authenticates a user
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	// Get user
	user, err := h.db.GetUserByEmail(c.Context(), req.Email)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "database error",
		})
	}
	if user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid credentials",
		})
	}

	// Verify password
	if !h.auth.VerifyPassword(req.Password, user.PasswordHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid credentials",
		})
	}

	// Generate tokens
	accessToken, err := h.auth.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate access token",
		})
	}

	refreshToken, tokenHash, err := h.auth.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate refresh token",
		})
	}

	// Save refresh token
	expiresAt := time.Now().AddDate(0, 0, h.cfg.JWTRefreshExpiry)
	if err := h.db.SaveRefreshToken(c.Context(), user.ID, tokenHash, expiresAt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to save refresh token",
		})
	}

	return c.JSON(models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    h.cfg.JWTAccessExpiry * 60,
		User:         user,
	})
}

// Refresh generates a new access token using a refresh token
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	type RefreshRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	// Hash the refresh token and look it up
	tokenHash := h.auth.HashRefreshToken(req.RefreshToken)
	userID, err := h.db.GetRefreshToken(c.Context(), tokenHash)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "database error",
		})
	}
	if userID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid refresh token",
		})
	}

	// Get user
	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "user not found",
		})
	}

	// Delete old refresh token (rotation)
	h.db.DeleteRefreshToken(c.Context(), tokenHash)

	// Generate new tokens
	accessToken, err := h.auth.GenerateAccessToken(user.ID, user.Email, user.Role)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate access token",
		})
	}

	newRefreshToken, newTokenHash, err := h.auth.GenerateRefreshToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate refresh token",
		})
	}

	// Save new refresh token
	expiresAt := time.Now().AddDate(0, 0, h.cfg.JWTRefreshExpiry)
	if err := h.db.SaveRefreshToken(c.Context(), user.ID, newTokenHash, expiresAt); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to save refresh token",
		})
	}

	return c.JSON(models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    h.cfg.JWTAccessExpiry * 60,
		User:         user,
	})
}

// Logout invalidates the refresh token
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	type LogoutRequest struct {
		RefreshToken string `json:"refresh_token"`
	}

	var req LogoutRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	// Delete refresh token
	tokenHash := h.auth.HashRefreshToken(req.RefreshToken)
	h.db.DeleteRefreshToken(c.Context(), tokenHash)

	return c.JSON(models.SuccessResponse{
		Message: "logged out successfully",
	})
}

// Me returns the current user's information
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil || user == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "user not found",
		})
	}

	// Get subscription
	subscription, _ := h.db.GetSubscription(c.Context(), userID)

	// Get usage stats
	monthlyUsage, _ := h.db.GetMonthlyUsage(c.Context(), userID)
	activeTorrents, _ := h.db.CountActiveTorrents(c.Context(), userID)

	type MeResponse struct {
		User         *models.User         `json:"user"`
		Subscription *models.Subscription `json:"subscription"`
		Usage        models.UsageStats    `json:"usage"`
	}

	usedGB := float64(monthlyUsage) / (1024 * 1024 * 1024)
	limitGB := 2
	concurrentLimit := 1
	plan := "free"
	
	if subscription != nil {
		limitGB = subscription.DownloadLimitGB
		concurrentLimit = subscription.ConcurrentLimit
		plan = subscription.Plan
	}

	return c.JSON(MeResponse{
		User:         user,
		Subscription: subscription,
		Usage: models.UsageStats{
			UsedGB:          usedGB,
			LimitGB:         limitGB,
			ActiveTorrents:  activeTorrents,
			ConcurrentLimit: concurrentLimit,
			Plan:            plan,
		},
	})
}
