package application

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gookit/color"
	"github.com/gookit/goutil/fsutil"
	"github.com/omnibuildplatform/OmniRepository/app"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type PackageType string

const (
	RPM       PackageType = "rpm"
	Image     PackageType = "image"
	Toolchain PackageType = "toolchain"
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
	return nil
}

func (r *RepositoryManager) checkToken(request *http.Request) error {
	token := request.URL.Query().Get("token")
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
		project  string
		dstFolder  string
		fileType string
	)
	if err := r.checkToken(c.Request); err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	//validate the metadata
	project = c.Request.FormValue("project")
	if len(project) == 0 {
		c.Data(http.StatusBadRequest, "text/html", []byte("missing project"))
		return
	}

	fileType = c.Request.FormValue("fileType")
	if len(fileType) == 0 {
		c.Data(http.StatusBadRequest, "text/html", []byte("missing file type"))
		return
	}
	if strings.ToLower(fileType) != string(Toolchain) && strings.ToLower(fileType) != string(
		RPM) && strings.ToLower(fileType) != string(Image) {
		c.Data(http.StatusBadRequest, "text/html", []byte("unacceptable file type, valid type are 'rpm', 'toolchain' or 'image'"))
		return
	}
	srcFile, info, err := c.Request.FormFile("file")
	defer srcFile.Close()
	filename := info.Filename
	if len(filename) == 0 {
		c.Data(http.StatusBadRequest, "text/html", []byte("missing file type"))
		return
	}
	if err != nil {
		return
	}
	if strings.ToLower(fileType) == string(Image) {
		dstFolder = path.Join(r.dataFolder, project, time.Now().Format("2006-01-02"))
	} else if strings.ToLower(fileType) == string(RPM) {
		dstFolder = path.Join(r.dataFolder, project, "source")
	} else {
		dstFolder = path.Join(r.dataFolder, project, "toolchain")
	}
	err = os.MkdirAll(dstFolder, os.ModePerm)
	if err != nil {
		c.Data(http.StatusBadRequest, "text/html", []byte(err.Error()))
		return
	}
	dstFile, err := os.OpenFile(path.Join(dstFolder, filename), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		c.Data(http.StatusBadRequest, "text/html", []byte(err.Error()))
		return
	}
	defer dstFile.Close()
	//TODO: read & write in chunk?
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return
	}
	//TODO: Sign the content
	rel, _ := filepath.Rel(r.dataFolder, path.Join(dstFolder, filename))
	c.JSON(http.StatusCreated,
		UploadFilePath{
			Path: fmt.Sprintf("http://%s/data/browse/%s", r.serverName, rel),
		})
}

func (r *RepositoryManager) StartLoop() {
}

func (r *RepositoryManager) Close() {

}
