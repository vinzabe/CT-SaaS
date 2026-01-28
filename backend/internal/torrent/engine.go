package torrent

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/freetorrent/freetorrent/internal/config"
	"github.com/freetorrent/freetorrent/internal/models"
	"github.com/google/uuid"
)

// Engine manages the torrent client and downloads
type Engine struct {
	client    *torrent.Client
	cfg       *config.Config
	torrents  map[string]*ManagedTorrent // keyed by info hash
	mu        sync.RWMutex
	updateCh  chan TorrentUpdate
	closeCh   chan struct{}
}

// ManagedTorrent wraps a torrent with metadata
type ManagedTorrent struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Torrent    *torrent.Torrent
	AddedAt    time.Time
	lastUpdate time.Time
}

// TorrentUpdate represents a status update for a torrent
type TorrentUpdate struct {
	ID             uuid.UUID
	InfoHash       string
	Status         string
	Progress       float64
	Downloaded     int64
	Uploaded       int64
	DownloadSpeed  float64
	UploadSpeed    float64
	Peers          int
	Seeds          int
	Name           string
	TotalSize      int64
	Files          []models.TorrentFile
	Error          string
}

// NewEngine creates a new torrent engine
func NewEngine(cfg *config.Config) (*Engine, error) {
	// Ensure download directory exists
	if err := os.MkdirAll(cfg.DownloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %w", err)
	}

	clientCfg := torrent.NewDefaultClientConfig()
	clientCfg.DataDir = cfg.DownloadDir
	clientCfg.ListenPort = cfg.DefaultPort
	clientCfg.Seed = false      // Disable seeding by default
	clientCfg.NoUpload = true   // No uploading
	clientCfg.DisableIPv6 = false
	clientCfg.Debug = false

	// Performance tuning
	clientCfg.EstablishedConnsPerTorrent = 50
	clientCfg.HalfOpenConnsPerTorrent = 25
	clientCfg.TorrentPeersHighWater = 500
	clientCfg.TorrentPeersLowWater = 50

	client, err := torrent.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create torrent client: %w", err)
	}

	engine := &Engine{
		client:   client,
		cfg:      cfg,
		torrents: make(map[string]*ManagedTorrent),
		updateCh: make(chan TorrentUpdate, 100),
		closeCh:  make(chan struct{}),
	}

	// Start update loop
	go engine.updateLoop()

	return engine, nil
}

// Close shuts down the engine
func (e *Engine) Close() {
	close(e.closeCh)
	e.client.Close()
}

// Updates returns the channel for torrent updates
func (e *Engine) Updates() <-chan TorrentUpdate {
	return e.updateCh
}

// AddMagnet adds a torrent from a magnet link
func (e *Engine) AddMagnet(ctx context.Context, id, userID uuid.UUID, magnetURI string) (*TorrentUpdate, error) {
	t, err := e.client.AddMagnet(magnetURI)
	if err != nil {
		return nil, fmt.Errorf("failed to add magnet: %w", err)
	}

	infoHash := t.InfoHash().HexString()

	// Check if already exists
	e.mu.Lock()
	if existing, ok := e.torrents[infoHash]; ok {
		e.mu.Unlock()
		return &TorrentUpdate{
			ID:       existing.ID,
			InfoHash: infoHash,
			Status:   "exists",
		}, nil
	}

	e.torrents[infoHash] = &ManagedTorrent{
		ID:      id,
		UserID:  userID,
		Torrent: t,
		AddedAt: time.Now(),
	}
	e.mu.Unlock()

	// Wait for info in background
	go func() {
		select {
		case <-t.GotInfo():
			// Start download
			t.DownloadAll()
			
			// Send initial update with metadata
			e.sendUpdate(infoHash)
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Minute):
			// Timeout waiting for metadata
			e.mu.Lock()
			if mt, ok := e.torrents[infoHash]; ok {
				e.updateCh <- TorrentUpdate{
					ID:       mt.ID,
					InfoHash: infoHash,
					Status:   "failed",
					Error:    "timeout waiting for torrent metadata",
				}
			}
			e.mu.Unlock()
		}
	}()

	return &TorrentUpdate{
		ID:       id,
		InfoHash: infoHash,
		Status:   "pending",
	}, nil
}

// AddTorrentFile adds a torrent from a .torrent file
func (e *Engine) AddTorrentFile(ctx context.Context, id, userID uuid.UUID, reader io.Reader) (*TorrentUpdate, error) {
	mi, err := metainfo.Load(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse torrent file: %w", err)
	}

	t, err := e.client.AddTorrent(mi)
	if err != nil {
		return nil, fmt.Errorf("failed to add torrent: %w", err)
	}

	infoHash := t.InfoHash().HexString()

	e.mu.Lock()
	if existing, ok := e.torrents[infoHash]; ok {
		e.mu.Unlock()
		return &TorrentUpdate{
			ID:       existing.ID,
			InfoHash: infoHash,
			Status:   "exists",
		}, nil
	}

	e.torrents[infoHash] = &ManagedTorrent{
		ID:      id,
		UserID:  userID,
		Torrent: t,
		AddedAt: time.Now(),
	}
	e.mu.Unlock()

	// Start download immediately since we have the info
	t.DownloadAll()

	// Send initial update
	e.sendUpdate(infoHash)

	return &TorrentUpdate{
		ID:       id,
		InfoHash: infoHash,
		Status:   "downloading",
	}, nil
}

// RemoveTorrent stops and removes a torrent
func (e *Engine) RemoveTorrent(infoHash string, deleteFiles bool) error {
	e.mu.Lock()
	mt, ok := e.torrents[infoHash]
	if !ok {
		e.mu.Unlock()
		return fmt.Errorf("torrent not found")
	}

	// Get file paths before dropping
	var filePaths []string
	if deleteFiles && mt.Torrent.Info() != nil {
		for _, f := range mt.Torrent.Files() {
			filePaths = append(filePaths, filepath.Join(e.cfg.DownloadDir, f.Path()))
		}
	}

	mt.Torrent.Drop()
	delete(e.torrents, infoHash)
	e.mu.Unlock()

	// Delete files if requested
	if deleteFiles {
		for _, path := range filePaths {
			os.Remove(path)
		}
		// Try to remove parent directories if empty
		if len(filePaths) > 0 {
			dir := filepath.Dir(filePaths[0])
			os.Remove(dir) // Will fail if not empty, which is fine
		}
	}

	return nil
}

// PauseTorrent pauses a torrent download
func (e *Engine) PauseTorrent(infoHash string) error {
	e.mu.RLock()
	mt, ok := e.torrents[infoHash]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("torrent not found")
	}

	mt.Torrent.SetMaxEstablishedConns(0)
	return nil
}

// ResumeTorrent resumes a paused torrent
func (e *Engine) ResumeTorrent(infoHash string) error {
	e.mu.RLock()
	mt, ok := e.torrents[infoHash]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("torrent not found")
	}

	mt.Torrent.SetMaxEstablishedConns(50)
	mt.Torrent.DownloadAll()
	return nil
}

// GetTorrentStatus returns current status of a torrent
func (e *Engine) GetTorrentStatus(infoHash string) (*TorrentUpdate, error) {
	e.mu.RLock()
	mt, ok := e.torrents[infoHash]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("torrent not found")
	}

	return e.buildUpdate(infoHash, mt), nil
}

// GetFilePath returns the absolute path to a torrent file
func (e *Engine) GetFilePath(infoHash, relativePath string) (string, error) {
	e.mu.RLock()
	mt, ok := e.torrents[infoHash]
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("torrent not found")
	}

	if mt.Torrent.Info() == nil {
		return "", fmt.Errorf("torrent metadata not available")
	}

	// Find the file
	for _, f := range mt.Torrent.Files() {
		if f.Path() == relativePath {
			fullPath := filepath.Join(e.cfg.DownloadDir, f.Path())
			// Security check - prevent path traversal
			if !strings.HasPrefix(fullPath, e.cfg.DownloadDir) {
				return "", fmt.Errorf("invalid file path")
			}
			return fullPath, nil
		}
	}

	return "", fmt.Errorf("file not found in torrent")
}

// GetFileReader returns a reader for streaming a file
func (e *Engine) GetFileReader(infoHash, relativePath string) (io.ReadSeeker, int64, error) {
	e.mu.RLock()
	mt, ok := e.torrents[infoHash]
	e.mu.RUnlock()

	if !ok {
		return nil, 0, fmt.Errorf("torrent not found")
	}

	if mt.Torrent.Info() == nil {
		return nil, 0, fmt.Errorf("torrent metadata not available")
	}

	for _, f := range mt.Torrent.Files() {
		if f.Path() == relativePath {
			reader := f.NewReader()
			reader.SetReadahead(10 * 1024 * 1024) // 10MB readahead for streaming
			reader.SetResponsive()
			return reader, f.Length(), nil
		}
	}

	return nil, 0, fmt.Errorf("file not found")
}

// updateLoop periodically updates torrent statuses
func (e *Engine) updateLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.closeCh:
			return
		case <-ticker.C:
			e.mu.RLock()
			for infoHash := range e.torrents {
				e.sendUpdate(infoHash)
			}
			e.mu.RUnlock()
		}
	}
}

func (e *Engine) sendUpdate(infoHash string) {
	e.mu.RLock()
	mt, ok := e.torrents[infoHash]
	e.mu.RUnlock()

	if !ok {
		return
	}

	update := e.buildUpdate(infoHash, mt)
	
	select {
	case e.updateCh <- *update:
	default:
		// Channel full, skip update
	}
}

func (e *Engine) buildUpdate(infoHash string, mt *ManagedTorrent) *TorrentUpdate {
	t := mt.Torrent
	
	update := &TorrentUpdate{
		ID:       mt.ID,
		InfoHash: infoHash,
	}

	// Check if we have metadata
	if t.Info() == nil {
		update.Status = "pending"
		update.Name = "Fetching metadata..."
		return update
	}

	// Get stats
	stats := t.Stats()
	bytesCompleted := t.BytesCompleted()
	totalLength := t.Length()

	update.Name = t.Name()
	update.TotalSize = totalLength
	update.Downloaded = bytesCompleted
	update.Uploaded = stats.BytesWrittenData.Int64()
	update.Peers = stats.ActivePeers
	update.Seeds = stats.ConnectedSeeders

	// Calculate progress
	if totalLength > 0 {
		update.Progress = float64(bytesCompleted) / float64(totalLength) * 100
	}

	// Calculate speeds (bytes per second)
	now := time.Now()
	if !mt.lastUpdate.IsZero() {
		elapsed := now.Sub(mt.lastUpdate).Seconds()
		if elapsed > 0 {
			// This is a simplified calculation - in production you'd track previous values
			update.DownloadSpeed = float64(stats.BytesReadData.Int64()) / elapsed
			update.UploadSpeed = float64(stats.BytesWrittenData.Int64()) / elapsed
		}
	}
	mt.lastUpdate = now

	// Determine status
	if bytesCompleted >= totalLength {
		update.Status = "completed"
	} else if t.Seeding() {
		update.Status = "seeding"
	} else if stats.ActivePeers > 0 {
		update.Status = "downloading"
	} else {
		update.Status = "stalled"
	}

	// Get file list
	for _, f := range t.Files() {
		completed := f.BytesCompleted()
		length := f.Length()
		progress := float64(0)
		if length > 0 {
			progress = float64(completed) / float64(length) * 100
		}
		
		update.Files = append(update.Files, models.TorrentFile{
			Path:     f.Path(),
			Size:     length,
			Progress: progress,
			Priority: 2, // normal
		})
	}

	return update
}

// GetActiveTorrents returns all active torrents
func (e *Engine) GetActiveTorrents() []TorrentUpdate {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var updates []TorrentUpdate
	for infoHash, mt := range e.torrents {
		updates = append(updates, *e.buildUpdate(infoHash, mt))
	}
	return updates
}

// GetUserTorrents returns torrents for a specific user
func (e *Engine) GetUserTorrents(userID uuid.UUID) []TorrentUpdate {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var updates []TorrentUpdate
	for infoHash, mt := range e.torrents {
		if mt.UserID == userID {
			updates = append(updates, *e.buildUpdate(infoHash, mt))
		}
	}
	return updates
}

// IsInfoHashActive checks if a torrent is currently managed
func (e *Engine) IsInfoHashActive(infoHash string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.torrents[infoHash]
	return ok
}

// ReloadTorrent reloads a torrent from magnet URI (used for server restarts)
func (e *Engine) ReloadTorrent(ctx context.Context, id, userID uuid.UUID, magnetURI, infoHash string, status string) error {
	// Skip if already loaded
	e.mu.RLock()
	if _, ok := e.torrents[infoHash]; ok {
		e.mu.RUnlock()
		return nil
	}
	e.mu.RUnlock()

	// Skip failed or cancelled torrents
	if status == "failed" || status == "cancelled" {
		return nil
	}

	var t *torrent.Torrent
	var err error

	if magnetURI != "" {
		t, err = e.client.AddMagnet(magnetURI)
	} else {
		// Try to add by info hash directly
		var ih metainfo.Hash
		if err := ih.FromHexString(infoHash); err != nil {
			return fmt.Errorf("invalid info hash: %w", err)
		}
		t, _ = e.client.AddTorrentInfoHash(ih)
	}

	if err != nil {
		return err
	}

	e.mu.Lock()
	e.torrents[infoHash] = &ManagedTorrent{
		ID:      id,
		UserID:  userID,
		Torrent: t,
		AddedAt: time.Now(),
	}
	e.mu.Unlock()

	// Start download in background if not completed
	if status != "completed" && status != "seeding" {
		go func() {
			select {
			case <-t.GotInfo():
				t.DownloadAll()
				e.sendUpdate(infoHash)
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Minute):
				// Timeout
			}
		}()
	}

	return nil
}

// GetDownloadDir returns the download directory path
func (e *Engine) GetDownloadDir() string {
	return e.cfg.DownloadDir
}
