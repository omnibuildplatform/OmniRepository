package messages

import (
	"fmt"
	"go.uber.org/zap"
)

type EchoNotifier struct {
	logger *zap.Logger
}

func NewEchoNotifier(logger *zap.Logger) (Notifier, error) {
	return &EchoNotifier{
		logger: logger,
	}, nil
}

func (n *EchoNotifier) Info(eventType, externalComponent, externalID string, data map[string]interface{}) {
	subject := fmt.Sprintf("%s.%s", externalComponent, externalID)
	n.logger.Info(fmt.Sprintf("[EchoNotifier] message send with event type %s, subject %s data %v", eventType, subject, data))
}

func (n *EchoNotifier) Close() {
	//Do nothing
}
