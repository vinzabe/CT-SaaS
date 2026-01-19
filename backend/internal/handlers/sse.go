package handlers

import (
	"bufio"
	"encoding/json"
	"fmt"
	"time"

	"github.com/freetorrent/freetorrent/internal/middleware"
	"github.com/freetorrent/freetorrent/internal/torrent"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type SSEHandler struct {
	engine *torrent.Engine
}

func NewSSEHandler(engine *torrent.Engine) *SSEHandler {
	return &SSEHandler{
		engine: engine,
	}
}

// Events streams real-time torrent updates via Server-Sent Events
func (h *SSEHandler) Events(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
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
	role := middleware.GetUserRole(c)
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
