package kafka

import (
	"context"
	"encoding/json"
	"log"
	"todolist/internal/config"
	"todolist/internal/event"

	"github.com/segmentio/kafka-go"
)

type Producer interface {
	PublishEvent(ctx context.Context, evt event.TodoEvent) error
	Close() error
}

type producer struct {
	writer *kafka.Writer
}

func NewProducer(cfg *config.Config) Producer {
	return &producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.KafkaBrokers),
			Topic:    cfg.KafkaTopic,
			Balancer: &kafka.LeastBytes{},
		},
	}
}

func (p *producer) PublishEvent(ctx context.Context, evt event.TodoEvent) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	err = p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(evt.Type),
		Value: data,
	})
	if err != nil {
		log.Printf("kafka publish error: %v", err)
	}
	return err
}

func (p *producer) Close() error {
	return p.writer.Close()
}
