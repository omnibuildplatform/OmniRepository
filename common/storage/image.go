package storage

import (
	"context"
	"time"

	"github.com/omnibuildplatform/omni-repository/common/models"
	"gorm.io/gorm"
)

type ImageStorage struct {
	db      *gorm.DB
	context context.Context
}

func NewImageStorage(db *gorm.DB, ctx context.Context) *ImageStorage {
	return &ImageStorage{db: db, context: ctx}
}

func (i *ImageStorage) AddImage(m *models.Image) (err error) {
	m.CreateTime = time.Now()
	m.UpdateTime = time.Now()
	if len(m.Status) == 0 {
		m.Status = models.ImageCreated
	}
	result := i.db.WithContext(i.context).Model(m).Create(m)
	return result.Error
}

func (i *ImageStorage) SoftDeleteImage(m *models.Image) (err error) {
	m.UpdateTime = time.Now()
	m.Deleted = true
	result := i.db.WithContext(i.context).Model(m).Select("deleted", "update_time").Updates(m)
	return result.Error
}

func (i *ImageStorage) UpdateImage(m *models.Image) (err error) {
	m.UpdateTime = time.Now()
	result := i.db.WithContext(i.context).Updates(m)
	return result.Error
}
func (i *ImageStorage) UpdateImageStatus(m *models.Image) (err error) {
	m.UpdateTime = time.Now()
	result := i.db.WithContext(i.context).Model(m).Select("status", "update_time").Updates(m)
	return result.Error
}

func (i *ImageStorage) UpdateImageExternalPath(m *models.Image) (err error) {
	m.UpdateTime = time.Now()
	result := i.db.WithContext(i.context).Model(m).Select("image_path", "checksum_path", "update_time").Updates(m)
	return result.Error
}

func (i *ImageStorage) GetImageByChecksumAndUserID(userID, checksum string) (models.Image, error) {
	var image models.Image
	result := i.db.WithContext(i.context).Where("checksum = ? AND user_id = ? AND deleted = ?", checksum, userID, false).Order("create_time desc").First(&image)
	return image, result.Error
}

func (i *ImageStorage) UpdateImageStatusAndDetail(m *models.Image) error {
	m.UpdateTime = time.Now()
	result := i.db.WithContext(i.context).Model(m).Select("status", "update_time", "status_detail").Updates(m)
	return result.Error
}

func (i *ImageStorage) GetImageByID(id int) (models.Image, error) {
	var image models.Image
	result := i.db.WithContext(i.context).Where("deleted = ?", false).First(&image, id)
	return image, result.Error
}

func (i *ImageStorage) GetImagesByStatus(status models.ImageStatus, limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("status = ? AND deleted = ? ", status, false).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetImageForDownload(limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("status = ? AND source_url != '' AND deleted = ?", models.ImageCreated, false).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetDownloadingImages() ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("status = ? AND deleted = ?", models.ImageDownloading, false).Order("create_time desc").Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetPushingImages() ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("status = ? AND deleted = ?", models.ImagePushing, false).Order("create_time desc").Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetImageForVerify(limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("status = ? AND deleted = ?", models.ImageDownloaded, false).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetImageForPush(limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("status = ? AND publish = ? AND deleted = ?", models.ImageVerified, true, false).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetImageForClean(limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("deleted = ? OR status = ?", true, models.ImagePushed).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}

func (i *ImageStorage) GetImagesByUserID(userid, offset, limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.WithContext(i.context).Where("user_id = ? AND deleted = ?", userid, false).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}
func (i *ImageStorage) GetImageByExternalID(externalID string) (models.Image, error) {
	var image models.Image
	result := i.db.WithContext(i.context).Where("external_id = ? AND deleted = ? ", externalID, false).First(&image)
	return image, result.Error
}

func (i *ImageStorage) DeleteImageById(id int) error {
	result := i.db.WithContext(i.context).Delete(&models.Image{}, id)
	return result.Error
}
