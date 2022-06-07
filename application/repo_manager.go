package application

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/gookit/color"
	"github.com/gookit/goutil/fsutil"
	"github.com/omnibuildplatform/omni-repository/app"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"github.com/omnibuildplatform/omni-repository/common/dtos"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"go.uber.org/zap"
)

const BROWSE_PREFIX = "/browse"

type PackageType string

type UploadFilePath struct {
	Path string
}

type RepositoryManager struct {
	Context             context.Context
	dataFolder          string
	publicRouterGroup   *gin.RouterGroup
	internalRouterGroup *gin.RouterGroup
	imageStore          *storage.ImageStorage
	config              config.RepoManager
	paraValidator       *validator.Validate
	imageDto            *dtos.ImageDTO
	Logger              *zap.Logger
}

func NewRepositoryManager(ctx context.Context, config config.RepoManager, publicRouterGroup *gin.RouterGroup, internalRouterGroup *gin.RouterGroup, imageStore *storage.ImageStorage, baseFolder string, logger *zap.Logger) (*RepositoryManager, error) {
	if !fsutil.DirExist(baseFolder) {
		color.Error.Println("data folder %s not existed", baseFolder)
		return nil, errors.New("data folder not existed")
	}
	return &RepositoryManager{
		Context:             ctx,
		dataFolder:          baseFolder,
		publicRouterGroup:   publicRouterGroup,
		internalRouterGroup: internalRouterGroup,
		imageStore:          imageStore,
		config:              config,
		imageDto:            dtos.NewImageDTO(BROWSE_PREFIX),
		paraValidator:       validator.New(),
		Logger:              logger,
	}, nil
}

func (r *RepositoryManager) Initialize() error {
	// register for public routes
	r.publicRouterGroup.Static(BROWSE_PREFIX, r.dataFolder)
	r.publicRouterGroup.GET("/images/query", r.Query)
	// register for internal routes
	r.internalRouterGroup.Static(BROWSE_PREFIX, r.dataFolder)
	r.internalRouterGroup.GET("/images/query", r.Query)
	r.internalRouterGroup.POST("/images/upload", r.Upload)
	r.internalRouterGroup.POST("/images/load", r.Load)
	return nil
}

// @BasePath /images/

// Upload godoc
// @Summary upload a image
// @Param body body dtos.ImageRequest true "body for upload a image"
// @Description Upload a image with specified parameter
// @Tags Image
// @Accept json
// @Produce json
// @Success 201 object models.Image
// @Router /upload [post]
func (r *RepositoryManager) Upload(c *gin.Context) {

	var imageRequest dtos.ImageRequest

	err := c.MustBindWith(&imageRequest, binding.FormMultipart)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to get upload request %v", err))
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "MustBindWith", err.Error()))
		return

	}

	err = r.paraValidator.Struct(imageRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	image := r.imageDto.GetImageFromRequest(imageRequest)
	if err := r.validCheckSum(image.Checksum, image.Algorithm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	image.ImagePath = path.Join(GetImageRelativeFolder(&image), image.FileName)
	image.ChecksumPath = path.Join(GetImageRelativeFolder(&image),
		fmt.Sprintf("%s.%ssum", image.Name, strings.ToLower(image.Algorithm)))
	image.Status = models.ImageDownloaded
	err = r.imageStore.AddImage(&image)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to save data into database %v", err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to save data into database"})
		return
	}

	srcFile, _, err := c.Request.FormFile("file")
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to get file from upload request %v", err))
		c.JSON(http.StatusBadRequest, app.ExportData(http.StatusBadRequest, "FormFile", err.Error()))
		return
	}
	defer srcFile.Close()

	localFile := path.Join(r.dataFolder, GetImageRelativeFolder(&image), image.FileName)

	err = os.MkdirAll(path.Dir(localFile), fsutil.DefaultDirPerm)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to create local folder for image %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create store folder for image"})
		return
	}

	dstFile, err := os.OpenFile(localFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		r.Logger.Error(fmt.Sprintf("failed to create local file for image %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create store file for image"})
		return
	}

	defer dstFile.Close()
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		r.Logger.Error(fmt.Sprintf("failed to copy image image %v", err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to copy image content into local"})
		return
	}
	c.JSON(http.StatusCreated, r.imageDto.GenerateResponseFromImage(image))
}

func (r *RepositoryManager) StartLoop() {
}

func (r *RepositoryManager) Close() {
	//todo :

}

// @BasePath /images/

// Query godoc
// @Summary query image by external ID
// @Param externalID query  string	true	"externalID"
// @Description Upload a image with specified parameter
// @Tags Image
// @Accept json
// @Produce json
// @Success 200 object models.Image
// @Router /query [post]
func (r *RepositoryManager) Query(c *gin.Context) {
	var queryImageRequest dtos.QueryImageRequest

	var err error
	if err = c.ShouldBindQuery(&queryImageRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = r.paraValidator.Struct(queryImageRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := r.imageStore.GetImageByExternalID(queryImageRequest.ExternalID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "image not found by this externalID"})
		return
	}
	c.JSON(http.StatusOK, r.imageDto.GenerateResponseFromImage(item))
	return
}

func (r *RepositoryManager) validCheckSum(checksum string, algorithm string) error {
	if strings.ToLower(algorithm) == "md5" {
		match, err := regexp.MatchString("^[a-fA-F0-9]{32}$", checksum)
		if err != nil {
			return err
		} else if !match {
			return errors.New("invalid md5 checksum")
		}
		return nil
	} else if strings.ToLower(algorithm) == "sha256" {
		match, err := regexp.MatchString("^[a-fA-F0-9]{64}$", checksum)
		if err != nil {
			return err
		} else if !match {
			return errors.New("invalid sha256 checksum")
		}
		return nil
	}
	return errors.New(fmt.Sprintf("unsupported algorithm %s", algorithm))
}

// @BasePath /images/

// Load godoc
// @Summary create a image from external system
// @Param body body dtos.ImageRequest true "body for upload a image"
// @Description create a image with specified parameter, image will be downloaded via source url
// @Tags Image
// @Accept json
// @Produce json
// @Success 201 object models.Image
// @Router /load [post]
func (r *RepositoryManager) Load(c *gin.Context) {
	var imageRequest dtos.ImageRequest

	var err error
	if err = c.ShouldBindJSON(&imageRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ShouldBindJSON error": err.Error()})
		return
	}
	err = r.paraValidator.Struct(imageRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"paraValidator error": err.Error()})
		return
	}
	if len(imageRequest.SourceUrl) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source url is empty"})
		return
	}

	image := r.imageDto.GetImageFromRequest(imageRequest)
	//TODO: use custom validator instead
	if err := r.validCheckSum(image.Checksum, image.Algorithm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"validCheckSum error": err.Error()})
		return
	}
	//calculate image relative path
	image.ImagePath = path.Join(GetImageRelativeFolder(&image), image.FileName)
	image.ChecksumPath = path.Join(GetImageRelativeFolder(&image),
		fmt.Sprintf("%s.%ssum", image.Name, strings.ToLower(image.Algorithm)))
	if existed, err := r.imageStore.GetImageByChecksum(image.Checksum); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"GetImageByChecksum error": fmt.Sprintf("image has identical checksum already existed %s",
			existed.FileName)})
		return
	}
	err = r.imageStore.AddImage(&image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"AddImage error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, image)

}
