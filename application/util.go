package application

import (
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/models"
)

func GetImageRelativeFolder(image *models.Image) string {
	//Local folder will be generated in the format of:
	//path:   <user-id>/<checksum>/
	//TODO: use checksum as path to reduce storage consumption
	return fmt.Sprintf("/%d/%s", image.UserId, image.Checksum)
}
