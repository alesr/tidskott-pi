package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alesr/tidskott-pi/cmd/tidskott-pi/components"
)

func Run() error {
	flags, err := parseFlags()
	if err != nil {
		return fmt.Errorf("could not parse flags: %w", err)
	}

	cfg, err := loadConfig(flags.ConfigPath)
	if err != nil {
		return fmt.Errorf("could not load config: %w", err)
	}

	logger, logFile, err := setupLogger()
	if err != nil {
		return fmt.Errorf("could not set up logger: %w", err)
	}
	defer logFile.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	videoBuffer, err := components.NewVideoBuffer(
		logger,
		cfg.Buffer.WindowSeconds,
		cfg.Buffer.SnapshotDuration,
		cfg.Buffer.SnapshotInterval,
		cfg.Camera.Width,
		cfg.Camera.Height,
		cfg.Camera.FPS,
		cfg.Camera.Bitrate,
		cfg.Camera.Codec,
	)
	if err != nil {
		return fmt.Errorf("could not create video buffer: %w", err)
	}
	defer stopVideoBuffer(videoBuffer, logger)

	if err := videoBuffer.Start(ctx); err != nil {
		return fmt.Errorf("could not start video buffer: %w", err)
	}

	uploader, err := components.NewUploader(
		logger,
		cfg.Upload.Endpoint,
		cfg.Upload.MaxRetries,
		cfg.Upload.MaxConcurrent,
		cfg.Upload.DeleteAfterUpload,
		cfg.Auth.Enabled,
		cfg.Auth.Endpoint,
		cfg.Auth.ClientID,
		cfg.Auth.ClientSecret,
	)
	if err != nil {
		return fmt.Errorf("could not create uploader: %w", err)
	}
	defer stopUploader(uploader, logger)

	if err := uploader.Start(); err != nil {
		return fmt.Errorf("could not start uploader: %w", err)
	}

	snapshotHandler := components.NewSnapshotHandler(
		videoBuffer,
		uploader,
		time.Duration(cfg.Buffer.SnapshotInterval)*time.Second,
		cfg.Device.ID,
		cfg.Device.Name,
		cfg.Auth.Enabled,
		cfg.Camera.Width,
		cfg.Camera.Height,
		logger,
	)
	go snapshotHandler.Start(ctx)

	return runMainLoop(ctx, snapshotHandler, logger)
}

func runMainLoop(ctx context.Context, snapshots *components.SnapshotHandler, logger *slog.Logger) error {
	startTime := time.Now()

	<-ctx.Done()

	count := snapshots.Count()

	logger.Info(
		"Recording completed",
		"duration", time.Since(startTime).Round(time.Second),
		"snapshots", count,
	)
	return nil
}

func stopVideoBuffer(buffer *components.VideoBuffer, logger *slog.Logger) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := buffer.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping video buffer", "error", err)
	}
}

func stopUploader(uploader *components.Uploader, logger *slog.Logger) {
	if err := uploader.Stop(); err != nil {
		logger.Error("Error stopping uploader", "error", err)
	}
}
