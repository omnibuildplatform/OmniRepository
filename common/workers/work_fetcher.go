package workers

import (
	"context"
	"errors"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"go.uber.org/zap"
	"sync"
)

var fetchDownloading sync.Once

type WorkFetcher struct {
	ImageStore    *storage.ImageStorage
	Logger        *zap.Logger
	PullChannel   chan models.Image
	VerifyChannel chan models.Image
}

func NewWorkFetcher(imageStore *storage.ImageStorage, logger *zap.Logger, pullCh, verifyCh chan models.Image) (*WorkFetcher, error) {
	return &WorkFetcher{
		ImageStore:    imageStore,
		Logger:        logger,
		PullChannel:   pullCh,
		VerifyChannel: verifyCh,
	}, nil
}

func (r *WorkFetcher) DoWork(ctx context.Context) error {
	//1. fetch image which are downloading before start
	fetchDownloadingFailed := false
	fetchDownloading.Do(func() {
		r.Logger.Info("==========initialize work: fetch unfinished work==========")
		images, err := r.ImageStore.GetDownloadingImages()
		if err != nil {
			r.Logger.Error("failed to fetch downloading work from database")
			fetchDownloadingFailed = true
		}
		if len(images) != 0 {
			r.Logger.Info(fmt.Sprintf("found %d unfinished downloading images for download", len(images)))
			for _, image := range images {
				r.PullChannel <- image
			}
		}
	})
	if fetchDownloadingFailed {
		return errors.New("failed to fetch downloading work from database")
	}
	//2. fetch image which are not downloaded
	images, err := r.ImageStore.GetImageForDownload(20)
	if err != nil {
		return err
	}
	if len(images) != 0 {
		r.Logger.Info(fmt.Sprintf("found %d images for download", len(images)))
		for _, image := range images {
			r.PullChannel <- image
		}
	}
	//3. fetch image which are not verified
	images, err = r.ImageStore.GetImageForVerify(20)
	if err != nil {
		return err
	}
	if len(images) != 0 {
		r.Logger.Info(fmt.Sprintf("found %d images for verify", len(images)))
		for _, image := range images {
			r.VerifyChannel <- image
		}
	}
	return nil
}

func (r *WorkFetcher) Close() error {
	return nil
}
