package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers),
			Balancer: &kafka.LeastBytes{},
		},
	}
}

// Publish sends a message. Pass headers to carry trace context.
func (p *Producer) Publish(ctx context.Context, topic string, key string, payload any, headers ...kafka.Header) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Topic:   topic,
		Key:     []byte(key),
		Value:   data,
		Headers: headers,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func CreateTopic(brokers, topic string) error {
	conn, err := kafka.Dial("tcp", brokers)
	if err != nil {
		return fmt.Errorf("failed to connect to kafka: %w", err)
	}
	defer conn.Close()
	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             topic,
			NumPartitions:     3,
			ReplicationFactor: 1,
		},
	}
	err = conn.CreateTopics(topicConfigs...)
	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}
	return nil
}
