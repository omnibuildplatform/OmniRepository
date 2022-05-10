package app

import (
	"testing"
	"time"
)

func Test_Database(t *testing.T) {
	Bootstrap("../config")
	InitDB()
	//-------------
	item := new(Images)
	item.Checksum = "md5234232"
	item.CreateTime = time.Now().In(CnTime)
	item.Desc = " just so so"
	item.Type = "iso"
	item.Name = "my iso file"
	item.UserId = 112
	item.UserName = "roland"
	err := AddImages(item)
	if err != nil {
		t.Fatalf("AddImages Error: %s", err)

	}
	t.Logf("AddImages result ID:%d", item.ID)

	getItem, getErr := GetImagesByID(item.ID)
	if getErr != nil {
		t.Fatalf("GetImagesByID Error: %s", getErr)
	}
	t.Logf("GetImagesByID result ID:%d", getItem.ID)

	getItem.Name = "other Name"
	err = UpdateImages(getItem)
	if err != nil {
		t.Fatalf("UpdateImages Error: %s", err)
	}
	t.Logf("UpdateImages result name:%s", getItem.Name)

	err = DeleteImagesById(getItem.ID)
	if err != nil {
		t.Fatalf("DeleteImagesById Error: %s", err)
	}
	t.Logf("DeleteImagesById result ID:%d", getItem.ID)

}
