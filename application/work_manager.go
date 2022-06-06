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

type WorkManager struct {
	Config        config.WorkManager
	Logger        *zap.Logger
	ImageStore    *storage.ImageStorage
	PullChannel   chan models.Image
	VerifyChannel chan models.Image
	PushChannel   chan models.Image
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
		PullChannel:   make(chan models.Image, config.Threads),
		VerifyChannel: make(chan models.Image, config.Threads),
		PushChannel:   make(chan models.Image, config.Threads),
		closeCh:       make(chan struct{}, 1),
		Context:       ctx,
		baseFolder:    baseFolder,
	}
	workFetcher, err := workers.NewWorkFetcher(imageStore, logger, workManager.PullChannel, workManager.VerifyChannel, workManager.PushChannel)
	if err != nil {
		return nil, err
	}
	workManager.syncWorker = workFetcher
	return &workManager, nil
}

func (w *WorkManager) GetVerifyingImageWorker(image *models.Image, localFolder string, worker int) (*workers.ImageVerifier, error) {
	return workers.NewImageVerifier(w.ImageStore, w.Logger, image, localFolder, worker)
}

func (w *WorkManager) GetPushImageWorker(image *models.Image, localFolder string, worker int) (*workers.ImagePusher, error) {
	return workers.NewImagePusher(w.Config.Workers.ImagePusher, w.ImageStore, image, localFolder, w.Logger, worker)
}

func (w *WorkManager) GetPullingImageWorker(image *models.Image, localFolder string, worker int) (*workers.ImagePuller, error) {
	return workers.NewImagePuller(w.Config.Workers.ImagePuller, w.ImageStore, w.Logger, image, localFolder, worker)
}

func (w *WorkManager) Close() {
	close(w.closeCh)
}

func (w *WorkManager) StartLoop() {
	for index := 1; index <= w.Config.Threads; index += 1 {
		go w.PerformImageWorks()
	}
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
				worker, err := w.GetPullingImageWorker(&image, w.baseFolder, w.Config.Threads)
				if err != nil {
					w.Logger.Error(fmt.Sprintf("failed to get image puller worker %v", err))
				} else {
					err := worker.DoWork(w.Context)
					if err != nil {
						w.Logger.Error(fmt.Sprintf("failed to perform image download work %v", err))
					}
				}
			}
		case image, ok := <-w.VerifyChannel:
			if ok {
				w.Logger.Info(fmt.Sprintf("start to perform image verify work for image %d", image.ID))
				worker, err := w.GetVerifyingImageWorker(&image, w.baseFolder, w.Config.Threads)
				if err != nil {
					w.Logger.Error(fmt.Sprintf("failed to get image sign worker %v", err))
				} else {
					err := worker.DoWork(w.Context)
					if err != nil {
						w.Logger.Error(fmt.Sprintf("failed to perform image verify work %v", err))
					}
				}
			}
		case image, ok := <-w.PushChannel:
			if ok {
				w.Logger.Info(fmt.Sprintf("start to perform image push work for image %d", image.ID))
				worker, err := w.GetPushImageWorker(&image, w.baseFolder, w.Config.Threads)
				if err != nil {
					w.Logger.Error(fmt.Sprintf("failed to get image push worker %v", err))
				} else {
					err := worker.DoWork(w.Context)
					if err != nil {
						w.Logger.Error(fmt.Sprintf("failed to perform image push work %v", err))
					}
				}
			}
		case <-w.closeCh:
			w.Logger.Info("work manager will quit")
			return
		}
	}
}
