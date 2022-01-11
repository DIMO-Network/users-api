package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/Shopify/sarama"
	"github.com/rs/zerolog"
	"github.com/segmentio/ksuid"
)

type EventService struct {
	Settings *config.Settings
	Logger   *zerolog.Logger
	Producer sarama.SyncProducer
}

type cloudEventMessage struct {
	ID          string      `json:"id"`
	Source      string      `json:"source"`
	SpecVersion string      `json:"specversion"`
	Subject     string      `json:"subject"`
	Time        time.Time   `json:"time"`
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
}

func (e *EventService) Emit(eventType, subject string, data interface{}) error {
	msgBytes, err := json.Marshal(cloudEventMessage{
		ID:          ksuid.New().String(),
		Source:      "users-api",
		SpecVersion: "1.0",
		Subject:     subject,
		Time:        time.Now(),
		Type:        eventType,
		Data:        data,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal CloudEvent: %w", err)
	}
	msg := &sarama.ProducerMessage{
		Topic: e.Settings.EventsTopic,
		Value: sarama.ByteEncoder(msgBytes),
	}
	_, _, err = e.Producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to produce CloudEvent to Kafka: %w", err)
	}
	return nil
}

func NewEventService(logger *zerolog.Logger, settings *config.Settings) *EventService {
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.Return.Successes = true
	producer, err := sarama.NewSyncProducer(strings.Split(settings.KafkaBrokers, ","), kafkaConfig)
	if err != nil {
		panic(err)
	}
	return &EventService{
		Settings: settings,
		Logger:   logger,
		Producer: producer,
	}
}
