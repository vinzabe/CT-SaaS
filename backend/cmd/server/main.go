package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/freetorrent/freetorrent/internal/auth"
	"github.com/freetorrent/freetorrent/internal/config"
	"github.com/freetorrent/freetorrent/internal/database"
	"github.com/freetorrent/freetorrent/internal/handlers"
	"github.com/freetorrent/freetorrent/internal/middleware"
	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/freetorrent/freetorrent/internal/torrent"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file if present
	godotenv.Load()

	// Load configuration
	cfg := config.Load()

	log.Printf("Starting CT-SaaS server...")
	log.Printf("Environment: %s", cfg.Environment)

	// Initialize database
	db, err := database.New(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(context.Background()); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migrations completed")

	// Initialize torrent engine
	engine, err := torrent.NewEngine(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize torrent engine: %v", err)
	}
	defer engine.Close()
	log.Println("Torrent engine initialized")

	// Start torrent update processor
	go processTorrentUpdates(db, engine, cfg)

	// Initialize auth service
	authService := auth.NewAuthService(cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, authService, cfg)
	torrentHandler := handlers.NewTorrentHandler(db, engine)
	adminHandler := handlers.NewAdminHandler(db, engine)
	sseHandler := handlers.NewSSEHandler(engine, authService)
	billingHandler := handlers.NewBillingHandler(db, cfg)

	// Initialize rate limiter (100 requests per minute)
	rateLimiter := middleware.NewRateLimiter(100, time.Minute)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "CT-SaaS",
		ServerHeader:          "CT-SaaS",
		DisableStartupMessage: cfg.Environment == "production",
		ReadTimeout:           30 * time.Second,
		WriteTimeout:          30 * time.Second,
		IdleTimeout:           120 * time.Second,
		BodyLimit:             50 * 1024 * 1024, // 50MB for torrent files
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(middleware.RequestIDMiddleware())
	app.Use(middleware.CORSMiddleware())
	
	if cfg.Environment != "production" {
		app.Use(logger.New(logger.Config{
			Format: "${time} | ${status} | ${latency} | ${ip} | ${method} | ${path}\n",
		}))
	}

	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "ct-saas",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Apply rate limiting to API routes
	api.Use(middleware.RateLimitMiddleware(rateLimiter))

	// Public auth routes
	authRoutes := api.Group("/auth")
	authRoutes.Post("/register", authHandler.Register)
	authRoutes.Post("/login", authHandler.Login)
	authRoutes.Post("/refresh", authHandler.Refresh)
	authRoutes.Post("/logout", authHandler.Logout)

	// Public download route (uses token-based auth, NOT JWT)
	api.Get("/download/:token", torrentHandler.Download)

	// Stripe webhook (no auth, uses signature verification)
	api.Post("/webhooks/stripe", billingHandler.HandleWebhook)

	// Protected routes (require authentication)
	protected := api.Group("", middleware.AuthMiddleware(authService))

	// User routes
	protected.Get("/auth/me", authHandler.Me)

	// Torrent routes
	torrents := protected.Group("/torrents")
	torrents.Post("", torrentHandler.AddTorrent)
	torrents.Post("/upload", torrentHandler.UploadTorrent)
	torrents.Get("", torrentHandler.ListTorrents)
	torrents.Get("/:id", torrentHandler.GetTorrent)
	torrents.Delete("/:id", torrentHandler.DeleteTorrent)
	torrents.Post("/:id/pause", torrentHandler.PauseTorrent)
	torrents.Post("/:id/resume", torrentHandler.ResumeTorrent)
	torrents.Post("/:id/token", torrentHandler.CreateDownloadToken)

	// SSE events
	protected.Get("/events", sseHandler.Events)

	// Billing routes
	billing := protected.Group("/subscription")
	billing.Get("", billingHandler.GetSubscription)
	billing.Post("/checkout", billingHandler.CreateCheckoutSession)
	billing.Post("/portal", billingHandler.CreatePortalSession)

	// Admin routes
	admin := protected.Group("/admin", middleware.AdminMiddleware())
	admin.Get("/users", adminHandler.ListUsers)
	admin.Get("/users/:id", adminHandler.GetUser)
	admin.Patch("/users/:id", adminHandler.UpdateUser)
	admin.Delete("/users/:id", adminHandler.DeleteUser)
	admin.Get("/torrents", adminHandler.ListAllTorrents)
	admin.Delete("/torrents/:id", adminHandler.DeleteTorrent)
	admin.Get("/stats", adminHandler.GetStats)
	admin.Post("/cleanup", adminHandler.CleanupExpired)
	admin.Get("/events", sseHandler.EventsAll)

	// Create demo admin if doesn't exist
	createDemoAdmin(db, authService)

	// Reload active torrents from database
	reloadActiveTorrents(db, engine)

	// Start cleanup job
	go cleanupJob(db, engine)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		app.Shutdown()
	}()

	// Start server
	addr := ":" + cfg.Port
	log.Printf("Server listening on %s", addr)
	if err := app.Listen(addr); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// processTorrentUpdates handles updates from the torrent engine
func processTorrentUpdates(db *database.Database, engine *torrent.Engine, cfg *config.Config) {
	for update := range engine.Updates() {
		ctx := context.Background()
		
		// Update database
		if update.Error != "" {
			db.SetTorrentError(ctx, update.ID, update.Error)
		} else if update.Progress >= 100 && update.Status == "completed" {
			// Get user's retention days
			t, err := db.GetTorrent(ctx, update.ID)
			if err == nil && t != nil {
				sub, _ := db.GetSubscription(ctx, t.UserID)
				retentionDays := 1
				if sub != nil {
					retentionDays = sub.RetentionDays
				}
				db.SetTorrentCompleted(ctx, update.ID, retentionDays)
				
				// Update name and size on completion
				if update.Name != "" && update.Name != "Fetching metadata..." {
					db.UpdateTorrentName(ctx, update.ID, update.Name, update.TotalSize)
				}
				
				// Save files to database
				if len(update.Files) > 0 {
					db.UpdateTorrentFiles(ctx, update.ID, update.Files)
					
					// Auto-zip if more than 1 file
					if len(update.Files) > 1 {
						go func(files []models.TorrentFile, name string, id uuid.UUID) {
							var filePaths []string
							for _, f := range files {
								filePaths = append(filePaths, f.Path)
							}
							
							zipPath, zipSize, err := torrent.CreateZipFromFiles(cfg.DownloadDir, name, filePaths)
							if err != nil {
								log.Printf("Failed to create zip for %s: %v", name, err)
								return
							}
							
							if err := db.UpdateTorrentZip(context.Background(), id, zipPath, zipSize); err != nil {
								log.Printf("Failed to save zip path: %v", err)
								return
							}
							
							log.Printf("Created zip archive: %s (%.2f MB)", zipPath, float64(zipSize)/1024/1024)
						}(update.Files, update.Name, update.ID)
					}
				}
				
				// Log usage
				db.LogUsage(ctx, t.UserID, "download_completed", update.TotalSize, update.Name)
			}
		} else {
			// Update status
			db.UpdateTorrentStatus(ctx, update.ID, update.Status, update.Progress,
				update.Downloaded, update.Uploaded, update.DownloadSpeed, update.UploadSpeed,
				update.Peers, update.Seeds)
			
			// Update name and size if we got metadata
			if update.Name != "" && update.Name != "Fetching metadata..." {
				db.UpdateTorrentName(ctx, update.ID, update.Name, update.TotalSize)
			}
			
			// Save files if available
			if len(update.Files) > 0 {
				db.UpdateTorrentFiles(ctx, update.ID, update.Files)
			}
		}
	}
}

// createDemoAccounts creates demo admin and demo user accounts if they don't exist
func createDemoAdmin(db *database.Database, authService *auth.AuthService) {
	ctx := context.Background()
	
	// Create admin account
	admin, err := db.GetUserByEmail(ctx, "admin@ct.saas")
	if err != nil {
		log.Printf("Error checking for admin user: %v", err)
	} else if admin == nil {
		passwordHash, err := authService.HashPassword("admin123")
		if err != nil {
			log.Printf("Failed to hash admin password: %v", err)
		} else {
			user, err := db.CreateUser(ctx, "admin@ct.saas", passwordHash)
			if err != nil {
				log.Printf("Failed to create admin user: %v", err)
			} else {
				if err := db.UpdateUserRole(ctx, user.ID, "admin"); err != nil {
					log.Printf("Failed to set admin role: %v", err)
				} else {
					log.Println("Demo admin created: admin@ct.saas / admin123")
				}
			}
		}
	} else {
		log.Println("Demo admin already exists")
	}
	
	// Create demo account (restricted - can't change password, 24hr retention)
	demo, err := db.GetUserByEmail(ctx, "demo@ct.saas")
	if err != nil {
		log.Printf("Error checking for demo user: %v", err)
	} else if demo == nil {
		passwordHash, err := authService.HashPassword("demo123")
		if err != nil {
			log.Printf("Failed to hash demo password: %v", err)
		} else {
			user, err := db.CreateUser(ctx, "demo@ct.saas", passwordHash)
			if err != nil {
				log.Printf("Failed to create demo user: %v", err)
			} else {
				// Set demo role (restricted user)
				if err := db.UpdateUserRole(ctx, user.ID, "demo"); err != nil {
					log.Printf("Failed to set demo role: %v", err)
				} else {
					log.Println("Demo user created: demo@ct.saas / demo123")
				}
			}
		}
	} else {
		log.Println("Demo user already exists")
	}
}

// reloadActiveTorrents loads active torrents from database into engine
func reloadActiveTorrents(db *database.Database, engine *torrent.Engine) {
	ctx := context.Background()
	
	// Get all non-expired, non-failed torrents
	torrents, _, err := db.GetAllTorrents(ctx, 1000, 0)
	if err != nil {
		log.Printf("Failed to load torrents from database: %v", err)
		return
	}
	
	reloaded := 0
	for _, t := range torrents {
		if t.Status == "failed" || t.Status == "cancelled" {
			continue
		}
		
		err := engine.ReloadTorrent(ctx, t.ID, t.UserID, t.MagnetURI, t.InfoHash, t.Status)
		if err != nil {
			log.Printf("Failed to reload torrent %s: %v", t.InfoHash, err)
			continue
		}
		reloaded++
	}
	
	if reloaded > 0 {
		log.Printf("Reloaded %d torrents from database", reloaded)
	}
}

// cleanupJob runs periodic cleanup tasks
func cleanupJob(db *database.Database, engine *torrent.Engine) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		
		// Get expired torrents
		expired, err := db.GetExpiredTorrents(ctx)
		if err != nil {
			log.Printf("Cleanup error: %v", err)
			continue
		}

		for _, t := range expired {
			log.Printf("Cleaning up expired torrent: %s", t.Name)
			engine.RemoveTorrent(t.InfoHash, true)
			db.DeleteTorrent(ctx, t.ID)
		}

		if len(expired) > 0 {
			log.Printf("Cleaned up %d expired torrents", len(expired))
		}
	}
}
