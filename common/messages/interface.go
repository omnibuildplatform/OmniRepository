package messages

type (
	Closeable interface {
		Close()
	}

	Notifier interface {
		Closeable
		Info(eventType, externalComponent, externalID string, data map[string]interface{})
	}
)
