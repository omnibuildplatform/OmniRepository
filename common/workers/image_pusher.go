package workers

import (
	"context"
	"errors"
	"fmt"
	"github.com/huaweicloud/huaweicloud-sdk-go-obs/obs"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"github.com/omnibuildplatform/omni-repository/common/messages"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"go.uber.org/zap"
	"os"
	"path"
	"strings"
)

const NotFoundError = "Status=404 Not Found"

type ImagePusher struct {
	imageStore  *storage.ImageStorage
	Image       *models.Image
	LocalFolder string
	Logger      *zap.Logger
	Config      config.ImagePusher
	OBSClient   *obs.ObsClient
	Worker      int
	Notifier    messages.Notifier
}

func NewImagePusher(config config.ImagePusher, imageStore *storage.ImageStorage, image *models.Image, localFolder string, logger *zap.Logger, worker int, notifier messages.Notifier) (*ImagePusher, error) {
	if len(config.AK) == 0 || len(config.SK) == 0 || len(config.Endpoint) == 0 {
		return nil, errors.New("incorrect ak/sk/endpoint config for image pusher")
	}
	obsClient, err := obs.New(config.AK, config.SK, fmt.Sprintf("https://%s", config.Endpoint))
	if err != nil {
		return nil, err

	}
	// check bucket whether exists
	_, err = obsClient.GetBucketLocation(config.Bucket)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to get bucket information %s", config.Bucket))
		return nil, err
	}

	return &ImagePusher{
		Config:      config,
		imageStore:  imageStore,
		Image:       image,
		Logger:      logger,
		LocalFolder: localFolder,
		OBSClient:   obsClient,
		Worker:      worker,
		Notifier:    notifier,
	}, nil
}

func (r *ImagePusher) cleanup(err error) {
	r.Image.Status = models.ImageFailed
	r.Image.StatusDetail = err.Error()
	_ = r.imageStore.UpdateImageStatusAndDetail(r.Image)
	r.Notifier.NonBlockPush(string(models.ImageEventFailed), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{
		"detail": err.Error(),
	})
}

func (r *ImagePusher) DoWork(ctx context.Context) error {
	r.Image.Status = models.ImagePushing
	err := r.imageStore.UpdateImageStatus(r.Image)
	if err != nil {
		return err
	}
	//1. create folder
	folderKey := fmt.Sprintf("%d/%s/", r.Image.UserId, r.Image.Checksum)
	if err = r.createFolderIfNeeded(folderKey); err != nil {
		r.cleanup(err)
		return err
	}
	//2. create image checksum object
	checksumNames := strings.Split(r.Image.ChecksumPath, "/")
	checksumName := fmt.Sprintf("%s%s", folderKey, checksumNames[len(checksumNames)-1])
	if exists, err := r.objectExists(checksumName); err != nil {
		return err
	} else if exists {
		r.Logger.Info(fmt.Sprintf("found existing file %s on obs will delete first", checksumName))
		err = r.deleteObject(checksumName)
		if err != nil {
			r.cleanup(err)
			return err
		}
	}
	err = r.concurrentPushObject(path.Join(r.LocalFolder, r.Image.ChecksumPath), checksumName)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to upload image file %s %v", checksumName, err))
		r.cleanup(err)
		return err
	}
	//3. create image object
	imageNames := strings.Split(r.Image.ImagePath, "/")
	imageKey := fmt.Sprintf("%s%s", folderKey, imageNames[len(imageNames)-1])
	if exists, err := r.objectExists(imageKey); err != nil {
		r.cleanup(err)
		return err
	} else if exists {
		r.Logger.Info(fmt.Sprintf("found existing file %s on obs will delete first", imageKey))
		err = r.deleteObject(imageKey)
		if err != nil {
			r.cleanup(err)
			return err
		}
	}
	err = r.concurrentPushObject(path.Join(r.LocalFolder, r.Image.ImagePath), imageKey)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to upload image file %s %v", imageKey, err))
		r.cleanup(err)
		return err
	}
	//4. update object status and link
	r.Image.Status = models.ImagePushed
	err = r.imageStore.UpdateImageStatus(r.Image)
	if err != nil {
		r.cleanup(err)
		return err
	}
	r.Image.ImagePath = fmt.Sprintf("https://%s.%s/%s", r.Config.Bucket, r.Config.Endpoint, strings.TrimLeft(r.Image.ImagePath, "/"))
	r.Image.ChecksumPath = fmt.Sprintf("https://%s.%s/%s", r.Config.Bucket, r.Config.Endpoint, strings.TrimLeft(r.Image.ChecksumPath, "/"))
	err = r.imageStore.UpdateImageExternalPath(r.Image)
	if err != nil {
		r.cleanup(err)
		return err
	}
	r.Notifier.NonBlockPush(string(models.ImageEventPushed), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{
		"imagePath":    r.Image.ImagePath,
		"checksumPath": r.Image.ChecksumPath,
	})
	return nil
}

func (r *ImagePusher) createFolderIfNeeded(name string) error {
	if exists, err := r.objectExists(name); err != nil {
		r.Logger.Error(fmt.Sprintf("failed to check obs folder existence for %s", name))
		return err
	} else if !exists {
		err = r.createSingleLevelFolder(name)
		if err != nil {
			r.Logger.Error(fmt.Sprintf("failed to create obs folder for %s", name))
			return err
		}
	}
	r.Logger.Info(fmt.Sprintf("folder %s has been successfully created", name))
	return nil
}

func (r *ImagePusher) Close() {
}

func (r *ImagePusher) objectExists(path string) (bool, error) {
	input := &obs.GetObjectMetadataInput{}
	input.Bucket = r.Config.Bucket
	input.Key = path
	_, err := r.OBSClient.GetObjectMetadata(input)
	if err != nil {
		if strings.Contains(err.Error(), NotFoundError) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *ImagePusher) createSingleLevelFolder(path string) error {
	var input = &obs.PutObjectInput{}
	input.Bucket = r.Config.Bucket
	input.Key = fmt.Sprintf("%s/", strings.TrimRight(path, "/"))
	_, err := r.OBSClient.PutObject(input)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to create single level folder %s ", path))
		return err
	}
	return nil
}

func (r *ImagePusher) concurrentPushObject(localPath, name string) error {
	input := &obs.InitiateMultipartUploadInput{}
	input.Bucket = r.Config.Bucket
	input.Key = name
	output, err := r.OBSClient.InitiateMultipartUpload(input)
	if err != nil {
		return err
	}
	uploadId := output.UploadId
	r.Logger.Info(fmt.Sprintf("Claiming a new upload id %s for file %s", uploadId, name))

	// Calculate how many blocks to be divided into small blocks, for example, 20MB, 419430400
	partSize := r.Config.PartSize
	stat, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	fileSize := stat.Size()
	partCount := int(fileSize / partSize)
	if fileSize%partSize != 0 {
		partCount++
	}
	r.Logger.Info(fmt.Sprintf("file %s will be devided into %d parts", name, partCount))

	partChan := make(chan obs.Part, r.Worker)

	for i := 0; i < partCount; i++ {
		partNumber := i + 1
		offset := int64(i) * partSize
		currPartSize := partSize
		if i+1 == partCount {
			currPartSize = fileSize - offset
		}
		go func(index int, offset, partSize int64, logger *zap.Logger) {
			uploadPartInput := &obs.UploadPartInput{}
			uploadPartInput.Bucket = r.Config.Bucket
			uploadPartInput.Key = name
			uploadPartInput.UploadId = uploadId
			uploadPartInput.SourceFile = localPath
			uploadPartInput.PartNumber = index
			uploadPartInput.Offset = offset
			uploadPartInput.PartSize = partSize
			logger.Info(fmt.Sprintf("starting to upload block %d.", index))
			uploadPartInputOutput, errMsg := r.OBSClient.UploadPart(uploadPartInput)
			if errMsg == nil {
				logger.Info(fmt.Sprintf("upload block %d finished", index))
				partChan <- obs.Part{ETag: uploadPartInputOutput.ETag, PartNumber: uploadPartInputOutput.PartNumber}
			} else {
				logger.Error(fmt.Sprintf("upload block %d failed with error %v", index, err))
				partChan <- obs.Part{ETag: "fake etag", PartNumber: -1}
			}
		}(partNumber, offset, currPartSize, r.Logger)
	}
	parts := make([]obs.Part, 0, partCount)
	for {
		part, ok := <-partChan
		if !ok {
			r.Logger.Info("push worker will quit")
			break
		}
		if part.PartNumber == -1 {
			r.Logger.Error("failed to push file part, job abandoned")
			break
		} else {
			parts = append(parts, part)
			if len(parts) == partCount {
				close(partChan)
			}
		}
	}

	completeMultipartUploadInput := &obs.CompleteMultipartUploadInput{}
	completeMultipartUploadInput.Bucket = r.Config.Bucket
	completeMultipartUploadInput.Key = name
	completeMultipartUploadInput.UploadId = uploadId
	completeMultipartUploadInput.Parts = parts
	_, err = r.OBSClient.CompleteMultipartUpload(completeMultipartUploadInput)
	if err != nil {
		return err
	}
	r.Logger.Info(fmt.Sprintf("push %s work finished", name))
	return nil
}

func (r *ImagePusher) deleteObject(name string) error {
	input := &obs.DeleteObjectInput{}
	input.Bucket = r.Config.Bucket
	input.Key = name

	_, err := r.OBSClient.DeleteObject(input)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to delete object %s", name))
		return err
	}
	r.Logger.Info(fmt.Sprintf("object %s deleted successfully", name))
	return nil
}
