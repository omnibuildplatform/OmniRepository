package app

import (
	"testing"
	"time"
)

func Test_Database(t *testing.T) {
	Bootstrap("../config")
	InitDB()
	//-------------
	item := new(UserImages)
	item.Checksum = "md5234232"
	item.CreateTime = time.Now().In(CnTime)
	item.Desc = " just so so"
	item.FileType = "iso"
	item.Name = "my iso file"
	item.UserId = 112
	item.UserName = "roland"
	err := AddUserImages(item)
	if err != nil {
		t.Fatalf("AddUserImages Error: %s", err)

	}
	t.Logf("AddUserImages result ID:%d", item.ID)

	getItem, getErr := GetUserImagesByID(item.ID)
	if getErr != nil {
		t.Fatalf("GetUserImagesByID Error: %s", getErr)
	}
	t.Logf("GetUserImagesByID result ID:%d", getItem.ID)

	getItem.Name = "other Name"
	err = UpdateUserImages(getItem)
	if err != nil {
		t.Fatalf("UpdateUserImages Error: %s", err)
	}
	t.Logf("UpdateUserImages result name:%s", getItem.Name)

	err = DeleteUserImagesById(getItem.ID)
	if err != nil {
		t.Fatalf("DeleteUserImagesById Error: %s", err)
	}
	t.Logf("DeleteUserImagesById result ID:%d", getItem.ID)

}
