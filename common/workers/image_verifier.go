package workers

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/messages"
	"hash"
	"io"
	"os"
	"path"
	"strings"

	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"go.uber.org/zap"
)

const HashingBuffer = 1024 * 1024 * 10

type ImageVerifier struct {
	ImageStore  *storage.ImageStorage
	Image       *models.Image
	LocalFolder string
	Logger      *zap.Logger
	Worker      int
	Notifier    messages.Notifier
}

func NewImageVerifier(imageStore *storage.ImageStorage, logger *zap.Logger, image *models.Image, localFolder string, worker int, notifier messages.Notifier) (*ImageVerifier, error) {
	return &ImageVerifier{
		LocalFolder: localFolder,
		Logger:      logger,
		ImageStore:  imageStore,
		Image:       image,
		Worker:      worker,
		Notifier:    notifier,
	}, nil
}

func (r *ImageVerifier) cleanup(err error) {
	r.Image.Status = models.ImageFailed
	r.Image.StatusDetail = err.Error()
	_ = r.ImageStore.UpdateImageStatusAndDetail(r.Image)
	r.Notifier.Info(string(models.ImageEventFailed), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{
		"detail": err.Error(),
	})
}

func (r *ImageVerifier) getHasher(algorithm string) (hash.Hash, error) {
	if strings.ToLower(algorithm) == "md5" {
		return md5.New(), nil
	} else if strings.ToLower(algorithm) == "sha256" {
		return sha256.New(), nil
	}
	return nil, errors.New(fmt.Sprintf("unsupport digest algorithm %s", algorithm))
}

func (r *ImageVerifier) DoWork(ctx context.Context) error {
	var err error
	r.Image.Status = models.ImageVerifying
	err = r.ImageStore.UpdateImageStatus(r.Image)
	if err != nil {
		return err
	}
	imagePath := path.Join(r.LocalFolder, r.Image.ImagePath)
	if _, err := os.Stat(imagePath); err != nil {
		r.cleanup(err)
		return err
	}
	imageReader, err := os.OpenFile(imagePath, os.O_RDONLY, 0644)
	if err != nil {
		r.cleanup(err)
		return err
	}
	defer imageReader.Close()
	hasher, err := r.getHasher(r.Image.Algorithm)
	if err != nil {
		r.cleanup(err)
		return err
	}
	copyBuf := make([]byte, HashingBuffer)
	if _, err := io.CopyBuffer(hasher, imageReader, copyBuf); err != nil {
		r.cleanup(err)
		return err
	}
	checksum := hex.EncodeToString(hasher.Sum(nil))
	if checksum != r.Image.Checksum {
		err = errors.New(fmt.Sprintf("checksum is not identical to image file's provided %s while actual %s ",
			r.Image.Checksum, checksum))
		r.cleanup(err)
		return err
	}
	err = r.generateChecksumFile(checksum)
	if err != nil {
		r.cleanup(err)
		return err
	}
	r.Image.Status = models.ImageVerified
	r.Image.StatusDetail = "checksum are verified"
	err = r.ImageStore.UpdateImageStatusAndDetail(r.Image)
	if err != nil {
		r.cleanup(err)
		return err
	}
	r.Notifier.Info(string(models.ImageEventVerified), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{
		"checksum": checksum,
	})
	r.Logger.Info(fmt.Sprintf("image %s successfully verified", r.Image.SourceUrl))
	return nil
}

func (r *ImageVerifier) generateChecksumFile(checksum string) error {
	checkSumFile := path.Join(r.LocalFolder, r.Image.ChecksumPath)
	_ = os.Remove(checkSumFile)
	checksumWriter, err := os.OpenFile(checkSumFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer checksumWriter.Close()
	_, err = checksumWriter.Write([]byte(fmt.Sprintf("%s %s", checksum, r.Image.Name)))
	if err != nil {
		return err
	}
	return nil
}

func (r *ImageVerifier) Close() error {
	return nil
}
