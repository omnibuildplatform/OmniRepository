package messages

import (
	"context"
	"errors"
	"fmt"
	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
	"github.com/omnibuildplatform/omni-repository/common/config"
	"go.uber.org/zap"
	"strings"
)

const (
	TopicImageStatus = "omni-repository-image-status"
	SourceUrl        = "github.com/omnibuildplatform/omni-repository"
)

type CloudEventNotifier struct {
	logger           *zap.Logger
	config           config.MQ
	cloudEventClient client.Client
	sender           *kafka_sarama.Sender
}

func NewCloudEventNotifier(config config.MQ, logger *zap.Logger) (Notifier, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_1_0
	brokers := strings.Split(config.KafkaBrokers, ",")
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	sender, err := kafka_sarama.NewSender(brokers, saramaConfig, TopicImageStatus)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create protocol: %v", err.Error()))
	}
	cloudEventClient, err := cloudevents.NewClient(sender, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return nil, errors.New(fmt.Sprintf("failed to create cloudevents client, %v", err))
	}
	return &CloudEventNotifier{
		logger:           logger,
		config:           config,
		cloudEventClient: cloudEventClient,
		sender:           sender,
	}, nil
}

func (n *CloudEventNotifier) NonBlockPush(eventType, externalComponent, externalID string, data map[string]interface{}) {
	subject := fmt.Sprintf("%s.%s", externalComponent, externalID)
	e := cloudevents.NewEvent()
	e.SetSpecVersion(cloudevents.VersionV1)
	e.SetType(eventType)
	e.SetSubject(subject)
	e.SetSource(SourceUrl)
	_ = e.SetData(cloudevents.ApplicationJSON, data)
	go func() {
		err := n.cloudEventClient.Send(kafka_sarama.WithMessageKey(context.Background(), sarama.StringEncoder(e.ID())), e)
		if err != nil {
			n.logger.Error(fmt.Sprintf("[CloudEventNotifier] failed to send message ,error: %v", err))
		}
		n.logger.Info(fmt.Sprintf("[CloudEventNotifier] message send with event type %s, subject %s data %v", eventType, subject, data))
	}()
}

func (n *CloudEventNotifier) Close() {
	if n.sender != nil {
		n.sender.Close(context.TODO())
	}
}
