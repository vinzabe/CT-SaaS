package handlers

import (
	"strconv"
	"time"

	"github.com/freetorrent/freetorrent/internal/database"
	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/freetorrent/freetorrent/internal/torrent"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type AdminHandler struct {
	db     *database.Database
	engine *torrent.Engine
}

func NewAdminHandler(db *database.Database, engine *torrent.Engine) *AdminHandler {
	return &AdminHandler{
		db:     db,
		engine: engine,
	}
}

// ListUsers returns all users with pagination
func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	users, total, err := h.db.GetAllUsers(c.Context(), pageSize, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to fetch users",
		})
	}

	// Enrich with subscription info
	type UserWithSub struct {
		models.User
		Subscription *models.Subscription `json:"subscription,omitempty"`
	}

	enrichedUsers := make([]UserWithSub, len(users))
	for i, user := range users {
		enrichedUsers[i].User = user
		sub, _ := h.db.GetSubscription(c.Context(), user.ID)
		enrichedUsers[i].Subscription = sub
	}

	return c.JSON(fiber.Map{
		"users":       enrichedUsers,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// GetUser returns a specific user's details
func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid user ID",
		})
	}

	user, err := h.db.GetUserByID(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "database error",
		})
	}
	if user == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "user not found",
		})
	}

	// Get subscription
	subscription, _ := h.db.GetSubscription(c.Context(), userID)

	// Get usage stats
	monthlyUsage, _ := h.db.GetMonthlyUsage(c.Context(), userID)
	activeTorrents, _ := h.db.CountActiveTorrents(c.Context(), userID)

	// Get torrents
	torrents, totalTorrents, _ := h.db.GetTorrentsByUser(c.Context(), userID, 10, 0)

	return c.JSON(fiber.Map{
		"user":         user,
		"subscription": subscription,
		"usage": fiber.Map{
			"monthly_bytes":   monthlyUsage,
			"monthly_gb":      float64(monthlyUsage) / (1024 * 1024 * 1024),
			"active_torrents": activeTorrents,
		},
		"torrents": fiber.Map{
			"items": torrents,
			"total": totalTorrents,
		},
	})
}

// UpdateUser updates a user's role or subscription
func (h *AdminHandler) UpdateUser(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid user ID",
		})
	}

	type UpdateRequest struct {
		Role string `json:"role,omitempty"`
		Plan string `json:"plan,omitempty"`
	}

	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	// Update role if provided
	if req.Role != "" {
		validRoles := map[string]bool{"user": true, "premium": true, "admin": true}
		if !validRoles[req.Role] {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid role",
			})
		}
		if err := h.db.UpdateUserRole(c.Context(), userID, req.Role); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
				Error: "failed to update role",
			})
		}
	}

	// Update plan if provided
	if req.Plan != "" {
		limits, ok := models.Plans[req.Plan]
		if !ok {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid plan",
			})
		}
		if err := h.db.UpdateSubscription(c.Context(), userID, req.Plan, "active", limits); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
				Error: "failed to update subscription",
			})
		}
	}

	return c.JSON(models.SuccessResponse{
		Message: "user updated",
	})
}

// DeleteUser removes a user and all their data
func (h *AdminHandler) DeleteUser(c *fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid user ID",
		})
	}

	// Get user's torrents and remove them from engine
	torrents, _, _ := h.db.GetTorrentsByUser(c.Context(), userID, 1000, 0)
	for _, t := range torrents {
		h.engine.RemoveTorrent(t.InfoHash, true)
	}

	// Delete user (cascades to torrents, subscriptions, etc.)
	if err := h.db.DeleteUser(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to delete user",
		})
	}

	return c.JSON(models.SuccessResponse{
		Message: "user deleted",
	})
}

// ListAllTorrents returns all torrents across all users
func (h *AdminHandler) ListAllTorrents(c *fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	torrents, total, err := h.db.GetAllTorrents(c.Context(), pageSize, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to fetch torrents",
		})
	}

	// Enrich with live stats
	for i := range torrents {
		if status, err := h.engine.GetTorrentStatus(torrents[i].InfoHash); err == nil {
			torrents[i].DownloadSpeed = status.DownloadSpeed
			torrents[i].Progress = status.Progress
			if status.Status != "" {
				torrents[i].Status = status.Status
			}
		}
	}

	return c.JSON(fiber.Map{
		"torrents":    torrents,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
	})
}

// DeleteTorrent removes any torrent (admin override)
func (h *AdminHandler) DeleteTorrent(c *fiber.Ctx) error {
	torrentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid torrent ID",
		})
	}

	t, err := h.db.GetTorrent(c.Context(), torrentID)
	if err != nil || t == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "torrent not found",
		})
	}

	deleteFiles := c.Query("delete_files", "true") == "true"

	// Remove from engine
	h.engine.RemoveTorrent(t.InfoHash, deleteFiles)

	// Remove from database
	if err := h.db.DeleteTorrent(c.Context(), torrentID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to delete torrent",
		})
	}

	return c.JSON(models.SuccessResponse{
		Message: "torrent deleted",
	})
}

// GetStats returns platform-wide statistics
func (h *AdminHandler) GetStats(c *fiber.Ctx) error {
	// User counts
	users, totalUsers, _ := h.db.GetAllUsers(c.Context(), 1, 0)
	_ = users // unused, we just need total

	// Torrent counts
	torrents, totalTorrents, _ := h.db.GetAllTorrents(c.Context(), 1, 0)
	_ = torrents // unused

	// Active torrents from engine
	activeTorrents := h.engine.GetActiveTorrents()
	
	var totalDownloading, totalSeeding, totalCompleted int
	var totalDownloadSpeed, totalUploadSpeed float64
	
	for _, t := range activeTorrents {
		switch t.Status {
		case "downloading":
			totalDownloading++
		case "seeding":
			totalSeeding++
		case "completed":
			totalCompleted++
		}
		totalDownloadSpeed += t.DownloadSpeed
		totalUploadSpeed += t.UploadSpeed
	}

	// Subscription breakdown
	type PlanCount struct {
		Plan  string `json:"plan"`
		Count int    `json:"count"`
	}

	return c.JSON(fiber.Map{
		"users": fiber.Map{
			"total": totalUsers,
		},
		"torrents": fiber.Map{
			"total":       totalTorrents,
			"active":      len(activeTorrents),
			"downloading": totalDownloading,
			"seeding":     totalSeeding,
			"completed":   totalCompleted,
		},
		"bandwidth": fiber.Map{
			"download_speed_bps": totalDownloadSpeed,
			"upload_speed_bps":   totalUploadSpeed,
		},
		"timestamp": time.Now(),
	})
}

// CleanupExpired removes expired torrents
func (h *AdminHandler) CleanupExpired(c *fiber.Ctx) error {
	expired, err := h.db.GetExpiredTorrents(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to fetch expired torrents",
		})
	}

	var cleaned int
	for _, t := range expired {
		h.engine.RemoveTorrent(t.InfoHash, true)
		h.db.DeleteTorrent(c.Context(), t.ID)
		cleaned++
	}

	return c.JSON(fiber.Map{
		"message": "cleanup complete",
		"removed": cleaned,
	})
}
