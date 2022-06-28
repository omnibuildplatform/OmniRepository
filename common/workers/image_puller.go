package workers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gookit/goutil/fsutil"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"github.com/omnibuildplatform/omni-repository/common/messages"
	"github.com/omnibuildplatform/omni-repository/common/models"
	"github.com/omnibuildplatform/omni-repository/common/storage"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const MaxTempFileSize = 100 * 1024 * 1024
const TempFolder = ".temp"
const UnReachableBlock = 100

type SingleBlock struct {
	Index      string
	StartIndex int64
	EndIndex   int64
	RetryCount int
}

type ImagePuller struct {
	ImageStore   *storage.ImageStorage
	Image        *models.Image
	LocalFolder  string
	Logger       *zap.Logger
	Client       http.Client
	BlockChannel chan SingleBlock
	Config       config.ImagePuller
	Worker       int
	ImageSize    int
	Notifier     messages.Notifier
}

func NewImagePuller(config config.ImagePuller, imageStore *storage.ImageStorage, logger *zap.Logger, image *models.Image, localFolder string, worker int, notifier messages.Notifier) (*ImagePuller, error) {
	client := http.Client{
		Timeout: 60 * 20 * time.Second,
	}
	return &ImagePuller{
		LocalFolder:  filepath.Dir(path.Join(localFolder, image.ImagePath)),
		Logger:       logger,
		ImageStore:   imageStore,
		Image:        image,
		Config:       config,
		Client:       client,
		BlockChannel: make(chan SingleBlock, 100),
		Worker:       worker,
		Notifier:     notifier,
	}, nil
}

func (r *ImagePuller) cleanup(err error) {
	blockTempFolder := path.Join(r.LocalFolder, TempFolder)
	_ = os.RemoveAll(blockTempFolder)
	r.Image.Status = models.ImageFailed
	r.Image.StatusDetail = err.Error()
	_ = r.ImageStore.UpdateImageStatusAndDetail(r.Image)

	//send failed message
	r.Notifier.NonBlockPush(string(models.ImageEventFailed), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{
		"detail": err.Error(),
	})

}

func (r *ImagePuller) DoWork(ctx context.Context) error {
	var err error
	// 1. prepare
	blockTempFolder := path.Join(r.LocalFolder, TempFolder)
	err = os.MkdirAll(blockTempFolder, fsutil.DefaultDirPerm)
	if err != nil {
		return err
	}
	r.Image.Status = models.ImageDownloading
	err = r.ImageStore.UpdateImageStatus(r.Image)
	if err != nil {
		return err
	}

	// 2. fetch object size
	// 3. split and download objects in parallel
	wg := sync.WaitGroup{}
	// used for total task recording
	var totalBlocks atomic.Int32
	//in case startWorkerLoop finished in advance of downloadPrepare
	totalBlocks.Add(UnReachableBlock)
	for i := 0; i < r.Worker; i++ {
		wg.Add(1)
		go r.startWorkerLoop(ctx, &wg, &totalBlocks)
	}
	wg.Add(1)
	size, err := r.downloadPrepare(ctx, &wg)
	if err != nil {
		close(r.BlockChannel)
		wg.Wait()
		r.cleanup(err)
		return err
	}
	r.Logger.Info(fmt.Sprintf("image %s will be downloaded in %d parts in parallel", r.Image.SourceUrl, size))
	totalBlocks.Add(int32(size))
	totalBlocks.Sub(UnReachableBlock)
	wg.Wait()
	files, err := ioutil.ReadDir(blockTempFolder)
	if len(files) != size {
		r.cleanup(err)
		return err
	}
	// 4. combine result
	err = r.ConstructImageFile()
	if err != nil {
		r.cleanup(err)
		return err
	}

	r.Logger.Info(fmt.Sprintf("image %s successfully created.", r.Image.SourceUrl))
	r.Image.Status = models.ImageDownloaded
	r.Image.StatusDetail = "image successfully downloaded"
	err = r.ImageStore.UpdateImageStatusAndDetail(r.Image)
	if err != nil {
		r.cleanup(err)
		return err
	}
	_ = os.RemoveAll(blockTempFolder)
	return nil
}

func (r *ImagePuller) ConstructImageFile() error {
	imagePath := path.Join(r.LocalFolder, r.Image.FileName)
	os.Remove(imagePath)
	out, err := os.OpenFile(imagePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	var partPaths []string
	err = filepath.Walk(path.Join(r.LocalFolder, TempFolder), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		//collect all files
		partPaths = append(partPaths, path)
		return nil
	})
	//path are started with index, we sort file before appending
	sort.Strings(partPaths)
	for _, p := range partPaths {
		handleError := func(temp string) error {
			f, err := os.OpenFile(temp, os.O_RDONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			n, err := io.Copy(out, f)
			if err != nil {
				return err
			}
			r.Logger.Info(fmt.Sprintf("write %d bytes from %s to file %s", n, path.Base(temp), imagePath))
			return nil
		}(p)
		if err != nil {
			return handleError
		}
	}
	return err
}

func (r *ImagePuller) Close() {
	close(r.BlockChannel)
}

func (r *ImagePuller) downloadPrepare(ctx context.Context, wg *sync.WaitGroup) (int, error) {
	defer wg.Done()
	rawUrl, err := url.Parse(r.Image.SourceUrl)
	if err != nil {
		return 0, err
	}
	if rawUrl.Scheme != "http" && rawUrl.Scheme != "https" {
		return 0, errors.New(fmt.Sprintf("source url schema not supported, %s", rawUrl.Scheme))
	}
	request, err := http.NewRequest("HEAD", r.Image.SourceUrl, nil)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("failed to construct request for source url, %s", r.Image.SourceUrl))
	}
	request = request.WithContext(ctx)
	// curl the pop star, we have to
	request.Header.Set("User-Agent", "curl")
	result, err := r.Client.Do(request)
	if err != nil {
		return 0, err
	}
	if result.StatusCode != http.StatusOK {
		return 0, errors.New(fmt.Sprintf("unacceptable status code %d when HEAD image meta %s",
			result.StatusCode,
			r.Image.SourceUrl))
	}
	if len(result.Header.Get("content-length")) == 0 {
		return 0, errors.New(fmt.Sprintf("unacceptable content type %s or content length empty %s image %s",
			result.Header.Get("content-type"),
			result.Header.Get("content-length"),
			r.Image.SourceUrl))
	}
	r.ImageSize, err = strconv.Atoi(result.Header.Get("content-length"))
	if err != nil {
		return 0, errors.New(fmt.Sprintf("unaccptable content length %s for image %s", result.Header.Get("content-length"), r.Image.SourceUrl))
	}

	var blocks []SingleBlock
	for start := 0; start <= r.ImageSize; start += MaxTempFileSize {
		endSize := start + MaxTempFileSize - 1
		if endSize > r.ImageSize-1 {
			endSize = r.ImageSize - 1
		}
		blocks = append(blocks, SingleBlock{
			StartIndex: int64(start),
			EndIndex:   int64(endSize),
			RetryCount: 1,
		})
	}
	totalBlocks := len(blocks)
	for index, b := range blocks {
		b.Index = fmt.Sprintf("%d/%d", index+1, totalBlocks)
		r.BlockChannel <- b
	}
	return len(blocks), nil
}

func (r *ImagePuller) startWorkerLoop(ctx context.Context, wg *sync.WaitGroup, totalBlocks *atomic.Int32) {
	defer wg.Done()
	finishTicker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-finishTicker.C:
			//NOTE: when all task finished no matter success or fail, break loop
			if totalBlocks.Load() == 0 {
				r.Logger.Info("image puller work finished")
				return
			}
		case block, ok := <-r.BlockChannel:
			if !ok {
				r.Logger.Info("image puller will quit")
				return
			}
			r.Logger.Info(fmt.Sprintf("starting to download block %s [%d, %d] for image %s",
				block.Index, block.StartIndex, block.EndIndex, r.Image.FileName))
			err := r.fetchSingleBlock(ctx, block)
			if err != nil {
				r.Logger.Error(fmt.Sprintf("Failed to download block %s [%d, %d] for image %s, error %v",
					block.Index, block.StartIndex, block.EndIndex, r.Image.FileName, err))
				if block.RetryCount <= r.Config.MaxRetry {
					r.Logger.Info(fmt.Sprintf("block %s [%d, %d] for image %s will have another try",
						block.Index, block.StartIndex, block.EndIndex, r.Image.FileName))
					block.RetryCount += 1
					r.BlockChannel <- block
				} else {
					r.Logger.Error(fmt.Sprintf("block %s [%d, %d] for image %s will reaches max retry. failed, error %v",
						block.Index, block.StartIndex, block.EndIndex, r.Image.FileName, err))
					totalBlocks.Sub(1)

				}
			} else {
				r.Notifier.NonBlockPush(string(models.ImageEventDownloaded), r.Image.ExternalComponent, r.Image.ExternalID, map[string]interface{}{
					"blockSize": block.EndIndex - block.StartIndex + 1,
					"imageSize": r.ImageSize,
				})
				totalBlocks.Sub(1)
			}
		}
	}
}

func (r *ImagePuller) fetchSingleBlock(ctx context.Context, block SingleBlock) error {
	//little hardcode here
	fileIndex, err := strconv.Atoi(strings.Split(block.Index, "/")[0])
	if err != nil {
		return errors.New(fmt.Sprintf("failed to get file index information from index attribute %s", block.Index))
	}
	fileName := path.Join(r.LocalFolder, TempFolder, fmt.Sprintf("%s-%d-%d", fmt.Sprintf("%06d", fileIndex), block.StartIndex, block.EndIndex))
	if fileInfo, err := os.Stat(fileName); err == nil {
		if fileInfo.Size() == block.EndIndex-block.StartIndex+1 {
			r.Logger.Info(fmt.Sprintf("block %s [%d, %d] for image %s already exists, skip downloading",
				block.Index, block.StartIndex, block.EndIndex, r.Image.FileName))
			return nil
		}
	}
	//delete it anyway
	r.Logger.Info(fmt.Sprintf("block %s [%d, %d] for image %s will be deleted due to block size mismatch",
		block.Index, block.StartIndex, block.EndIndex, r.Image.FileName))
	os.Remove(fileName)
	request, err := http.NewRequest("GET", r.Image.SourceUrl, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to construct request for source url, %s", r.Image.SourceUrl))
	}
	request = request.WithContext(ctx)
	// curl the pop star, we have to
	request.Header.Set("User-Agent", "curl")
	request.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", block.StartIndex, block.EndIndex))
	result, err := r.Client.Do(request)
	if err != nil {
		return err
	}
	defer result.Body.Close()
	blockFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer blockFile.Close()
	_, err = io.Copy(blockFile, result.Body)
	//validate new file size
	if info, err := os.Stat(fileName); err != nil {
		return err
	} else {
		if info.Size() != block.EndIndex-block.StartIndex+1 {
			return errors.New(fmt.Sprintf(
				"block %s [%d, %d] for image %s actually size %d not equal to request size %d",
				block.Index, block.StartIndex, block.EndIndex, r.Image.FileName, info.Size(),
				block.EndIndex-block.StartIndex+1))
		}
	}
	r.Logger.Info(fmt.Sprintf("block %s [%d, %d] for image %s has been successfully created",
		block.Index, block.StartIndex, block.EndIndex, r.Image.FileName))
	return nil
}
