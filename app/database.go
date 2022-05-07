package app

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func GetDB() *gorm.DB {
	return db
}

//connect to database
func InitDB() (err error) {
	conf := Config.StringMap("database")
	dbHost := conf["dbHost"]
	dbUser := conf["dbUser"]
	dbPswd := conf["dbPswd"]
	dbName := conf["dbName"]
	dbPort := conf["dbPort"]
	sqlStr := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", dbUser, dbPswd, dbHost, dbPort, dbName)

	db, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       sqlStr, // DSN data source name
		DefaultStringSize:         256,    // default string size
		DisableDatetimePrecision:  true,   // disable datetime Precision
		DontSupportRenameIndex:    true,   //
		DontSupportRenameColumn:   true,   //
		SkipInitializeWithVersion: false,  //
	}), &gorm.Config{})
	if err != nil {
		return err
	}
	db.Logger.LogMode(3)
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	// SetMaxIdleConns
	sqlDB.SetMaxIdleConns(10)
	// SetMaxOpenConns
	sqlDB.SetMaxOpenConns(100)
	// SetConnMaxLifetime
	sqlDB.SetConnMaxLifetime(time.Hour)
	return nil
}

type Images struct {
	//name, type，url, description, sha256sum, externalID, author
	ID         int       `description:"id" gorm:"primaryKey"`
	Name       string    `description:"name"  form:"name"`
	Desc       string    `description:"desc"   form:"description"`
	UserName   string    `description:"username" form:"username"`
	Checksum   string    `description:"checksum" form:"checksum"`
	Type       string    `description:"type" form:"type"`
	ExternalID string    `description:"externalID" form:"externalID"`
	SourceUrl  string    `description:"source url of images" json:"source_url" form:"source_url"`
	ExtName    string    `description:"file extension name" json:"ext_name"`
	Status     string    `description:"status:start, downloading,done" json:"status"`
	UserId     int       ` description:"user id" `
	CreateTime time.Time ` description:"create time"`
	UpdateTime time.Time ` description:"update time"`
}

func (t *Images) TableName() string {
	return "images"
}

// AddImages insert a new ImageMeta into database and returns
// last inserted ID on success.
func AddImages(m *Images) (err error) {
	o := GetDB()
	m.CreateTime = time.Now().In(CnTime)
	result := o.Create(m)
	return result.Error
}
func UpdateImages(m *Images) (err error) {
	o := GetDB()
	result := o.Updates(m)
	return result.Error
}
func UpdateImagesStatus(m *Images) (err error) {
	o := GetDB()
	result := o.Model(m).Select("status", "update_time").Updates(m)
	return result.Error
}

func GetImagesByID(id int) (v *Images, err error) {
	o := GetDB()
	v = new(Images)
	v.ID = id
	tx := o.Model(v)
	return v, tx.Error
}

func GetImagesByUserID(userid, offset, limit int) (result []*Images, err error) {
	o := GetDB()
	v := new(Images)
	sql := fmt.Sprintf("select * from %s where user_id = %d order by create_time desc limit %d,%d", v.TableName(), userid, offset, limit)
	tx := o.Raw(sql).Scan(&result)
	return result, tx.Error
}
func GetImagesByExternalID(externalID string) (result *Images, err error) {
	o := GetDB()
	v := new(Images)
	sql := fmt.Sprintf("select * from %s where external_id = '%s'  limit 1", v.TableName(), externalID)
	tx := o.Raw(sql).Scan(&result)
	return result, tx.Error
}

// DeleteImagesById
func DeleteImagesById(id int) (err error) {
	o := GetDB()
	m := new(Images)
	m.ID = id
	result := o.Delete(m)
	return result.Error
}

// DeleteMultiImagess
func DeleteMultiImages(names string) (err error) {
	o := GetDB()
	m := new(Images)
	sql := fmt.Sprintf("delete from %s where name in (%s)", m.TableName(), names)
	result := o.Model(m).Exec(sql)
	return result.Error
}
