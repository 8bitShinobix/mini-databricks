package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	dbgen "github.com/8bitShinobix/mini-databricks/internal/db/generated"
	"github.com/segmentio/kafka-go"
)

type Dispatcher struct {
	queries  *dbgen.Queries
	producer *Producer
}

func NewDispatcher(queries *dbgen.Queries, producer *Producer) *Dispatcher {
	return &Dispatcher{queries: queries, producer: producer}
}

func (d *Dispatcher) Start(ctx context.Context) {
	slog.Info("outbox dispatcher starting")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("dispatcher shutting down")
			return
		case <-ticker.C:
			if err := d.dispatch(ctx); err != nil {
				slog.Error("dispatch error", "error", err)
			}
		}
	}
}

func (d *Dispatcher) dispatch(ctx context.Context) error {
	events, err := d.queries.GetPendingOutboxEvents(ctx)
	if err != nil {
		return err
	}
	for _, event := range events {
		// extract traceparent saved by job_service so the scheduler
		// can continue the same trace across the Kafka boundary
		var raw map[string]string
		var headers []kafka.Header
		if err := json.Unmarshal(event.Payload, &raw); err == nil {
			if tp, ok := raw["traceparent"]; ok && tp != "" {
				headers = append(headers, kafka.Header{
					Key:   "traceparent",
					Value: []byte(tp),
				})
			}
		}

		if err := d.producer.Publish(ctx, event.EventType, event.AggregateID.String(), event.Payload, headers...); err != nil {
			slog.Error("failed to publish event", "event_id", event.ID, "error", err)
			d.queries.MarkOutboxFailed(ctx, event.ID)
			continue
		}
		d.queries.MarkOutboxDelivered(ctx, event.ID)
		slog.Info("dispatched event", "event_id", event.ID, "event_type", event.EventType)
	}
	return nil
}
