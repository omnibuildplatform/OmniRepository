package models

import "time"

type ImageStatus string

const (
	ImageCreated     ImageStatus = "ImageCreated"
	ImageDownloading ImageStatus = "ImageDownloading"
	ImageDownloaded  ImageStatus = "ImageDownloaded"
	ImageVerifying   ImageStatus = "ImageVerifying"
	ImageVerified    ImageStatus = "ImageVerified"
	ImagePushing     ImageStatus = "ImagePushing"
	ImagePushed      ImageStatus = "ImagePushed"
	ImageFailed      ImageStatus = "ImageFailed"
)

type ImageBlockStatus string

const (
	ImageBlockFinished ImageBlockStatus = "ImageBlockFinished"
	ImageBlockFailed   ImageBlockStatus = "ImageBlockFailed"
	ImageBlockCreated  ImageBlockStatus = "ImageBlockCreated"
)

type Image struct {
	ID                int         `description:"id" gorm:"primaryKey"`
	Name              string      `description:"name"`
	Desc              string      `description:"desc"`
	Checksum          string      `description:"checksum"`
	Algorithm         string      `description:"algorithm" gorm:"sha256"`
	ExternalID        string      `description:"externalID"`
	SourceUrl         string      `description:"source url of images"`
	FileName          string      `description:"file name"`
	UserId            int         `description:"user id"`
	Status            ImageStatus `description:"image status"`
	StatusDetail      string      `description:"status detail"`
	ImagePath         string      `description:"image store path"`
	ChecksumPath      string      `description:"image checksum store path"`
	CreateTime        time.Time   `description:"create time"`
	UpdateTime        time.Time   `description:"update time"`
	Publish           bool        `description:"publish image to third party storage"`
	ExternalComponent string      `description:"eg. omni-manager , ....."`
}

func (Image) TableName() string {
	return "images"
}
