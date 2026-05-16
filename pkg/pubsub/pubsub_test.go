package pubsub

import (
	"fmt"
	"testing"
	"time"
)

func TestPubSubSingleSubscriber(t *testing.T) {
	t.Parallel()

	ps := New(2)
	ch, err := ps.Subscribe("user", "topic")
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"a":1}`
	go ps.Publish("user", "topic", []byte(expected))

	out := receive(t, ch)
	if string(out.Data) != expected {
		t.Errorf("want: %s, got: %s", expected, string(out.Data))
	}
}

func TestPubSubMultiSubscribers(t *testing.T) {
	t.Parallel()

	ps := New(3)
	ch1, err := ps.Subscribe("user1", "topic")
	if err != nil {
		t.Fatal(err)
	}
	ch2, err := ps.Subscribe("user2", "topic")
	if err != nil {
		t.Fatal(err)
	}

	expected := `{"a":1}`
	go ps.Publish("user", "topic", []byte(expected))

	for _, ch := range []DataChannel{ch1, ch2} {
		out := receive(t, ch)
		if string(out.Data) != expected {
			t.Errorf("want: %s, got: %s", expected, string(out.Data))
		}
	}
}

func TestUnsubscribeClosesChannel(t *testing.T) {
	t.Parallel()

	ps := New(1)
	ch, err := ps.Subscribe("user", "topic")
	if err != nil {
		t.Fatal(err)
	}

	ps.Unsubscribe("user", "topic")

	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be closed")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for channel to close")
	}
}

func TestSubscribeExistingWhenAtCapacity(t *testing.T) {
	t.Parallel()

	ps := New(1)
	first, err := ps.Subscribe("user", "topic")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ps.Subscribe("user", "topic")
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("expected existing subscription channel")
	}
}

func TestConcurrentPublishAndUnsubscribe(t *testing.T) {
	ps := New(100)
	done := make(chan struct{})

	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("user-%d", i)
		ch, err := ps.Subscribe(id, "topic")
		if err != nil {
			t.Fatal(err)
		}
		go func(ch DataChannel) {
			for {
				select {
				case _, ok := <-ch:
					if !ok {
						return
					}
				case <-done:
					return
				}
			}
		}(ch)
	}

	publishDone := make(chan struct{})
	go func() {
		defer close(publishDone)
		for i := 0; i < 100; i++ {
			ps.Publish("source", "topic", []byte(`{"a":1}`))
		}
	}()

	for i := 0; i < 20; i++ {
		ps.Unsubscribe(fmt.Sprintf("user-%d", i), "topic")
	}

	select {
	case <-publishDone:
	case <-time.After(time.Second):
		t.Fatal("publish did not finish")
	}
	close(done)
}

func receive(t *testing.T, ch DataChannel) Data {
	t.Helper()

	select {
	case out, ok := <-ch:
		if !ok {
			t.Fatal("channel closed before receiving data")
		}
		return out
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for data")
		return Data{}
	}
}
