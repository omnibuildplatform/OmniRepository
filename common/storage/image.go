package storage

import (
	"github.com/omnibuildplatform/omni-repository/common/models"
	"gorm.io/gorm"
	"time"
)

type ImageStorage struct {
	db       *gorm.DB
	location *time.Location
}

func NewImageStorage(db *gorm.DB, location *time.Location) *ImageStorage {
	return &ImageStorage{db: db, location: location}
}

func (i *ImageStorage) AddImage(m *models.Image) (err error) {
	m.CreateTime = time.Now().In(i.location)
	result := i.db.Model(m).Create(m)
	return result.Error
}
func (i *ImageStorage) UpdateImage(m *models.Image) (err error) {
	result := i.db.Updates(m)
	return result.Error
}
func (i *ImageStorage) UpdateImageStatus(m *models.Image) (err error) {
	result := i.db.Model(m).Select("status", "update_time").Updates(m)
	return result.Error
}

func (i *ImageStorage) GetImageByID(id int) (v *models.Image, err error) {
	var image models.Image
	result := i.db.First(image, id)
	return &image, result.Error
}

func (i *ImageStorage) GetImagesByUserID(userid, offset, limit int) ([]models.Image, error) {
	var images []models.Image
	result := i.db.Where("user_id = ?", userid).Order("create_time desc").Limit(limit).Find(&images)
	return images, result.Error
}
func (i *ImageStorage) GetImageByExternalID(externalID string) (*models.Image, error) {
	var image models.Image
	result := i.db.Where("external_id = ?", externalID).First(image)
	return &image, result.Error
}

func (i *ImageStorage) DeleteImageById(id int) (err error) {
	result := i.db.Delete(&models.Image{}, id)
	return result.Error
}
