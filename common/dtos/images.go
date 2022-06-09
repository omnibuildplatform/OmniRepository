package dtos

import (
	"fmt"
	"strings"
	"time"

	"github.com/omnibuildplatform/omni-repository/common/models"
)

type ImageRequest struct {
	Name              string `description:"name"  form:"name" json:"name" validate:"required"`
	Desc              string `description:"desc"  form:"desc" json:"desc"`
	Checksum          string `description:"checksum" form:"checksum" json:"checksum"`
	Algorithm         string `description:"algorithm" form:"algorithm" json:"algorithm" validate:"required,oneof=md5 sha256"`
	ExternalID        string `description:"externalID" form:"externalID" json:"externalID" validate:"required"`
	SourceUrl         string `description:"source url of images" json:"sourceUrl" form:"sourceUrl"`
	FileName          string `description:"file name" form:"fileName" json:"fileName" validate:"required"`
	UserId            int    `description:"user id" form:"userID" json:"userID" validate:"required"`
	Publish           bool   `description:"publish image to third party storage" form:"publish" json:"publish"  `
	ExternalComponent string `description:"From APP" form:"externalComponent" json:"externalComponent" validate:"required"`
}

type ImageResponse struct {
	ImageRequest
	ID           int                `description:"id" form:"id" json:"id"`
	Status       models.ImageStatus `description:"image status" json:"status"`
	StatusDetail string             `description:"status detail"  json:"statusDetail"`
	ImagePath    string             `description:"image store path"  json:"imagePath"`
	ChecksumPath string             `description:"image checksum store path"  json:"checksumPath"`
	CreateTime   time.Time          `description:"create time" json:"createTime"`
	UpdateTime   time.Time          `description:"update time" json:"updateTime"`
}

type QueryImageRequest struct {
	ExternalID string `form:"externalID" json:"externalID" validate:"required"`
}

type DeleteImageRequest struct {
	UserID   string `form:"userID" json:"userID" validate:"required"`
	Checksum string `form:"checksum" json:"checksum" validate:"required"`
}

type ImageDTO struct {
	browsePrefix string
}

func NewImageDTO(browsePrefix string) *ImageDTO {
	return &ImageDTO{
		browsePrefix: browsePrefix,
	}
}

func (i *ImageDTO) GetImageFromRequest(imageRequest ImageRequest) models.Image {
	return models.Image{
		Name:              imageRequest.Name,
		Desc:              imageRequest.Desc,
		Checksum:          imageRequest.Checksum,
		Algorithm:         imageRequest.Algorithm,
		ExternalID:        imageRequest.ExternalID,
		SourceUrl:         imageRequest.SourceUrl,
		FileName:          imageRequest.FileName,
		UserId:            imageRequest.UserId,
		Publish:           imageRequest.Publish,
		ExternalComponent: imageRequest.ExternalComponent,
	}
}

func (i *ImageDTO) GenerateResponseFromImage(image models.Image) ImageResponse {
	imageResponse := ImageResponse{
		ImageRequest: ImageRequest{
			Name:              image.Name,
			Desc:              image.Desc,
			Checksum:          image.Checksum,
			Algorithm:         image.Algorithm,
			ExternalID:        image.ExternalID,
			SourceUrl:         image.SourceUrl,
			FileName:          image.FileName,
			UserId:            image.UserId,
			Publish:           image.Publish,
			ExternalComponent: image.ExternalComponent,
		},
		ID:           image.ID,
		Status:       image.Status,
		StatusDetail: image.StatusDetail,
		CreateTime:   image.CreateTime,
		UpdateTime:   image.UpdateTime,
	}
	if imageResponse.Status != models.ImagePushed {
		imageResponse.ImagePath = fmt.Sprintf("%s/%s", strings.TrimRight(i.browsePrefix, "/"), strings.TrimLeft(image.ImagePath, "/"))
		imageResponse.ChecksumPath = fmt.Sprintf("%s/%s", strings.TrimRight(i.browsePrefix, "/"), strings.TrimLeft(image.ChecksumPath, "/"))
	} else {
		imageResponse.ImagePath = image.ImagePath
		imageResponse.ChecksumPath = image.ChecksumPath
	}
	return imageResponse
}
