package application

import (
	"context"
	"errors"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"github.com/omnibuildplatform/omni-repository/common/messages"
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
	WorkerChannel chan workers.ImageWork
	closeCh       chan struct{}
	syncWorker    *workers.WorkFetcher
	Context       context.Context
	baseFolder    string
	Notifier      messages.Notifier
}

func NewWorkManager(ctx context.Context, config config.WorkManager, logger *zap.Logger, imageStore *storage.ImageStorage, baseFolder string, notifier messages.Notifier) (*WorkManager, error) {
	workManager := WorkManager{
		Config:        config,
		Logger:        logger,
		ImageStore:    imageStore,
		WorkerChannel: make(chan workers.ImageWork, config.Threads*4),
		closeCh:       make(chan struct{}, 1),
		Context:       ctx,
		baseFolder:    baseFolder,
		Notifier:      notifier,
	}
	workFetcher, err := workers.NewWorkFetcher(imageStore, logger, workManager.WorkerChannel)
	if err != nil {
		return nil, err
	}
	workManager.syncWorker = workFetcher
	return &workManager, nil
}

func (w *WorkManager) GetVerifyingImageWorker(image *models.Image, localFolder string, worker int) (*workers.ImageVerifier, error) {
	return workers.NewImageVerifier(w.ImageStore, w.Logger, image, localFolder, worker, w.Notifier)
}

func (w *WorkManager) GetPushImageWorker(image *models.Image, localFolder string, worker int) (*workers.ImagePusher, error) {
	return workers.NewImagePusher(w.Config.Workers.ImagePusher, w.ImageStore, image, localFolder, w.Logger, worker, w.Notifier)
}

func (w *WorkManager) GetPullingImageWorker(image *models.Image, localFolder string, worker int) (*workers.ImagePuller, error) {
	return workers.NewImagePuller(w.Config.Workers.ImagePuller, w.ImageStore, w.Logger, image, localFolder, worker, w.Notifier)
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

func (w *WorkManager) GetImageWorker(work workers.ImageWork) (workers.Worker, error) {
	if work.Type == workers.PullImageWork {
		w.Logger.Info(fmt.Sprintf("start to perform image download work for image %d", work.Image.ID))
		return workers.NewImagePuller(
			w.Config.Workers.ImagePuller,
			w.ImageStore, w.Logger, &work.Image,
			w.baseFolder, w.Config.Threads, w.Notifier)
	} else if work.Type == workers.PushImageWork {
		w.Logger.Info(fmt.Sprintf("start to perform image push work for image %d", work.Image.ID))
		return workers.NewImagePusher(
			w.Config.Workers.ImagePusher,
			w.ImageStore, &work.Image, w.baseFolder,
			w.Logger, w.Config.Threads, w.Notifier)
	} else if work.Type == workers.SignImageWork {
		w.Logger.Info(fmt.Sprintf(
			"start to perform image verify work for image %d", work.Image.ID))
		return workers.NewImageVerifier(w.ImageStore, w.Logger,
			&work.Image, w.baseFolder, w.Config.Threads, w.Notifier)
	} else if work.Type == workers.CleanImageWork {
		return workers.NewImageCleaner(w.ImageStore, w.Logger, &work.Image, w.baseFolder, w.Notifier)
	}
	return nil, errors.New("unsupported image work")
}

func (w *WorkManager) PerformImageWorks() {
	for {
		select {
		case work, ok := <-w.WorkerChannel:
			if ok {
				worker, err := w.GetImageWorker(work)
				if err != nil {
					w.Logger.Error(fmt.Sprintf("failed to get image worker %v", err))
				} else {
					err := worker.DoWork(w.Context)
					if err != nil {
						w.Logger.Error(fmt.Sprintf("failed to perform image work %v", err))
					}
				}
			}
		case <-w.closeCh:
			w.Logger.Info("work manager will quit")
			return
		}
	}
}
