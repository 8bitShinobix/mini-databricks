package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

const maxRetries = 3

type Consumer struct {
	reader   *kafka.Reader
	producer *Producer
	dlqTopic string
}

func NewConsumer(brokers, topic, groupID string) *Consumer {
	dlqTopic := topic + ".deadletter"
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{brokers},
			Topic:   topic,
			GroupID: groupID,
		}),
		producer: NewProducer(brokers),
		dlqTopic: dlqTopic,
	}
}

func (c *Consumer) Consume(ctx context.Context, handler func(ctx context.Context, payload []byte) error) {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				slog.Info("consumer shutting down")
				return
			}
			slog.Error("error reading message", "error", err)
			continue
		}
		slog.Info("received event", "payload", string(msg.Value))

		// restore trace context from Kafka headers so downstream spans
		// attach to the original request trace instead of starting fresh
		msgCtx := ctx
		for _, h := range msg.Headers {
			if h.Key == "traceparent" {
				carrier := propagation.MapCarrier{"traceparent": string(h.Value)}
				msgCtx = otel.GetTextMapPropagator().Extract(ctx, carrier)
				break
			}
		}

		var lastErr error
		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := handler(msgCtx, msg.Value); err != nil { // ← pass msgCtx not ctx
				lastErr = err
				if attempt < maxRetries {
					backoff := time.Duration(attempt) * time.Second
					slog.Warn("handler failed, retrying",
						"attempt", attempt,
						"max_retries", maxRetries,
						"backoff", backoff,
						"error", err,
					)
					select {
					case <-time.After(backoff):
					case <-ctx.Done():
						return
					}
				}
			} else {
				lastErr = nil
				break
			}
		}

		if lastErr != nil {
			slog.Error("message failed after retries, sending to DLQ",
				"max_retries", maxRetries,
				"dlq_topic", c.dlqTopic,
				"error", lastErr,
			)
			if err := c.producer.Publish(ctx, c.dlqTopic, string(msg.Key), map[string]string{
				"original_payload": string(msg.Value),
				"error":            lastErr.Error(),
				"topic":            msg.Topic,
			}); err != nil {
				slog.Error("failed to publish to DLQ", "error", err)
			}
		}
	}
}

func (c *Consumer) Close() error {
	c.producer.Close()
	return c.reader.Close()
}

func ParsePayload[T any](data []byte) (T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("failed to parse payload: %w", err)
	}
	return result, nil
}
