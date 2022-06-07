package messages

type (
	Closeable interface {
		Close()
	}

	Notifier interface {
		Closeable
		NonBlockPush(eventType, externalComponent, externalID string, data map[string]interface{})
	}
)
