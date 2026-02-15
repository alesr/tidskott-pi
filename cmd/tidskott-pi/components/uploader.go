package components

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/alesr/tidskott-pi/internal/pkg/errutil"
	uploader "github.com/alesr/tidskott-uploader/pkg/uploader"
)

type Uploader struct {
	logger   *slog.Logger
	uploader *uploader.Uploader
	endpoint string
}

func NewUploader(
	logger *slog.Logger,
	endpoint string,
	maxRetries, maxConcurrent int,
	deleteAfterUpload, authEnabled bool,
	authEndpoint, clientID, clientSecret string,
) (*Uploader, error) {
	uploadConfig := uploader.DefaultConfig()
	uploadConfig.Endpoint = endpoint
	uploadConfig.MaxRetries = maxRetries
	uploadConfig.MaxConcurrent = maxConcurrent
	uploadConfig.DeleteAfterUpload = deleteAfterUpload

	if authEnabled {
		baseURL := endpoint
		if before, ok := strings.CutSuffix(baseURL, "/upload"); ok {
			baseURL = before
		}

		uploadConfig.AuthEnabled = true
		uploadConfig.AuthEndpoint = baseURL + authEndpoint
		uploadConfig.ClientID = clientID
		uploadConfig.ClientSecret = clientSecret
	}

	up, err := uploader.New(uploadConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("could notcreate uploader: %w", err)
	}
	return &Uploader{
		uploader: up,
		logger:   logger,
		endpoint: endpoint,
	}, nil
}

func (u *Uploader) Start() error {
	if err := u.uploader.Start(); err != nil {
		if errutil.IsConnRefused(err) {
			u.logger.Error(
				"Failed to connect to the server",
				"endpoint", u.endpoint,
				"error", err,
				"hint", "Make sure the external hub server is running at the specified endpoint",
			)
			return fmt.Errorf("could not connect to server: %w", err)
		}
		return fmt.Errorf("could notstart uploader: %w", err)
	}
	return nil
}

func (u *Uploader) Stop() error {
	return u.uploader.Stop()
}

func (u *Uploader) QueueSnapshot(snapshot *uploader.Snapshot) error {
	return u.uploader.QueueSnapshot(snapshot)
}

func (u *Uploader) Results() <-chan uploader.UploadResult { return u.uploader.Results() }
