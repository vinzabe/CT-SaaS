package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	pool *pgxpool.Pool
}

func New(databaseURL string) (*Database, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{pool: pool}, nil
}

func (db *Database) Close() {
	db.pool.Close()
}

func (db *Database) Migrate(ctx context.Context) error {
	schema := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		email VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		role VARCHAR(50) DEFAULT 'user',
		stripe_customer_id VARCHAR(255),
		created_at TIMESTAMPTZ DEFAULT NOW(),
		updated_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS subscriptions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		stripe_subscription_id VARCHAR(255),
		plan VARCHAR(50) NOT NULL DEFAULT 'free',
		status VARCHAR(50) NOT NULL DEFAULT 'active',
		current_period_end TIMESTAMPTZ,
		download_limit_gb INT DEFAULT 2,
		concurrent_limit INT DEFAULT 1,
		retention_days INT DEFAULT 1,
		created_at TIMESTAMPTZ DEFAULT NOW(),
		UNIQUE(user_id)
	);

	CREATE TABLE IF NOT EXISTS torrents (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		info_hash VARCHAR(40) NOT NULL,
		name VARCHAR(500),
		magnet_uri TEXT,
		status VARCHAR(50) NOT NULL DEFAULT 'pending',
		total_size BIGINT DEFAULT 0,
		downloaded_size BIGINT DEFAULT 0,
		uploaded_size BIGINT DEFAULT 0,
		download_speed FLOAT DEFAULT 0,
		upload_speed FLOAT DEFAULT 0,
		progress FLOAT DEFAULT 0,
		peers INT DEFAULT 0,
		seeds INT DEFAULT 0,
		files JSONB DEFAULT '[]',
		zip_path VARCHAR(1000),
		zip_size BIGINT DEFAULT 0,
		error_message TEXT,
		started_at TIMESTAMPTZ,
		completed_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS download_tokens (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		torrent_id UUID REFERENCES torrents(id) ON DELETE CASCADE,
		file_path VARCHAR(1000),
		token VARCHAR(64) UNIQUE NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		download_count INT DEFAULT 0,
		max_downloads INT DEFAULT 10,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS usage_logs (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID REFERENCES users(id) ON DELETE SET NULL,
		action VARCHAR(50) NOT NULL,
		bytes_transferred BIGINT DEFAULT 0,
		metadata JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		token_hash VARCHAR(255) NOT NULL,
		expires_at TIMESTAMPTZ NOT NULL,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_torrents_user_status ON torrents(user_id, status);
	CREATE INDEX IF NOT EXISTS idx_torrents_info_hash ON torrents(info_hash);
	CREATE INDEX IF NOT EXISTS idx_download_tokens_token ON download_tokens(token);
	CREATE INDEX IF NOT EXISTS idx_usage_logs_user_date ON usage_logs(user_id, created_at);
	CREATE INDEX IF NOT EXISTS idx_refresh_tokens_hash ON refresh_tokens(token_hash);

	-- Migrations for existing databases
	ALTER TABLE torrents ADD COLUMN IF NOT EXISTS zip_path TEXT;
	ALTER TABLE torrents ADD COLUMN IF NOT EXISTS zip_size BIGINT DEFAULT 0;
	`

	_, err := db.pool.Exec(ctx, schema)
	return err
}

// User methods
func (db *Database) CreateUser(ctx context.Context, email, passwordHash string) (*models.User, error) {
	user := &models.User{
		ID:        uuid.New(),
		Email:     email,
		PasswordHash: passwordHash,
		Role:      "user",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	_, err := db.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID, user.Email, user.PasswordHash, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Create default free subscription
	_, err = db.pool.Exec(ctx,
		`INSERT INTO subscriptions (user_id, plan, status, download_limit_gb, concurrent_limit, retention_days)
		 VALUES ($1, 'free', 'active', 2, 1, 1)`,
		user.ID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (db *Database) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, stripe_customer_id, created_at, updated_at
		 FROM users WHERE email = $1`,
		email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.StripeCustomerID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (db *Database) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, stripe_customer_id, created_at, updated_at
		 FROM users WHERE id = $1`,
		id).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.StripeCustomerID, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (db *Database) GetAllUsers(ctx context.Context, limit, offset int) ([]models.User, int, error) {
	var total int
	err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, email, role, stripe_customer_id, created_at, updated_at
		 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Email, &user.Role, &user.StripeCustomerID, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, total, nil
}

func (db *Database) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2`,
		role, userID)
	return err
}

func (db *Database) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	return err
}

// Subscription methods
func (db *Database) GetSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	sub := &models.Subscription{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, stripe_subscription_id, plan, status, current_period_end, 
		 download_limit_gb, concurrent_limit, retention_days, created_at
		 FROM subscriptions WHERE user_id = $1`,
		userID).Scan(&sub.ID, &sub.UserID, &sub.StripeSubscriptionID, &sub.Plan, &sub.Status,
		&sub.CurrentPeriodEnd, &sub.DownloadLimitGB, &sub.ConcurrentLimit, &sub.RetentionDays, &sub.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return sub, nil
}

func (db *Database) UpdateSubscription(ctx context.Context, userID uuid.UUID, plan, status string, limits models.PlanLimits) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE subscriptions SET plan = $1, status = $2, download_limit_gb = $3, 
		 concurrent_limit = $4, retention_days = $5 WHERE user_id = $6`,
		plan, status, limits.DownloadLimitGB, limits.ConcurrentLimit, limits.RetentionDays, userID)
	return err
}

// Torrent methods
func (db *Database) CreateTorrent(ctx context.Context, t *models.Torrent) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	
	_, err := db.pool.Exec(ctx,
		`INSERT INTO torrents (id, user_id, info_hash, name, magnet_uri, status, total_size, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.UserID, t.InfoHash, t.Name, t.MagnetURI, t.Status, t.TotalSize, t.CreatedAt)
	return err
}

func (db *Database) GetTorrent(ctx context.Context, id uuid.UUID) (*models.Torrent, error) {
	t := &models.Torrent{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, info_hash, name, magnet_uri, status, total_size, downloaded_size,
		 uploaded_size, download_speed, upload_speed, progress, peers, seeds, files, 
		 zip_path, zip_size, error_message, started_at, completed_at, expires_at, created_at
		 FROM torrents WHERE id = $1`,
		id).Scan(&t.ID, &t.UserID, &t.InfoHash, &t.Name, &t.MagnetURI, &t.Status, &t.TotalSize,
		&t.DownloadedSize, &t.UploadedSize, &t.DownloadSpeed, &t.UploadSpeed, &t.Progress,
		&t.Peers, &t.Seeds, &t.Files, &t.ZipPath, &t.ZipSize, &t.ErrorMessage, 
		&t.StartedAt, &t.CompletedAt, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (db *Database) GetTorrentByInfoHash(ctx context.Context, userID uuid.UUID, infoHash string) (*models.Torrent, error) {
	t := &models.Torrent{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, user_id, info_hash, name, magnet_uri, status, total_size, downloaded_size,
		 uploaded_size, download_speed, upload_speed, progress, peers, seeds, files, 
		 zip_path, zip_size, error_message, started_at, completed_at, expires_at, created_at
		 FROM torrents WHERE user_id = $1 AND info_hash = $2 ORDER BY created_at DESC LIMIT 1`,
		userID, infoHash).Scan(&t.ID, &t.UserID, &t.InfoHash, &t.Name, &t.MagnetURI, &t.Status, &t.TotalSize,
		&t.DownloadedSize, &t.UploadedSize, &t.DownloadSpeed, &t.UploadSpeed, &t.Progress,
		&t.Peers, &t.Seeds, &t.Files, &t.ZipPath, &t.ZipSize, &t.ErrorMessage, 
		&t.StartedAt, &t.CompletedAt, &t.ExpiresAt, &t.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return t, nil
}

func (db *Database) GetTorrentsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]models.Torrent, int, error) {
	var total int
	err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM torrents WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, info_hash, name, magnet_uri, status, total_size, downloaded_size,
		 uploaded_size, download_speed, upload_speed, progress, peers, seeds, 
		 zip_path, zip_size, error_message, started_at, completed_at, expires_at, created_at
		 FROM torrents WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var torrents []models.Torrent
	for rows.Next() {
		var t models.Torrent
		if err := rows.Scan(&t.ID, &t.UserID, &t.InfoHash, &t.Name, &t.MagnetURI, &t.Status,
			&t.TotalSize, &t.DownloadedSize, &t.UploadedSize, &t.DownloadSpeed, &t.UploadSpeed,
			&t.Progress, &t.Peers, &t.Seeds, &t.ZipPath, &t.ZipSize, &t.ErrorMessage, 
			&t.StartedAt, &t.CompletedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		torrents = append(torrents, t)
	}
	return torrents, total, nil
}

func (db *Database) GetAllTorrents(ctx context.Context, limit, offset int) ([]models.Torrent, int, error) {
	var total int
	err := db.pool.QueryRow(ctx, `SELECT COUNT(*) FROM torrents`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, info_hash, name, magnet_uri, status, total_size, downloaded_size,
		 uploaded_size, download_speed, upload_speed, progress, peers, seeds, 
		 zip_path, zip_size, error_message, started_at, completed_at, expires_at, created_at
		 FROM torrents ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var torrents []models.Torrent
	for rows.Next() {
		var t models.Torrent
		if err := rows.Scan(&t.ID, &t.UserID, &t.InfoHash, &t.Name, &t.MagnetURI, &t.Status,
			&t.TotalSize, &t.DownloadedSize, &t.UploadedSize, &t.DownloadSpeed, &t.UploadSpeed,
			&t.Progress, &t.Peers, &t.Seeds, &t.ZipPath, &t.ZipSize, &t.ErrorMessage,
			&t.StartedAt, &t.CompletedAt, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, 0, err
		}
		torrents = append(torrents, t)
	}
	return torrents, total, nil
}

func (db *Database) UpdateTorrentStatus(ctx context.Context, id uuid.UUID, status string, progress float64, downloaded, uploaded int64, dlSpeed, ulSpeed float64, peers, seeds int) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE torrents SET status = $1, progress = $2, downloaded_size = $3, uploaded_size = $4,
		 download_speed = $5, upload_speed = $6, peers = $7, seeds = $8 WHERE id = $9`,
		status, progress, downloaded, uploaded, dlSpeed, ulSpeed, peers, seeds, id)
	return err
}

func (db *Database) SetTorrentCompleted(ctx context.Context, id uuid.UUID, retentionDays int) error {
	expiresAt := time.Now().AddDate(0, 0, retentionDays)
	_, err := db.pool.Exec(ctx,
		`UPDATE torrents SET status = 'completed', progress = 100, completed_at = NOW(), expires_at = $1 WHERE id = $2`,
		expiresAt, id)
	return err
}

func (db *Database) UpdateTorrentFiles(ctx context.Context, id uuid.UUID, files []models.TorrentFile) error {
	filesJSON, err := json.Marshal(files)
	if err != nil {
		return err
	}
	_, err = db.pool.Exec(ctx,
		`UPDATE torrents SET files = $1 WHERE id = $2`,
		filesJSON, id)
	return err
}

func (db *Database) UpdateTorrentName(ctx context.Context, id uuid.UUID, name string, totalSize int64) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE torrents SET name = $1, total_size = $2 WHERE id = $3`,
		name, totalSize, id)
	return err
}

func (db *Database) UpdateTorrentZip(ctx context.Context, id uuid.UUID, zipPath string, zipSize int64) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE torrents SET zip_path = $1, zip_size = $2 WHERE id = $3`,
		zipPath, zipSize, id)
	return err
}

func (db *Database) SetTorrentError(ctx context.Context, id uuid.UUID, errMsg string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE torrents SET status = 'failed', error_message = $1 WHERE id = $2`,
		errMsg, id)
	return err
}

func (db *Database) DeleteTorrent(ctx context.Context, id uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM torrents WHERE id = $1`, id)
	return err
}

func (db *Database) CountActiveTorrents(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	err := db.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM torrents WHERE user_id = $1 AND status IN ('pending', 'downloading')`,
		userID).Scan(&count)
	return count, err
}

func (db *Database) GetExpiredTorrents(ctx context.Context) ([]models.Torrent, error) {
	rows, err := db.pool.Query(ctx,
		`SELECT id, user_id, info_hash, name FROM torrents WHERE expires_at < NOW()`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []models.Torrent
	for rows.Next() {
		var t models.Torrent
		if err := rows.Scan(&t.ID, &t.UserID, &t.InfoHash, &t.Name); err != nil {
			return nil, err
		}
		torrents = append(torrents, t)
	}
	return torrents, nil
}

// Download token methods
func (db *Database) CreateDownloadToken(ctx context.Context, torrentID uuid.UUID, filePath, token string, maxDownloads int, expiresIn time.Duration) error {
	expiresAt := time.Now().Add(expiresIn)
	_, err := db.pool.Exec(ctx,
		`INSERT INTO download_tokens (torrent_id, file_path, token, expires_at, max_downloads)
		 VALUES ($1, $2, $3, $4, $5)`,
		torrentID, filePath, token, expiresAt, maxDownloads)
	return err
}

func (db *Database) GetDownloadToken(ctx context.Context, token string) (*models.DownloadToken, error) {
	dt := &models.DownloadToken{}
	err := db.pool.QueryRow(ctx,
		`SELECT id, torrent_id, file_path, token, expires_at, download_count, max_downloads, created_at
		 FROM download_tokens WHERE token = $1`,
		token).Scan(&dt.ID, &dt.TorrentID, &dt.FilePath, &dt.Token, &dt.ExpiresAt, &dt.DownloadCount, &dt.MaxDownloads, &dt.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return dt, nil
}

func (db *Database) IncrementDownloadCount(ctx context.Context, token string) error {
	_, err := db.pool.Exec(ctx,
		`UPDATE download_tokens SET download_count = download_count + 1 WHERE token = $1`,
		token)
	return err
}

// Usage logging
func (db *Database) LogUsage(ctx context.Context, userID uuid.UUID, action string, bytes int64, metadata string) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO usage_logs (user_id, action, bytes_transferred, metadata) VALUES ($1, $2, $3, $4)`,
		userID, action, bytes, metadata)
	return err
}

func (db *Database) GetMonthlyUsage(ctx context.Context, userID uuid.UUID) (int64, error) {
	var total int64
	err := db.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(bytes_transferred), 0) FROM usage_logs 
		 WHERE user_id = $1 AND action = 'download_completed'
		 AND created_at >= date_trunc('month', CURRENT_DATE)`,
		userID).Scan(&total)
	return total, err
}

// Refresh token methods
func (db *Database) SaveRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := db.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt)
	return err
}

func (db *Database) GetRefreshToken(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	var userID uuid.UUID
	err := db.pool.QueryRow(ctx,
		`SELECT user_id FROM refresh_tokens WHERE token_hash = $1 AND expires_at > NOW()`,
		tokenHash).Scan(&userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, nil
		}
		return uuid.Nil, err
	}
	return userID, nil
}

func (db *Database) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	return err
}

func (db *Database) DeleteUserRefreshTokens(ctx context.Context, userID uuid.UUID) error {
	_, err := db.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}
