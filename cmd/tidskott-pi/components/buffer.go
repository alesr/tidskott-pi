package components

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/alesr/tidskott-core/pkg/buffer"
	"github.com/alesr/tidskott-core/pkg/interfaces"
	"github.com/alesr/tidskott-pi/pkg/camera/macos"
	"github.com/alesr/tidskott-pi/pkg/camera/raspberry"
)

type VideoBuffer struct {
	logger *slog.Logger
	buffer *buffer.Buffer
}

func NewVideoBuffer(
	logger *slog.Logger,
	windowSeconds, snapshotDuration, snapshotInterval, width, height, fps, bitrate int,
	codec string,
) (*VideoBuffer, error) {
	var cameraFactory interfaces.Factory
	if runtime.GOOS == "darwin" {
		cameraFactory = macos.NewMacOSCameraFactory(logger, "0:none")
	} else {
		cameraFactory = raspberry.NewRaspberryPiCameraFactory(logger)
	}

	bufferOpts := []buffer.Option{
		buffer.WithVideoSource(cameraFactory),
		buffer.WithWindow(time.Duration(windowSeconds)),
		buffer.WithFrameSize(width * height),
		buffer.WithFPS(fps),
		buffer.WithBitrate(bitrate),
		buffer.WithVideoCodec(codec),
		buffer.WithSnapshotDuration(time.Duration(snapshotDuration)),
		buffer.WithSnapshotInterval(time.Duration(snapshotInterval)),
	}

	vb, err := buffer.NewBuffer(logger, bufferOpts...)
	if err != nil {
		return nil, fmt.Errorf("could not create video buffer: %w", err)
	}
	return &VideoBuffer{buffer: vb, logger: logger}, nil
}

func (vb *VideoBuffer) Start(ctx context.Context) error {
	vb.logger.Info("Starting video buffer")
	if err := vb.buffer.Start(ctx); err != nil {
		return fmt.Errorf("could not start video buffer: %w", err)
	}
	vb.logger.Info("Video buffer started successfully")
	return nil
}

func (vb *VideoBuffer) Stop(ctx context.Context) error {
	vb.logger.Info("Stopping video buffer")
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := vb.buffer.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("could not stop video buffer: %w", err)
	}
	return nil
}

func (vb *VideoBuffer) Snapshots() <-chan *buffer.Snapshot    { return vb.buffer.Snapshots() }
func (vb *VideoBuffer) GetSnapshot(ctx context.Context) error { return vb.buffer.GetSnapshot(ctx) }
