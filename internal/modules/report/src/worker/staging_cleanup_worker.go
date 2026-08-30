package worker

import (
	"context"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/shared/client"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// StagingCleanupWorker periodically sweeps orphan staging photos. Anything still
// under temp_reports/ past the TTL is garbage by definition: a submitted report
// promotes its photo out of staging during CreateReport.
type StagingCleanupWorker struct {
	Log        *logrus.Logger
	Cloudinary *client.CloudinaryClient
	TTL        time.Duration
	Cron       *cron.Cron
}

func NewStagingCleanupWorker(log *logrus.Logger, cloudinary *client.CloudinaryClient, ttl time.Duration) *StagingCleanupWorker {
	return &StagingCleanupWorker{
		Log:        log,
		Cloudinary: cloudinary,
		TTL:        ttl,
		Cron:       cron.New(cron.WithSeconds()),
	}
}

// StartScheduler runs cleanup immediately, then every hour.
func (w *StagingCleanupWorker) StartScheduler() error {
	go func() {
		w.Log.Info("Starting initial staging cleanup run...")
		w.RunCleanup()
	}()

	if _, err := w.Cron.AddFunc("0 0 * * * *", func() {
		w.Log.Info("Starting scheduled staging cleanup...")
		w.RunCleanup()
	}); err != nil {
		w.Log.Warnf("Failed to schedule staging cleanup: %+v", err)
		return err
	}

	w.Cron.Start()
	w.Log.Info("Staging cleanup scheduler started")
	return nil
}

// RunCleanup lists every staged asset and deletes those older than the TTL.
// No DB join is needed: promotion already moves kept photos out of staging.
func (w *StagingCleanupWorker) RunCleanup() {
	if w.Cloudinary == nil || w.Cloudinary.Cloudinary == nil {
		w.Log.Warn("Cloudinary is not configured, skipping staging cleanup")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	assets, err := w.Cloudinary.ListByPrefix(ctx, client.StagingFolder+"/")
	if err != nil {
		w.Log.Warnf("Failed to list staging assets: %+v", err)
		return
	}

	cutoff := time.Now().Add(-w.TTL)
	var expired []string
	for _, asset := range assets {
		if asset.CreatedAt.Before(cutoff) {
			expired = append(expired, asset.PublicID)
		}
	}

	if len(expired) == 0 {
		w.Log.Info("No expired staging assets to clean up")
		return
	}

	deleted, err := w.Cloudinary.DeleteAssets(ctx, expired)
	if err != nil {
		w.Log.Warnf("Failed to delete expired staging assets: %+v", err)
		return
	}

	w.Log.Infof("Staging cleanup deleted %d of %d expired assets", deleted, len(expired))
}
