package worker

import (
	"context"
	"time"

	"github.com/arttVinci/fixora-Backend/internal/modules/verification/src/usecase"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type VerificationWorker struct {
	Log     *logrus.Logger
	UseCase *usecase.VerificationUseCase
	Cron    *cron.Cron
}

func NewVerificationWorker(log *logrus.Logger, uc *usecase.VerificationUseCase) *VerificationWorker {
	return &VerificationWorker{Log: log, UseCase: uc, Cron: cron.New(cron.WithSeconds())}
}
func (w *VerificationWorker) StartScheduler() error {
	_, err := w.Cron.AddFunc(
		"*/30 * * * * *", 
		func() { w.RunPending() },
	)

	if err != nil {
		w.Log.Fatalf("Failed to schedule verification worker: %+v", err)
		return err
	}
	
	w.Cron.Start()
	w.Log.Info("Verification scheduler started")
	return nil
}
func (w *VerificationWorker) RunPending() {
	ctx := context.Background()
	sessions, err := w.UseCase.FindPending(ctx, 5)
	if err != nil {
		w.Log.Warnf("Failed get pending verification sessions : %+v", err)
		return
	}
	for _, s := range sessions {
		sessCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		if err := w.UseCase.RunVerification(sessCtx, s.ID); err != nil {
			w.Log.Warnf("Failed run verification session %s : %+v", s.ID, err)
		}
		cancel()
	}
}
