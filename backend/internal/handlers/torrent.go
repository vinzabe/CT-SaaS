package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/freetorrent/freetorrent/internal/auth"
	"github.com/freetorrent/freetorrent/internal/database"
	"github.com/freetorrent/freetorrent/internal/middleware"
	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/freetorrent/freetorrent/internal/torrent"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type TorrentHandler struct {
	db     *database.Database
	engine *torrent.Engine
}

func NewTorrentHandler(db *database.Database, engine *torrent.Engine) *TorrentHandler {
	return &TorrentHandler{
		db:     db,
		engine: engine,
	}
}

// AddTorrent adds a new torrent from magnet link or URL
func (h *TorrentHandler) AddTorrent(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	var req models.AddTorrentRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	// Check quota
	if err := h.checkQuota(c, userID); err != nil {
		return err
	}

	// Must have either magnet or URL
	if req.MagnetURI == "" && req.TorrentURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "magnet_uri or torrent_url required",
		})
	}

	torrentID := uuid.New()
	var update *torrent.TorrentUpdate

	if req.MagnetURI != "" {
		// Validate magnet link
		if !strings.HasPrefix(req.MagnetURI, "magnet:") {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "invalid magnet URI",
			})
		}

		update, err = h.engine.AddMagnet(c.Context(), torrentID, userID, req.MagnetURI)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error:   "failed to add magnet",
				Details: err.Error(),
			})
		}
	} else {
		// Download torrent file from URL
		resp, err := http.Get(req.TorrentURL)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error:   "failed to download torrent file",
				Details: err.Error(),
			})
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error: "failed to download torrent file: " + resp.Status,
			})
		}

		update, err = h.engine.AddTorrentFile(c.Context(), torrentID, userID, resp.Body)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Error:   "failed to parse torrent file",
				Details: err.Error(),
			})
		}
	}

	// Check if torrent already exists
	if update.Status == "exists" {
		// Return existing torrent from database
		existing, err := h.db.GetTorrentByInfoHash(c.Context(), userID, update.InfoHash)
		if err == nil && existing != nil {
			return c.Status(fiber.StatusOK).JSON(existing)
		}
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error: "torrent already exists",
			Code:  "TORRENT_EXISTS",
		})
	}

	// Save to database
	t := &models.Torrent{
		ID:        torrentID,
		UserID:    userID,
		InfoHash:  update.InfoHash,
		Name:      update.Name,
		MagnetURI: req.MagnetURI,
		Status:    update.Status,
		TotalSize: update.TotalSize,
	}

	if err := h.db.CreateTorrent(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to save torrent",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(t)
}

// UploadTorrent handles .torrent file uploads
func (h *TorrentHandler) UploadTorrent(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	// Check quota
	if err := h.checkQuota(c, userID); err != nil {
		return err
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "no file uploaded",
		})
	}

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".torrent") {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "file must be a .torrent file",
		})
	}

	// Open file
	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to open file",
		})
	}
	defer f.Close()

	torrentID := uuid.New()
	update, err := h.engine.AddTorrentFile(c.Context(), torrentID, userID, f)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error:   "failed to parse torrent file",
			Details: err.Error(),
		})
	}

	// Check if torrent already exists
	if update.Status == "exists" {
		existing, err := h.db.GetTorrentByInfoHash(c.Context(), userID, update.InfoHash)
		if err == nil && existing != nil {
			return c.Status(fiber.StatusOK).JSON(existing)
		}
		return c.Status(fiber.StatusConflict).JSON(models.ErrorResponse{
			Error: "torrent already exists",
			Code:  "TORRENT_EXISTS",
		})
	}

	// Save to database
	t := &models.Torrent{
		ID:        torrentID,
		UserID:    userID,
		InfoHash:  update.InfoHash,
		Name:      update.Name,
		Status:    update.Status,
		TotalSize: update.TotalSize,
	}

	if err := h.db.CreateTorrent(c.Context(), t); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to save torrent",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(t)
}

// ListTorrents returns all torrents for the authenticated user
func (h *TorrentHandler) ListTorrents(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	torrents, total, err := h.db.GetTorrentsByUser(c.Context(), userID, pageSize, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to fetch torrents",
		})
	}

	// Enrich with live stats from engine
	for i := range torrents {
		if status, err := h.engine.GetTorrentStatus(torrents[i].InfoHash); err == nil {
			torrents[i].DownloadSpeed = status.DownloadSpeed
			torrents[i].UploadSpeed = status.UploadSpeed
			torrents[i].Progress = status.Progress
			torrents[i].Peers = status.Peers
			torrents[i].Seeds = status.Seeds
			torrents[i].DownloadedSize = status.Downloaded
			if len(status.Files) > 0 {
				torrents[i].Files = status.Files
			}
			if status.Name != "" && status.Name != "Fetching metadata..." {
				torrents[i].Name = status.Name
			}
			if status.Status != "" && status.Status != "exists" {
				torrents[i].Status = status.Status
			}
		}
	}

	return c.JSON(models.TorrentListResponse{
		Torrents:   torrents,
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
	})
}

// GetTorrent returns details for a specific torrent
func (h *TorrentHandler) GetTorrent(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	torrentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid torrent ID",
		})
	}

	t, err := h.db.GetTorrent(c.Context(), torrentID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to fetch torrent",
		})
	}
	if t == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "torrent not found",
		})
	}

	// Check ownership (unless admin)
	role := middleware.GetUserRole(c)
	if t.UserID != userID && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "access denied",
		})
	}

	// Enrich with live stats
	if status, err := h.engine.GetTorrentStatus(t.InfoHash); err == nil {
		t.DownloadSpeed = status.DownloadSpeed
		t.UploadSpeed = status.UploadSpeed
		t.Progress = status.Progress
		t.Peers = status.Peers
		t.Seeds = status.Seeds
		t.DownloadedSize = status.Downloaded
		t.Files = status.Files
		if status.Status != "" {
			t.Status = status.Status
		}
	}

	return c.JSON(t)
}

// DeleteTorrent removes a torrent
func (h *TorrentHandler) DeleteTorrent(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

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

	// Check ownership (unless admin)
	role := middleware.GetUserRole(c)
	if t.UserID != userID && role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "access denied",
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

// PauseTorrent pauses a torrent download
func (h *TorrentHandler) PauseTorrent(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

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

	if t.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "access denied",
		})
	}

	if err := h.engine.PauseTorrent(t.InfoHash); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to pause torrent",
		})
	}

	h.db.UpdateTorrentStatus(c.Context(), torrentID, "paused", t.Progress, t.DownloadedSize, t.UploadedSize, 0, 0, 0, 0)

	return c.JSON(models.SuccessResponse{
		Message: "torrent paused",
	})
}

// ResumeTorrent resumes a paused torrent
func (h *TorrentHandler) ResumeTorrent(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

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

	if t.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "access denied",
		})
	}

	// Check quota before resuming
	if err := h.checkQuota(c, userID); err != nil {
		return err
	}

	if err := h.engine.ResumeTorrent(t.InfoHash); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to resume torrent",
		})
	}

	h.db.UpdateTorrentStatus(c.Context(), torrentID, "downloading", t.Progress, t.DownloadedSize, t.UploadedSize, 0, 0, 0, 0)

	return c.JSON(models.SuccessResponse{
		Message: "torrent resumed",
	})
}

// CreateDownloadToken generates a secure download link
func (h *TorrentHandler) CreateDownloadToken(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(models.ErrorResponse{
			Error: "invalid user",
		})
	}

	torrentID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid torrent ID",
		})
	}

	type TokenRequest struct {
		FilePath string `json:"file_path"`
		UseZip   bool   `json:"use_zip"`
	}

	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "invalid request body",
		})
	}

	t, err := h.db.GetTorrent(c.Context(), torrentID)
	if err != nil || t == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "torrent not found",
		})
	}

	if t.UserID != userID {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "access denied",
		})
	}

	// Generate token
	token, err := auth.GenerateDownloadToken()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to generate token",
		})
	}

	// Determine file path - use zip if available and requested or if multiple files
	filePath := req.FilePath
	if req.UseZip && t.ZipPath != nil && *t.ZipPath != "" {
		filePath = *t.ZipPath
	}

	// Save token (expires in 24 hours, max 10 downloads)
	if err := h.db.CreateDownloadToken(c.Context(), torrentID, filePath, token, 10, 24*time.Hour); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to save token",
		})
	}

	downloadURL := fmt.Sprintf("/api/v1/download/%s", token)

	return c.JSON(fiber.Map{
		"token":        token,
		"download_url": downloadURL,
		"expires_in":   24 * 60 * 60,
		"is_zip":       req.UseZip && t.ZipPath != nil && *t.ZipPath != "",
	})
}

// Download serves a file using a download token
func (h *TorrentHandler) Download(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Error: "missing token",
		})
	}

	// Look up token
	dt, err := h.db.GetDownloadToken(c.Context(), token)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "database error",
		})
	}
	if dt == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "invalid or expired token",
		})
	}

	// Check expiry
	if time.Now().After(dt.ExpiresAt) {
		return c.Status(fiber.StatusGone).JSON(models.ErrorResponse{
			Error: "token expired",
		})
	}

	// Check download count
	if dt.DownloadCount >= dt.MaxDownloads {
		return c.Status(fiber.StatusGone).JSON(models.ErrorResponse{
			Error: "download limit exceeded",
		})
	}

	// Get torrent
	t, err := h.db.GetTorrent(c.Context(), dt.TorrentID)
	if err != nil || t == nil {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "torrent not found",
		})
	}

	// Increment download count
	h.db.IncrementDownloadCount(c.Context(), token)

	// Set headers
	filename := dt.FilePath
	if idx := strings.LastIndex(filename, "/"); idx >= 0 {
		filename = filename[idx+1:]
	}

	// Try to get file reader from engine first
	reader, size, err := h.engine.GetFileReader(t.InfoHash, dt.FilePath)
	if err == nil {
		// Log usage
		h.db.LogUsage(c.Context(), t.UserID, "download_started", size, dt.FilePath)

		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Set("Content-Type", "application/octet-stream")
		c.Set("Content-Length", strconv.FormatInt(size, 10))
		c.Set("Accept-Ranges", "bytes")

		// Handle range requests for streaming
		rangeHeader := c.Get("Range")
		if rangeHeader != "" {
			return h.handleRangeRequest(c, reader, size, rangeHeader)
		}

		// Stream the file
		c.Status(fiber.StatusOK)
		_, err = io.Copy(c.Response().BodyWriter(), reader)
		return err
	}

	// Fall back to serving from disk
	downloadDir := h.engine.GetDownloadDir()
	filePath := filepath.Join(downloadDir, dt.FilePath)
	
	// Security check - prevent path traversal
	if !strings.HasPrefix(filepath.Clean(filePath), filepath.Clean(downloadDir)) {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "invalid file path",
		})
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
			Error: "file not found on disk",
		})
	}

	// Log usage
	fileInfo, _ := os.Stat(filePath)
	if fileInfo != nil {
		h.db.LogUsage(c.Context(), t.UserID, "download_started", fileInfo.Size(), dt.FilePath)
	}

	c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return c.SendFile(filePath)
}

func (h *TorrentHandler) handleRangeRequest(c *fiber.Ctx, reader io.ReadSeeker, size int64, rangeHeader string) error {
	// Parse range header: "bytes=start-end"
	rangeHeader = strings.TrimPrefix(rangeHeader, "bytes=")
	parts := strings.Split(rangeHeader, "-")
	if len(parts) != 2 {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
	}

	start, _ := strconv.ParseInt(parts[0], 10, 64)
	end := size - 1
	if parts[1] != "" {
		end, _ = strconv.ParseInt(parts[1], 10, 64)
	}

	if start > end || start < 0 || end >= size {
		return c.Status(fiber.StatusRequestedRangeNotSatisfiable).SendString("Invalid range")
	}

	// Seek to start
	if _, err := reader.Seek(start, io.SeekStart); err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Seek failed")
	}

	length := end - start + 1

	c.Status(fiber.StatusPartialContent)
	c.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))
	c.Set("Content-Length", strconv.FormatInt(length, 10))

	_, err := io.CopyN(c.Response().BodyWriter(), reader, length)
	return err
}

func (h *TorrentHandler) checkQuota(c *fiber.Ctx, userID uuid.UUID) error {
	// Get subscription
	sub, err := h.db.GetSubscription(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Error: "failed to check subscription",
		})
	}

	limits := models.Plans["free"]
	if sub != nil {
		if planLimits, ok := models.Plans[sub.Plan]; ok {
			limits = planLimits
		}
	}

	// Check concurrent limit
	activeCount, _ := h.db.CountActiveTorrents(c.Context(), userID)
	if activeCount >= limits.ConcurrentLimit {
		return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
			Error: "concurrent download limit reached",
			Code:  "CONCURRENT_LIMIT",
		})
	}

	// Check monthly bandwidth (if not unlimited)
	if limits.DownloadLimitGB > 0 {
		monthlyUsage, _ := h.db.GetMonthlyUsage(c.Context(), userID)
		limitBytes := int64(limits.DownloadLimitGB) * 1024 * 1024 * 1024
		if monthlyUsage >= limitBytes {
			return c.Status(fiber.StatusForbidden).JSON(models.ErrorResponse{
				Error: "monthly download limit reached",
				Code:  "BANDWIDTH_LIMIT",
			})
		}
	}

	return nil
}
