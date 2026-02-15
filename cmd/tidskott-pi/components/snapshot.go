package components

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/alesr/tidskott-core/pkg/buffer"
	"github.com/alesr/tidskott-pi/internal/pkg/errutil"
	"github.com/alesr/tidskott-uploader/pkg/uploader"
)

type SnapshotHandler struct {
	buffer           *VideoBuffer
	uploader         *Uploader
	snapshotInterval time.Duration
	deviceID         string
	deviceName       string
	authEnabled      bool
	width            int
	height           int

	logger *slog.Logger

	mu    sync.Mutex
	count int
}

func NewSnapshotHandler(
	buffer *VideoBuffer,
	uploader *Uploader,
	snapshotInterval time.Duration,
	deviceID, deviceName string,
	authEnabled bool,
	width, height int,
	logger *slog.Logger,
) *SnapshotHandler {
	return &SnapshotHandler{
		buffer:           buffer,
		uploader:         uploader,
		snapshotInterval: snapshotInterval,
		deviceID:         deviceID,
		deviceName:       deviceName,
		authEnabled:      authEnabled,
		width:            width,
		height:           height,
		logger:           logger,
	}
}

func (sh *SnapshotHandler) Start(ctx context.Context) {
	sh.startScheduler(ctx)
	sh.startProcessor(ctx)
	sh.startResultLogger(ctx)
}

func (sh *SnapshotHandler) Count() int {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	return sh.count
}

func (sh *SnapshotHandler) startScheduler(ctx context.Context) {
	if sh.snapshotInterval <= 0 {
		sh.logger.Warn("Snapshot interval disabled", "interval", sh.snapshotInterval)
		return
	}

	ticker := time.NewTicker(sh.snapshotInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sh.buffer.GetSnapshot(ctx); err != nil {
					sh.logger.Error("Failed to request snapshot", "error", err)
				} else {
					sh.logger.Debug("Snapshot requested")
				}
			}
		}
	}()
}

func (sh *SnapshotHandler) startProcessor(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case snapshot := <-sh.buffer.Snapshots():
				sh.processSnapshot(snapshot)
			}
		}
	}()
}

func (sh *SnapshotHandler) startResultLogger(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-sh.uploader.Results():
				if result.Success {
					sh.logger.Info(
						"Upload successful",
						"id", result.Snapshot.ID,
						"size_mb", fmt.Sprintf("%.2f", float64(result.Snapshot.Size)/(1024*1024)),
						"speed_kbps", fmt.Sprintf("%.2f", result.Speed/1024),
						"auth_enabled", sh.authEnabled,
					)
					return
				}

				sh.logger.Error(
					"Upload failed",
					"id", result.Snapshot.ID,
					"error", result.Error,
				)

				if result.Error != nil && errutil.IsConnRefused(result.Error) {
					sh.logger.Error(
						"Server connection lost",
						"endpoint", sh.uploader.endpoint,
						"hint", "Make sure the external hub server is running at the specified endpoint")
				}

			}
		}
	}()
}

func (sh *SnapshotHandler) processSnapshot(snapshot *buffer.Snapshot) {
	if snapshot == nil || snapshot.VideoPath == "" {
		sh.logger.Error("Empty snapshot received")
		return
	}

	fileInfo, err := os.Stat(snapshot.VideoPath)
	if err != nil {
		sh.logger.Error("Failed to get snapshot file info", "error", err, "path", snapshot.VideoPath)
		return
	}

	hash, err := calculateHash(snapshot.VideoPath)
	if err != nil {
		sh.logger.Error("Failed to calculate snapshot hash", "error", err, "path", snapshot.VideoPath)
		return
	}

	duration := int(snapshot.EndTime.Sub(snapshot.StartTime).Seconds())

	sh.mu.Lock()
	sh.count++
	count := sh.count
	sh.mu.Unlock()

	sh.logger.Info(
		"Snapshot created",
		"number", count,
		"id", snapshot.ID,
		"path", snapshot.VideoPath,
		"size_mb", fmt.Sprintf("%.2f", float64(fileInfo.Size())/(1024*1024)),
	)

	metadata := map[string]string{
		"width":        fmt.Sprintf("%d", sh.width),
		"height":       fmt.Sprintf("%d", sh.height),
		"duration":     fmt.Sprintf("%d", duration),
		"source":       "tidskott-pi",
		"snapshot_id":  snapshot.ID,
		"device_id":    sh.deviceID,
		"device_name":  sh.deviceName,
		"auth_enabled": fmt.Sprintf("%v", sh.authEnabled),
	}

	uploadSnapshot := &uploader.Snapshot{
		ID:         snapshot.ID,
		Path:       snapshot.VideoPath,
		Timestamp:  snapshot.Timestamp,
		Size:       fileInfo.Size(),
		Hash:       hash,
		Metadata:   metadata,
		Width:      sh.width,
		Height:     sh.height,
		Duration:   duration,
		DeviceID:   sh.deviceID,
		DeviceName: sh.deviceName,
	}

	if err := sh.uploader.QueueSnapshot(uploadSnapshot); err != nil {
		sh.logger.Warn("Failed to queue snapshot for upload", "error", err)
		if errutil.IsConnRefused(err) {
			sh.logger.Error(
				"Cannot connect to server",
				"endpoint", sh.uploader.endpoint,
				"hint", "Make sure the external hub server is running at the specified endpoint",
			)
		}
		return
	}
	sh.logger.Debug("Queued snapshot for upload", "id", uploadSnapshot.ID)
}

func calculateHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("could not open snapshot for hashing: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("could not hash snapshot: %w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}
