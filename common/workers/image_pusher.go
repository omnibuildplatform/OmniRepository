package workers

import (
	"context"
	"github.com/omnibuildplatform/omni-repository/common/storage"
)

type ImagePusher struct {
	imageStore *storage.ImageStorage
}

func (r *ImagePusher) DoWork(ctx context.Context) error {
	return nil
}

func (r *ImagePusher) Close() error {
	return nil
}
