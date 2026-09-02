// Package mq defines message queue abstractions and an in-memory pub/sub implementation.
package mq

import (
	"context"
	"sync"

	"github.com/project-horizon/horizon-core/internal/model"
)

type Handler func(context.Context, model.Event) error
type Publisher interface {
	Publish(context.Context, string, model.Event) error
}
type Subscriber interface {
	Subscribe(context.Context, string, Handler) error
}
type Broker interface {
	Publisher
	Subscriber
	Close() error
}
type MemoryBroker struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewMemoryBroker() *MemoryBroker { return &MemoryBroker{handlers: map[string][]Handler{}} }
func (b *MemoryBroker) Publish(ctx context.Context, topic string, evt model.Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.RLock()
	hs := append([]Handler(nil), b.handlers[topic]...)
	b.mu.RUnlock()
	for _, h := range hs {
		if err := h(ctx, evt); err != nil {
			return err
		}
	}
	return nil
}
func (b *MemoryBroker) Subscribe(ctx context.Context, topic string, h Handler) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	b.handlers[topic] = append(b.handlers[topic], h)
	b.mu.Unlock()
	return nil
}
func (b *MemoryBroker) Close() error {
	b.mu.Lock()
	b.handlers = map[string][]Handler{}
	b.mu.Unlock()
	return nil
}
