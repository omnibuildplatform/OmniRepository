package application

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
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
	RPM                   PackageType = "rpm"
	Image                 PackageType = "image"
	Toolchain             PackageType = "toolchain"
	BuildImageFromRelease string      = "buildimagefromrelease"
	BuildImageFromImages  string      = "buildimagefromimages"
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
	if token == "" || token != r.uploadToken {
		return errors.New("token mismatch")
	}
	return nil
}

func (r *RepositoryManager) Upload(c *gin.Context) {

	var (
		userimage app.UserImages
	)
	if err := r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "checkToken", err.Error()))
		return
	}

	err := c.MustBindWith(&userimage, binding.FormMultipart)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "BindQuery", err.Error()))
		return
	}
	srcFile, fileinfo, err := c.Request.FormFile("file")
	defer srcFile.Close()
	srcFileBody, err := ioutil.ReadAll(srcFile)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "responseBody ReadAll", err.Error()))
		return
	}
	checksumValue := fmt.Sprintf("%X", sha256.Sum256(srcFileBody))
	if userimage.Checksum != checksumValue {
		c.JSON(http.StatusConflict, app.ExportData(http.StatusConflict, "file's sha256SUM not equal input checkSum ", checksumValue))
		return
	}

	var targetDir, fullPath, filename, extName string

	if strings.Contains(fileinfo.Filename, ".") {
		extName = strings.Split(fileinfo.Filename, ".")[1]
		if strings.Contains(extName, "?") {
			extName = strings.Split(extName, "?")[0]
		}
		if strings.Contains(extName, "#") {
			extName = strings.Split(extName, "#")[0]
		}
		if strings.Contains(extName, "&") {
			extName = strings.Split(extName, "&")[0]
		}
		targetDir = path.Join(r.dataFolder, extName)
		filename = userimage.Checksum + "." + extName
	} else {
		extName = "binary"
		targetDir = path.Join(r.dataFolder, extName)
		filename = userimage.Checksum
	}
	fullPath = path.Join(targetDir, filename)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(500, "MkdirAll", err.Error()))
		return
	}

	_, err = os.Stat(fullPath)
	if err == nil {
		c.JSON(http.StatusConflict, app.ExportData(http.StatusConflict, "file exist", filename))
		return
	}

	dstFile, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(500, "OpenFile", err.Error()))
		return
	}

	defer dstFile.Close()
	//TODO: read & write in chunk?
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		c.JSON(http.StatusInternalServerError, app.ExportData(500, "Copy", err.Error()))
		return
	}
	userimage.ExtName = extName
	err = app.AddUserImages(&userimage)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(500, "AddUserImages", err.Error()))
		return
	}
	c.JSON(http.StatusOK, app.ExportData(200, "ok", userimage))
}

func (r *RepositoryManager) StartLoop() {
}

func (r *RepositoryManager) Close() {

}

func (r *RepositoryManager) Query(c *gin.Context) {
	if err := r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "forbidden", err.Error()))
		return
	}
	externalID := c.Query("externalID")
	if len(externalID) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "FormValue", "missing externalID"))
		return
	}
	item, err := app.GetUserImagesByExternalID(externalID)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "error", err.Error()))
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
		userimage app.UserImages
		isoUrl    string
		err       error
	)
	if err = r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "forbidden", err.Error()))
		return
	}
	isoUrl = c.Query("url")
	if len(isoUrl) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "FormValue", "missing url"))
		return
	}
	userimage.FromURL = isoUrl
	userimage.UserId, _ = strconv.Atoi(c.Query("userid"))
	if userimage.UserId <= 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "FormValue", "missing userid"))
		return
	}

	userimage.Checksum = strings.ToUpper(c.Query("checksum"))
	if len(userimage.Checksum) == 0 {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "FormValue", "missing checksum"))
		return
	}

	var fullPath, extName string
	_, filename := path.Split(isoUrl)
	targetDir := r.dataFolder
	if strings.Contains(filename, ".") {
		extName = strings.Split(filename, ".")[1]
		if strings.Contains(extName, "?") {
			extName = strings.Split(extName, "?")[0]
		}
		if strings.Contains(extName, "#") {
			extName = strings.Split(extName, "#")[0]
		}
		if strings.Contains(extName, "&") {
			extName = strings.Split(extName, "&")[0]
		}
		filename = userimage.Checksum + "." + extName
	} else {
		extName = "binary"
		filename = userimage.Checksum
	}
	targetDir = path.Join(r.dataFolder, extName)
	fullPath = path.Join(targetDir, filename)
	err = os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "MkdirAll", err.Error()))
		return
	}
	userimage.ExtName = extName
	_, err = os.Stat(fullPath)
	if err == nil {
		c.JSON(http.StatusConflict, app.ExportData(http.StatusConflict, "file exist", filename))
		return
	}

	//---------start download  file-----------
	var response *http.Response
	response, err = http.Get(isoUrl)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "Get", err.Error()))
		return
	}
	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "responseBody ReadAll", err.Error()))
		return
	}

	checksumValue := fmt.Sprintf("%X", sha256.Sum256(responseBody))
	if userimage.Checksum != checksumValue {
		c.JSON(http.StatusConflict, app.ExportData(http.StatusConflict, "file's md5 not equal checkSum ", checksumValue))
		return
	}

	err = ioutil.WriteFile(fullPath, responseBody, os.ModePerm)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(400, "WriteFile", err.Error()))
		return
	}
	userimage.CreateTime = time.Now().In(app.CnTime)
	err = app.AddUserImages(&userimage)
	if err != nil {
		c.JSON(http.StatusBadRequest, app.ExportData(500, "AddUserImages", err.Error()))
		return
	}
	c.JSON(http.StatusOK, app.ExportData(200, "ok", filename))
}
