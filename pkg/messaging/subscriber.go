package messaging

import (
	"context"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
)

// MessageHandler processes a received message.
type MessageHandler func(msg *nats.Msg) error

// Subscriber subscribes to tenant-scoped NATS subjects.
type Subscriber struct {
	conn *Conn
	subs []*nats.Subscription
}

// NewSubscriber creates a new Subscriber.
func NewSubscriber(conn *Conn) *Subscriber {
	return &Subscriber{conn: conn}
}

// SubscribeAll subscribes to all tenant telemetry with a durable consumer.
func (s *Subscriber) SubscribeAll(ctx context.Context, consumerName string, handler MessageHandler) error {
	sub, err := s.conn.js.Subscribe(
		"tenant.*.telemetry.>",
		func(msg *nats.Msg) {
			if err := handler(msg); err != nil {
				log.Printf("message handler error: %v", err)
				_ = msg.Nak()
				return
			}
			_ = msg.Ack()
		},
		nats.Durable(consumerName),
		nats.ManualAck(),
		nats.AckWait(30*1000000000), // 30 seconds
		nats.MaxDeliver(3),
		nats.DeliverAll(),
	)
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}

	s.subs = append(s.subs, sub)

	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()

	return nil
}

// SubscribeTenant subscribes to a specific tenant's telemetry.
func (s *Subscriber) SubscribeTenant(ctx context.Context, tenantID string, consumerName string, handler MessageHandler) error {
	subject := fmt.Sprintf("tenant.%s.telemetry.>", tenantID)
	sub, err := s.conn.js.Subscribe(
		subject,
		func(msg *nats.Msg) {
			if err := handler(msg); err != nil {
				log.Printf("message handler error for tenant %s: %v", tenantID, err)
				_ = msg.Nak()
				return
			}
			_ = msg.Ack()
		},
		nats.Durable(consumerName),
		nats.ManualAck(),
		nats.AckWait(30*1000000000),
		nats.MaxDeliver(3),
	)
	if err != nil {
		return fmt.Errorf("subscribe tenant %s: %w", tenantID, err)
	}

	s.subs = append(s.subs, sub)

	go func() {
		<-ctx.Done()
		_ = sub.Unsubscribe()
	}()

	return nil
}

// Close unsubscribes from all subscriptions.
func (s *Subscriber) Close() {
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
}
