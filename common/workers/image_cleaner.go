package workers

import (
	"context"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/messages"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"go.uber.org/zap"
	"os"
	"path"
	"path/filepath"
)

type ImageCleaner struct {
	ImageStore  *storage.ImageStorage
	Image       *models.Image
	LocalFolder string
	Logger      *zap.Logger
	Notifier    messages.Notifier
}

func NewImageCleaner(imageStore *storage.ImageStorage, logger *zap.Logger, image *models.Image, localFolder string, notifier messages.Notifier) (*ImageCleaner, error) {
	return &ImageCleaner{
		LocalFolder: filepath.Dir(path.Join(localFolder, image.ImagePath)),
		Logger:      logger,
		ImageStore:  imageStore,
		Image:       image,
		Notifier:    notifier,
	}, nil
}

func (r *ImageCleaner) DoWork(ctx context.Context) error {
	err := os.RemoveAll(r.LocalFolder)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to clean up folder for image %s, %v", r.Image.ImagePath, err.Error()))
	}
	r.Notifier.NonBlockPush(string(models.ImageEventCleaned), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{})
	if r.Image.Deleted == true {
		err := r.ImageStore.DeleteImageById(r.Image.ID)
		if err != nil {
			r.Logger.Error(fmt.Sprintf("failed to hard delete image record %s, %v", r.Image.ImagePath, err.Error()))
		}
	}
	return nil
}

func (r *ImageCleaner) Close() {
}
