package mq

import (
	"context"
	"testing"
	"time"

	"github.com/project-horizon/horizon-core/internal/model"
)

func TestMemoryBrokerPublishSubscribe(t *testing.T) {
	b := NewMemoryBroker()
	ctx := context.Background()
	var got model.Event
	if err := b.Subscribe(ctx, "devices", func(_ context.Context, e model.Event) error { got = e; return nil }); err != nil {
		t.Fatal(err)
	}
	want := model.Event{ID: "evt-1", Type: "device.online", OccurredAt: time.Now()}
	if err := b.Publish(ctx, "devices", want); err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID || got.Type != want.Type {
		t.Fatalf("got %+v want %+v", got, want)
	}
}
