package runtime

import (
	"sync"
	"testing"
)

func TestTelemetryHubCloseDrainsBufferedEventsAndRejectsNewSubscribers(t *testing.T) {
	hub := NewTelemetryHub(2)
	subscriber := hub.Subscribe()
	hub.Publish(TelemetryEvent{Type: "committed"})
	hub.Close()
	hub.Close()
	hub.Publish(TelemetryEvent{Type: "after-close"})
	hub.Unsubscribe(subscriber)
	if event, open := <-subscriber.C(); !open || event.Type != "committed" {
		t.Fatal("close discarded a previously buffered event")
	}
	if _, open := <-subscriber.C(); open {
		t.Fatal("closed hub published another event")
	}
	if hub.Subscribe() != nil {
		t.Fatal("closed hub accepted new subscriber")
	}
}

func TestTelemetryBackpressureRetiresOnlySlowSubscriber(t *testing.T) {
	hub := NewTelemetryHub(1)
	defer hub.Close()
	slow, fast := hub.Subscribe(), hub.Subscribe()
	hub.Publish(TelemetryEvent{StateRevision: 1})
	<-fast.C()
	hub.Publish(TelemetryEvent{StateRevision: 2})
	if event, open := <-slow.C(); !open || event.StateRevision != 1 {
		t.Fatal("slow subscriber lost its buffered event")
	}
	if _, open := <-slow.C(); open {
		t.Fatal("overflowed subscriber stayed open")
	}
	if event, open := <-fast.C(); !open || event.StateRevision != 2 {
		t.Fatal("slow subscriber interrupted healthy subscriber")
	}
}

func TestTelemetryConcurrentPublishAndSubscriptionShutdown(t *testing.T) {
	hub := NewTelemetryHub(4)
	start := make(chan struct{})
	closeHub := make(chan struct{})
	var workers sync.WaitGroup
	for worker := 0; worker < 12; worker++ {
		workers.Add(1)
		go func(worker int) {
			defer workers.Done()
			<-start
			for revision := 0; revision < 100; revision++ {
				subscriber := hub.Subscribe()
				if worker%2 != 0 {
					hub.Unsubscribe(subscriber)
					subscriber = hub.SubscribeRevisions()
				}
				hub.Publish(TelemetryEvent{StateRevision: uint64(revision)})
				hub.Unsubscribe(subscriber)
				if worker == 0 && revision == 50 {
					close(closeHub)
				}
			}
		}(worker)
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-closeHub
		hub.Close()
	}()
	close(start)
	workers.Wait()
	if hub.Subscribe() != nil {
		t.Fatal("concurrent shutdown left hub open")
	}
}
