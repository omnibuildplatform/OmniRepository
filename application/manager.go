package application

import (
	"crypto/sha256"
	"errors"
	"fmt"
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
	"github.com/omnibuildplatform/OmniRepository/app"
)

type PackageType string

const (
	RPM                    PackageType = "rpm"
	Image                  PackageType = "image"
	Toolchain              PackageType = "toolchain"
	BuildImageFromRelease  string      = "buildimagefromrelease"
	BuildImageFromISO      string      = "buildimagefromiso"
	ImageStatusStart       string      = "start"
	ImageStatusDownloading string      = "downloading"
	ImageStatusDone        string      = "done"
	ImageStatusFailed      string      = "failed"
)

type UploadFilePath struct {
	Path string
}

type RepositoryManager struct {
	dataFolder  string
	routerGroup *gin.RouterGroup
	uploadToken string
	serverName  string
}

func NewRepositoryManager(routerGroup *gin.RouterGroup) (*RepositoryManager, error) {

	conf := app.Config.StringMap("manager")
	baseFolder := conf["dataFolder"]
	if !fsutil.DirExist(baseFolder) {
		color.Error.Printf("data folder %s not existed", baseFolder)
		return nil, errors.New("data folder not existed")
	}
	token := conf["uploadToken"]
	tokenEnv := os.Getenv("UPLOAD_TOKEN")
	if len(tokenEnv) != 0 {
		token = tokenEnv
	}
	if len(token) == 0 {
		color.Error.Printf("upload token is empty")
		return nil, errors.New("upload token is empty")
	}

	serverName := app.Config.String("serverName")
	serverNameEnv := os.Getenv("Server_Name")
	if len(serverNameEnv) != 0 {
		serverName = serverNameEnv
	}
	if len(serverName) == 0 {
		serverName = "localhost"
	}
	return &RepositoryManager{
		dataFolder:  baseFolder,
		routerGroup: routerGroup,
		uploadToken: token,
		serverName:  serverName,
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
		token = request.FormValue("token")
	}
	if token == "" {
		token = request.URL.Query().Get("token")
	}
	if token == "" || token != r.uploadToken {
		return errors.New("token mismatch:" + r.uploadToken)
	}
	return nil
}

func (r *RepositoryManager) Upload(c *gin.Context) {

	var (
		image                                  app.Images
		targetDir, fullPath, filename, extName string
	)
	if err := r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "checkToken", err.Error()))
		return
	}
	err := c.MustBindWith(&image, binding.FormMultipart)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "BindQuery", err.Error()))
		return
	}
	srcFile, fileinfo, err := c.Request.FormFile("file")
	defer srcFile.Close()

	if strings.Contains(fileinfo.Filename, ".") {
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
	targetDir = path.Join(r.dataFolder, extName)
	fullPath = path.Join(targetDir, filename)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "MkdirAll", err.Error()))
		return
	}

	_, err = os.Stat(fullPath)
	if err == nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "file exist", filename))
		return
	}

	dstFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "OpenFile", err.Error()))
		return
	}

	defer dstFile.Close()
	//TODO: read & write in chunk?
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "Copy", err.Error()))
		return
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, dstFile); err != nil {
		image.Status = ImageStatusFailed
		return
	}
	checksumValue := fmt.Sprintf("%X", hash.Sum(nil))
	if image.Checksum != checksumValue {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "file's sha256sum not equal input checkSum ", checksumValue))
		return
	}

	image.ExtName = extName
	image.Status = ImageStatusDone
	err = app.AddImages(&image)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "AddUserImages", err.Error()))
		return
	}
	c.JSON(http.StatusCreated, app.ExportData(http.StatusCreated, "ok", image))
}

func (r *RepositoryManager) StartLoop() {
}

func (r *RepositoryManager) Close() {

}

func (r *RepositoryManager) Query(c *gin.Context) {
	if err := r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "forbidden", err.Error()))
		return
	}
	externalID := c.Query("externalID")
	if len(externalID) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "bad request", "missing externalID"))
		return
	}
	item, err := app.GetImagesByExternalID(externalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "error", err.Error()))
		return
	}
	downloadURL := "/data/browse/" + item.ExtName + "/" + item.Checksum
	if item.ExtName != "binary" {
		downloadURL = downloadURL + "." + item.ExtName
	}
	location := url.URL{Path: downloadURL}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func (r *RepositoryManager) LoadFrom(c *gin.Context) {
	var (
		image app.Images
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
	image.Checksum = strings.ToUpper(c.Query("checksum"))
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
	targetDir = path.Join(r.dataFolder, extName)
	fullPath = path.Join(targetDir, filename)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "MkdirAll", err.Error()))
		return
	}
	image.ExtName = extName
	_, err = os.Stat(fullPath)
	if err == nil {
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "file exist", filename))
		return
	}
	image.CreateTime = time.Now().In(app.CnTime)
	image.Status = ImageStatusStart
	err = app.AddImages(&image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(http.StatusInternalServerError, "AddImages", err.Error()))
		return
	}
	//---------start download  file-----------

	go downLoadImages(&image, fullPath)
	c.JSON(http.StatusCreated, app.ExportData(http.StatusCreated, "ok", filename))
}
