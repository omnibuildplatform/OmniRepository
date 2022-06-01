package application

import (
	"context"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"github.com/omnibuildplatform/omni-repository/common/workers"
	"go.uber.org/zap"
	"time"
)

const PullWorkConcurrency = 10

type WorkManager struct {
	Config        config.WorkManager
	Logger        *zap.Logger
	ImageStore    *storage.ImageStorage
	PullChannel   chan models.Image
	VerifyChannel chan models.Image
	closeCh       chan struct{}
	syncWorker    *workers.WorkFetcher
	Context       context.Context
	baseFolder    string
}

func NewWorkManager(ctx context.Context, config config.WorkManager, logger *zap.Logger, imageStore *storage.ImageStorage, baseFolder string) (*WorkManager, error) {
	workManager := WorkManager{
		Config:        config,
		Logger:        logger,
		ImageStore:    imageStore,
		PullChannel:   make(chan models.Image, config.Worker),
		VerifyChannel: make(chan models.Image, config.Worker),
		closeCh:       make(chan struct{}, 1),
		Context:       ctx,
		baseFolder:    baseFolder,
	}
	workFetcher, err := workers.NewWorkFetcher(imageStore, logger, workManager.PullChannel, workManager.VerifyChannel)
	if err != nil {
		return nil, err
	}
	workManager.syncWorker = workFetcher
	return &workManager, nil
}

func (w *WorkManager) GetPullingImageWorker(image *models.Image, localFolder string, worker int) *workers.ImagePuller {
	return workers.NewImagePuller(w.ImageStore, w.Logger, image, localFolder, worker)
}

func (w *WorkManager) GetVerifyingImageWorker(image *models.Image, localFolder string) *workers.ImageVerifier {
	return workers.NewImageVerifier(w.ImageStore, w.Logger, image, localFolder)
}

func (w *WorkManager) Close() {
	close(w.closeCh)
}

func (w *WorkManager) StartLoop() {
	go w.PerformImageWorks()
	syncTicker := time.NewTicker(time.Duration(w.Config.SyncInterval) * time.Second)
	for {
		select {
		case <-syncTicker.C:
			w.Logger.Debug("starting to fetch available works from database")
			err := w.syncWorker.DoWork(w.Context)
			if err != nil {
				w.Logger.Error(fmt.Sprintf("failed to perform database work fetch task, %v", err))
			}
		case <-w.closeCh:
			w.Logger.Info("work manager will quit")
			return
		}
	}
}

func (w *WorkManager) PerformImageWorks() {
	for {
		select {
		case image, ok := <-w.PullChannel:
			if ok {
				w.Logger.Info(fmt.Sprintf("start to perform image download work for image %d", image.ID))
				worker := w.GetPullingImageWorker(&image, w.baseFolder, PullWorkConcurrency)
				err := worker.DoWork(w.Context)
				if err != nil {
					w.Logger.Error(fmt.Sprintf("failed to perform image download work %v", err))
				}
			}
		case image, ok := <-w.VerifyChannel:
			if ok {
				w.Logger.Info(fmt.Sprintf("start to perform image verify work for image %d", image.ID))
				worker := w.GetVerifyingImageWorker(&image, w.baseFolder)
				err := worker.DoWork(w.Context)
				if err != nil {
					w.Logger.Error(fmt.Sprintf("failed to perform image verify work %v", err))
				}
			}
		case <-w.closeCh:
			w.Logger.Info("work manager will quit")
			return
		}
	}
}
