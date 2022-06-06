package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/Shopify/sarama"
	"github.com/cloudevents/sdk-go/protocol/kafka_sarama/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/client"
)

const (
	Topic_DownloadStatus = "omni-repository-downloadStatus"
	GroupID              = "omni-repository"
)

var (
	saramaConfig     *sarama.Config
	brokers          []string
	cloudEventClient client.Client
)

func InitMQ() error {
	saramaConfig = sarama.NewConfig()
	saramaConfig.Version = sarama.V2_8_1_0
	brokers = strings.Split(AppConfig.MQ.KafkaBrokers, ",")
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	sender, err := kafka_sarama.NewSender(brokers, saramaConfig, Topic_DownloadStatus)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create protocol: %v", err.Error()))
	}
	cloudEventClient, err = cloudevents.NewClient(sender, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create cloudevents client, %v", err))
	}
	return nil
}

func RegisterEventLinstener() {

}

func PostDownloadStatusEvent(externalID, eventType, subject string, blockSize int64, totalSize int, sourceurl string) {
	e := cloudevents.NewEvent()
	e.SetSpecVersion(cloudevents.VersionV1)
	e.SetType(eventType)
	e.SetSubject(subject)
	e.SetSource(sourceurl)
	_ = e.SetData(cloudevents.ApplicationJSON, map[string]interface{}{
		"id":        externalID,
		"blockSize": blockSize,
		"totalSize": totalSize,
	})
	err := cloudEventClient.Send(kafka_sarama.WithMessageKey(context.Background(), sarama.StringEncoder(e.ID())), e)
	if err != nil {
		log.Fatalf("failed to PostDownloadStatusEvent ,error: %v \n", err)
	}
}
