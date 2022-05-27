package application

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gookit/color"
	"github.com/gookit/goutil/fsutil"
	"github.com/omnibuildplatform/omni-repository/app"
)

type PackageType string

const (
	RPM                    PackageType = "rpm"
	Image                  PackageType = "image"
	Toolchain              PackageType = "toolchain"
	BuildImageFromRelease  string      = "buildimagefromrelease"
	BuildImageFromISO      string      = "buildimagefromiso"
	ImageStatusStart       string      = "created"
	ImageStatusDownloading string      = "running"
	ImageStatusDone        string      = "succeed"
	ImageStatusFailed      string      = "failed"
)

type UploadFilePath struct {
	Path string
}

type RepositoryManager struct {
	dataFolder  string
	routerGroup *gin.RouterGroup
	uploadToken string
	imageStore  *storage.ImageStorage
	config      config.RepoManager
}

func NewRepositoryManager(config config.RepoManager, routerGroup *gin.RouterGroup, imageStore *storage.ImageStorage) (*RepositoryManager, error) {
	baseFolder := config.DataFolder
	if !fsutil.DirExist(baseFolder) {
		color.Error.Println("data folder %s not existed", baseFolder)
		return nil, errors.New("data folder not existed")
	}
	token := config.UploadToken
	if len(token) == 0 {
		color.Error.Println("upload token is empty")
		return nil, errors.New("upload token is empty")
	}

	return &RepositoryManager{
		dataFolder:  baseFolder,
		routerGroup: routerGroup,
		uploadToken: token,
		imageStore:  imageStore,
		config:      config,
	}, nil
}

func (r *RepositoryManager) Initialize() error {
	// register upload and file browse route
	r.routerGroup.POST("/upload", r.Upload)
	r.routerGroup.StaticFS("/browse", http.Dir(r.dataFolder))
	r.routerGroup.POST("/loadfrom", r.LoadFrom)
	r.routerGroup.GET("/query", r.Query)
	return nil
}

func (r *RepositoryManager) checkToken(request *http.Request) error {
	token := request.URL.Query().Get("token")
	if token == "" {
		token = request.Form.Get("token")
	}
	if token == "" {
		token = request.URL.Query().Get("token")
	}
	if token == "" {
		token = request.URL.Query().Get("token")
	}
	if token == "" {
		token = request.URL.Query().Get("token")
	}
	if token == "" || token != r.uploadToken {
		return errors.New(token + "token mismatch:" + r.uploadToken)
	}
	return nil
}

func (r *RepositoryManager) Upload(c *gin.Context) {

	var (
		image                                  models.Image
		targetDir, fullPath, filename, extName string
	)

	err := c.MustBindWith(&image, binding.FormMultipart)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "MustBindWith", err.Error()))
		return
	}

	image.Checksum = strings.ToLower(image.Checksum)
	srcFile, fileinfo, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "FormFile", err.Error()))
		return
	}
	defer srcFile.Close()

	filename = fileinfo.Filename
	if strings.Contains(filename, ".") {
		splitList := strings.Split(filename, ".")
		extName = splitList[len(splitList)-1]
		if strings.Contains(extName, "?") {
			extName = strings.Split(extName, "?")[0]
		}
		if strings.Contains(extName, "#") {
			extName = strings.Split(extName, "#")[0]
		}
		if strings.Contains(extName, "&") {
			extName = strings.Split(extName, "&")[0]
		}
	} else {
		extName = "binary"
	}
	if len(image.ExternalID) < 10 {
		image.ExternalID = app.RandomString(20)
	}
	image.ExternalID = strings.ToLower(image.ExternalID)
	hasChecksum := true

	if len(image.Checksum) < 10 {
		targetDir = path.Join(r.dataFolder, image.ExternalID[0:3])
		filename = image.ExternalID + "." + extName
		image.Checksum = image.ExternalID
		hasChecksum = false
	} else {
		filename = image.Checksum + "." + extName
		targetDir = path.Join(r.dataFolder, image.Checksum[0:3])
	}

	fullPath = path.Join(targetDir, filename)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		color.Error.Println(fullPath + "----------------MkdirAll----" + err.Error())
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "MkdirAll", err.Error()))
		return
	}

	_, err = os.Stat(fullPath)
	if err == nil {
		color.Error.Println(fullPath + "----------------Stat----")
		image.ExtName = extName
		image.Status = ImageStatusDone
		c.JSON(http.StatusCreated, app.ExportData(http.StatusCreated, "ok", image))
		return
	}

	dstFile, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		color.Error.Println(fullPath + "----------------OpenFile----" + err.Error())
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "OpenFile", err.Error()))
		return
	}

	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		color.Error.Println("-------------file---Copy----" + err.Error())
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "Copy", err.Error()))
		return
	}
	dstFile.Seek(0, io.SeekStart)
	hash := sha256.New()
	if _, err := io.Copy(hash, dstFile); err != nil {
		color.Error.Println("---------------- io.Copy----" + err.Error())
		image.Status = ImageStatusFailed
		return
	}
	checksumValue := fmt.Sprintf("%x", hash.Sum(nil))
	if hasChecksum {
		if image.Checksum != checksumValue {
			color.Error.Println(image.Checksum + "---------------Checksum----" + checksumValue)
			c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "file's sha256sum not equal input checkSum ", checksumValue))
			return
		}
	}
	image.ExtName = extName
	image.Status = ImageStatusDone
	err = r.imageStore.AddImage(&image)
	if err != nil {

		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "AddUserImages", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, app.ExportData(http.StatusCreated, "ok", image))
}

func (r *RepositoryManager) StartLoop() {
}

func (r *RepositoryManager) Close() {
	//todo :

}

func (r *RepositoryManager) Query(c *gin.Context) {
	externalID := c.Query("externalID")
	if len(externalID) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "bad request", "missing externalID"))
		return
	}
	item, _ := r.imageStore.GetImageByExternalID(externalID)
	if item == nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "image not found by this externalID", externalID))
		return
	}
	item.Checksum = strings.ToLower(item.Checksum)
	downloadURL := " "
	// if item.Type == BuildImageFromISO {
	// 	downloadURL = "/data/browse/" + item.ExternalID[0:3] + "/" + item.ExternalID
	// } else {
	downloadURL = "/data/browse/" + item.Checksum[0:3] + "/" + item.Checksum
	// }
	if item.ExtName != "binary" {
		downloadURL = downloadURL + "." + item.ExtName
	}
	location := url.URL{Path: downloadURL}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func (r *RepositoryManager) LoadFrom(c *gin.Context) {
	var (
		image models.Image
		err   error
	)
	if err = r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "forbidden", err.Error()))
		return
	}
	image.SourceUrl = c.Query("url")
	if len(image.SourceUrl) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "bad request", "missing url"))
		return
	}
	image.UserId, _ = strconv.Atoi(c.Query("userid"))
	image.Checksum = strings.ToLower(c.Query("checksum"))
	if len(image.Checksum) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "bad request", "missing checksum"))
		return
	}
	image.Name = c.Query("name")
	image.Desc = c.Query("desc")
	image.ExternalID = c.Query("externalID")

	var fullPath, extName string
	_, filename := path.Split(image.SourceUrl)
	targetDir := r.dataFolder
	if strings.Contains(filename, ".") {
		splitList := strings.Split(filename, ".")
		extName = splitList[len(splitList)-1]
		if strings.Contains(extName, "?") {
			extName = strings.Split(extName, "?")[0]
		}
		if strings.Contains(extName, "#") {
			extName = strings.Split(extName, "#")[0]
		}
		if strings.Contains(extName, "&") {
			extName = strings.Split(extName, "&")[0]
		}
		filename = image.Checksum + "." + extName
	} else {
		extName = "binary"
		filename = image.Checksum
	}
	targetDir = path.Join(r.dataFolder, image.Checksum[0:3])
	fullPath = path.Join(targetDir, filename)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "MkdirAll", err.Error()))
		return
	}
	image.ExtName = extName
	image.CreateTime = time.Now().In(app.TimeZone)
	_, err = os.Stat(fullPath)
	if err != nil {
		// if this file not exist .then download it .
		image.Status = ImageStatusStart
		go downloadImages(&image, fullPath, r.imageStore, r.config.CallBackUrl)
		err = r.imageStore.AddImage(&image)
		if err != nil {
			c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "AddImages", err.Error()))
			return
		}
		c.JSON(http.StatusCreated, app.ExportData(http.StatusCreated, "ok.", filename))
	} else {
		//if this file exist . then use it . and mark it succeed
		image.Status = ImageStatusDone
		image.UpdateTime = time.Now().In(app.TimeZone)
		c.JSON(http.StatusAlreadyReported, app.ExportData(http.StatusAlreadyReported, ".ok.", filename))
	}

}
