package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID               uuid.UUID  `json:"id"`
	Email            string     `json:"email"`
	PasswordHash     string     `json:"-"`
	Role             string     `json:"role"` // user, premium, admin
	StripeCustomerID *string    `json:"stripe_customer_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// Subscription represents a user's subscription plan
type Subscription struct {
	ID                   uuid.UUID  `json:"id"`
	UserID               uuid.UUID  `json:"user_id"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id,omitempty"`
	Plan                 string     `json:"plan"` // free, starter, pro, unlimited
	Status               string     `json:"status"` // active, past_due, canceled, trialing
	CurrentPeriodEnd     *time.Time `json:"current_period_end,omitempty"`
	DownloadLimitGB      int        `json:"download_limit_gb"`
	ConcurrentLimit      int        `json:"concurrent_limit"`
	RetentionDays        int        `json:"retention_days"`
	CreatedAt            time.Time  `json:"created_at"`
}

// Torrent represents a torrent download
type Torrent struct {
	ID             uuid.UUID        `json:"id"`
	UserID         uuid.UUID        `json:"user_id"`
	InfoHash       string           `json:"info_hash"`
	Name           string           `json:"name"`
	MagnetURI      string           `json:"magnet_uri,omitempty"`
	Status         string           `json:"status"` // pending, downloading, seeding, completed, failed, paused
	TotalSize      int64            `json:"total_size"`
	DownloadedSize int64            `json:"downloaded_size"`
	UploadedSize   int64            `json:"uploaded_size"`
	DownloadSpeed  float64          `json:"download_speed"`
	UploadSpeed    float64          `json:"upload_speed"`
	Progress       float64          `json:"progress"`
	Peers          int              `json:"peers"`
	Seeds          int              `json:"seeds"`
	Files          []TorrentFile    `json:"files,omitempty"`
	ZipPath        *string          `json:"zip_path,omitempty"`
	ZipSize        int64            `json:"zip_size,omitempty"`
	ErrorMessage   *string          `json:"error_message,omitempty"`
	StartedAt      *time.Time       `json:"started_at,omitempty"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
}

// TorrentFile represents a file within a torrent
type TorrentFile struct {
	Path     string  `json:"path"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
	Priority int     `json:"priority"` // 0=skip, 1=low, 2=normal, 3=high
}

// DownloadToken represents a secure download token
type DownloadToken struct {
	ID            uuid.UUID  `json:"id"`
	TorrentID     uuid.UUID  `json:"torrent_id"`
	FilePath      string     `json:"file_path"`
	Token         string     `json:"token"`
	ExpiresAt     time.Time  `json:"expires_at"`
	DownloadCount int        `json:"download_count"`
	MaxDownloads  int        `json:"max_downloads"`
	CreatedAt     time.Time  `json:"created_at"`
}

// UsageLog represents usage tracking
type UsageLog struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	Action           string     `json:"action"`
	BytesTransferred int64      `json:"bytes_transferred"`
	Metadata         string     `json:"metadata,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// Plan constants
type PlanLimits struct {
	DownloadLimitGB int
	ConcurrentLimit int
	RetentionDays   int
	PriceMonthly    int // cents
}

var Plans = map[string]PlanLimits{
	"free":      {DownloadLimitGB: 2, ConcurrentLimit: 1, RetentionDays: 1, PriceMonthly: 0},
	"starter":   {DownloadLimitGB: 50, ConcurrentLimit: 3, RetentionDays: 7, PriceMonthly: 500},
	"pro":       {DownloadLimitGB: 500, ConcurrentLimit: 10, RetentionDays: 30, PriceMonthly: 1500},
	"unlimited": {DownloadLimitGB: -1, ConcurrentLimit: 25, RetentionDays: 90, PriceMonthly: 3000},
}

// API Request/Response types
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         *User  `json:"user"`
}

type AddTorrentRequest struct {
	MagnetURI  string `json:"magnet_uri,omitempty"`
	TorrentURL string `json:"torrent_url,omitempty"`
}

type TorrentListResponse struct {
	Torrents   []Torrent `json:"torrents"`
	TotalCount int       `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type UsageStats struct {
	UsedGB          float64 `json:"used_gb"`
	LimitGB         int     `json:"limit_gb"`
	ActiveTorrents  int     `json:"active_torrents"`
	ConcurrentLimit int     `json:"concurrent_limit"`
	Plan            string  `json:"plan"`
}
