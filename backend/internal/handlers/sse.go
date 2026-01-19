package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/freetorrent/freetorrent/internal/auth"
	"github.com/freetorrent/freetorrent/internal/middleware"
	"github.com/freetorrent/freetorrent/internal/torrent"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type SSEHandler struct {
	engine      *torrent.Engine
	authService *auth.AuthService
}

func NewSSEHandler(engine *torrent.Engine, authService *auth.AuthService) *SSEHandler {
	return &SSEHandler{
		engine:      engine,
		authService: authService,
	}
}

// getSSEUserID extracts user ID from either Authorization header or token query param
// This allows SSE to work with both standard auth middleware and browser EventSource
func (h *SSEHandler) getSSEUserID(c *fiber.Ctx) (uuid.UUID, string, error) {
	// First try standard middleware (Authorization header)
	userID, err := middleware.GetUserID(c)
	if err == nil {
		role := middleware.GetUserRole(c)
		return userID, role, nil
	}

	// Fall back to token query parameter for EventSource compatibility
	token := c.Query("token")
	if token == "" {
		// Also check Authorization header directly (in case middleware didn't run)
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		return uuid.Nil, "", fmt.Errorf("no authentication token")
	}

	// Validate the token
	claims, err := h.authService.ValidateAccessToken(token)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid token: %w", err)
	}

	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("invalid user ID in token")
	}

	return uid, claims.Role, nil
}

// Events streams real-time torrent updates via Server-Sent Events
func (h *SSEHandler) Events(c *fiber.Ctx) error {
	userID, _, err := h.getSSEUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")
	c.Set("Access-Control-Allow-Origin", "*")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		// Send initial connection message
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		w.Flush()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		// Keep connection alive for max 30 minutes
		timeout := time.After(30 * time.Minute)

		for {
			select {
			case <-timeout:
				fmt.Fprintf(w, "event: timeout\ndata: {\"message\":\"connection timeout, please reconnect\"}\n\n")
				w.Flush()
				return

			case <-ticker.C:
				// Get user's torrents
				torrents := h.engine.GetUserTorrents(userID)
				
				if len(torrents) > 0 {
					data, err := json.Marshal(torrents)
					if err != nil {
						continue
					}
					
					fmt.Fprintf(w, "event: torrents\ndata: %s\n\n", data)
					if err := w.Flush(); err != nil {
						// Client disconnected
						return
					}
				}

				// Send heartbeat
				fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":%d}\n\n", time.Now().Unix())
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}

// EventsAll streams all torrent updates (admin only)
func (h *SSEHandler) EventsAll(c *fiber.Ctx) error {
	_, role, err := h.getSSEUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	if role != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "admin access required",
		})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		w.Flush()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		timeout := time.After(30 * time.Minute)

		for {
			select {
			case <-timeout:
				fmt.Fprintf(w, "event: timeout\ndata: {\"message\":\"timeout\"}\n\n")
				w.Flush()
				return

			case <-ticker.C:
				torrents := h.engine.GetActiveTorrents()
				
				if len(torrents) > 0 {
					data, err := json.Marshal(torrents)
					if err != nil {
						continue
					}
					
					fmt.Fprintf(w, "event: torrents\ndata: %s\n\n", data)
					if err := w.Flush(); err != nil {
						return
					}
				}

				fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":%d}\n\n", time.Now().Unix())
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}
