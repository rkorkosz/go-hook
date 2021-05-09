package pubsub

import "testing"

func TestPubSubSingleSubscriber(t *testing.T) {
	t.Parallel()
	ps := New(2)
	ch, err := ps.Subscribe("user", "topic")
	if err != nil {
		t.Error(err)
	}
	expected := `{"a":1}`
	go ps.Publish("user", "topic", []byte(expected))
	for out := range ch {
		if string(out.Data) != expected {
			t.Errorf("want: %s, got: %s", expected, string(out.Data))
		}
		close(ch)
	}
}

func TestPubSubMultiSubscribers(t *testing.T) {
	t.Parallel()
	ps := New(3)
	ch1, err := ps.Subscribe("user1", "topic")
	if err != nil {
		t.Error(err)
	}
	ch2, err := ps.Subscribe("user2", "topic")
	if err != nil {
		t.Error(err)
	}
	expected := `{"a":1}`
	go ps.Publish("user", "topic", []byte(expected))
	for out := range ch1 {
		if string(out.Data) != expected {
			t.Errorf("want: %s, got: %s", expected, string(out.Data))
		}
		close(ch1)
	}
	for out := range ch2 {
		if string(out.Data) != expected {
			t.Errorf("want: %s, got: %s", expected, string(out.Data))
		}
		close(ch2)
	}
}
