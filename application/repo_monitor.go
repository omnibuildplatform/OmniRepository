package application

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/omnibuildplatform/OmniRepository/app"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type RepoMonitor struct {
	app.UnimplementedRepoServerServer
}

func (s *RepoMonitor) CallLoadFrom(ctx context.Context, in *app.RepRequest) (*app.RepResponse, error) {

	return nil, status.Errorf(codes.Unimplemented, "method CallLoadFrom not implemented")

}

func (r *RepoMonitor) SyncImageStatus() {
	for {

		time.Sleep(time.Second * 30)
	}
}

func downLoadImages(image *app.Images, fullPath string) {
	image.Status = ImageStatusStart
	defer func() {
		// update the image status at last
		image.UpdateTime = time.Now().In(app.CnTime)
		err := app.UpdateImagesStatus(image)
		if err != nil {
			app.Logger.Error(fmt.Sprintf("UpdateImagesStatus id:[%d] ,status:[%s],sourceurl:[%s] Error:%s", image.ID, image.Status, image.SourceUrl, err))
		}
	}()
	var err error
	var response *http.Response
	response, err = http.Get(image.SourceUrl)
	if err != nil {
		image.Status = ImageStatusFailed
		return
	}
	defer response.Body.Close()
	var savefile *os.File
	savefile, err = os.Create(fullPath)
	defer savefile.Close()

	_, err = io.Copy(savefile, response.Body)
	if err != nil {
		image.Status = ImageStatusFailed
		return
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, savefile); err != nil {
		image.Status = ImageStatusFailed
		return
	}

	checksumValue := fmt.Sprintf("%X", hash.Sum(nil))
	fmt.Println(checksumValue)
	if image.Checksum != checksumValue {
		err = fmt.Errorf("file's md5 not equal checkSum ")
		image.Status = ImageStatusFailed
		return
	}
	image.Status = ImageStatusDone
	return
}
