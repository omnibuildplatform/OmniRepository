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

type UserImages struct {
	ID         int       ` description:"id" gorm:"primaryKey"`
	Name       string    ` description:"name"`
	Desc       string    ` description:"desc"`
	UserName   string    ` description:"user_name"`
	Checksum   string    ` description:"checksum"`
	FileType   string    ` description:"file_type"`
	UserId     int       ` description:"user id"`
	CreateTime time.Time ` description:"create time"`
}

func (t *UserImages) TableName() string {
	return "user_images"
}

// AddUserImages insert a new ImageMeta into database and returns
// last inserted Id on success.
func AddUserImages(m *UserImages) (err error) {
	o := GetDB()
	result := o.Create(m)
	return result.Error
}
func UpdateUserImages(m *UserImages) (err error) {
	o := GetDB()
	result := o.Updates(m)
	return result.Error
}

func GetUserImagesByID(id int) (v *UserImages, err error) {
	o := GetDB()
	v = new(UserImages)
	v.ID = id
	tx := o.Model(v)
	return v, tx.Error
}

func GetUserImagesByUserID(userid, offset, limit int) (result []*UserImages, err error) {
	o := GetDB()
	v := new(UserImages)
	sql := fmt.Sprintf("select * from %s where user_id = %d order by create_time desc limit %d,%d", v.TableName(), userid, offset, limit)
	tx := o.Raw(sql).Scan(&result)
	return result, tx.Error
}

// DeleteUserImagesById
func DeleteUserImagesById(id int) (err error) {
	o := GetDB()
	m := new(UserImages)
	m.ID = id
	result := o.Delete(m)
	return result.Error
}

// DeleteMultiUserImagess
func DeleteMultiUserImagess(names string) (err error) {
	o := GetDB()
	m := new(UserImages)
	sql := fmt.Sprintf("delete from %s  where job_name in (%s)", m.TableName(), names)
	result := o.Model(m).Exec(sql)
	return result.Error
}
