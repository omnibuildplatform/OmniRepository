package workers

import (
	"context"
	"github.com/omnibuildplatform/omni-repository/common/models"
)

type (
	Closeable interface {
		Close()
	}

	Worker interface {
		Closeable
		DoWork(ctx context.Context) error
	}
)

type ImageWorkType string

const (
	PullImageWork  ImageWorkType = "PullImageWork"
	SignImageWork  ImageWorkType = "SignImageWork"
	PushImageWork  ImageWorkType = "PushImageWork"
	CleanImageWork ImageWorkType = "CleanImageWork"
)

type ImageWork struct {
	Image models.Image
	Type  ImageWorkType
}
