package models

import "time"

type Image struct {
	ID         int       `description:"id" gorm:"primaryKey"`
	Name       string    `description:"name"  form:"name"`
	Desc       string    `description:"desc"   form:"description"`
	UserName   string    `description:"username" form:"username"`
	Checksum   string    `description:"checksum" form:"checksum"`
	Type       string    `description:"type" form:"type"`
	ExternalID string    `description:"externalID" form:"externalID"`
	SourceUrl  string    `description:"source url of images" json:"source_url" form:"source_url"`
	ExtName    string    `description:"file extension name" json:"ext_name"`
	Status     string    `description:"status:start, downloading, done" json:"status"`
	UserId     int       `description:"user id" `
	CreateTime time.Time `description:"create time"`
	UpdateTime time.Time `description:"update time"`
}

func (Image) TableName() string {
	return "images"
}
