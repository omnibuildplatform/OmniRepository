package workers

import "context"

type (
	Closeable interface {
		Close()
	}

	Worker interface {
		Closeable
		DoWork(ctx context.Context) error
	}
)
