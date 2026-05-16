package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"testing"
	"time"

	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

func TestTCPRunStopsWhenNoOneWaits(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	tcp := NewTCP(func(tcp *TCP) {
		tcp.PubAddress = "127.0.0.1:0"
		tcp.SubAddress = "127.0.0.1:0"
		tcp.Log = log.New(io.Discard, "", 0)
	})

	done := make(chan error, 1)
	go func() {
		done <- tcp.Run(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error after cancel: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run did not stop after cancel")
	}
}

func TestTCPPublishesToSubscriber(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tcp := NewTCP(func(tcp *TCP) {
		tcp.PubAddress = "127.0.0.1:0"
		tcp.SubAddress = "127.0.0.1:0"
		tcp.Log = log.New(io.Discard, "", 0)
	})

	done := make(chan error, 1)
	go func() {
		done <- tcp.Run(ctx)
	}()
	tcp.Wait()
	defer func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("Run returned error after cancel: %v", err)
			}
		case <-time.After(time.Second):
			t.Error("Run did not stop after cancel")
		}
	}()

	subConn, err := net.Dial("tcp", tcp.SubAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = subConn.Close() }()
	if err := json.NewEncoder(subConn).Encode(pubsub.Data{Source: "subscriber", Topic: "topic"}); err != nil {
		t.Fatal(err)
	}

	pubConn, err := net.Dial("tcp", tcp.PubAddr())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = pubConn.Close() }()
	if err := json.NewEncoder(pubConn).Encode(pubsub.Data{Source: "source", Topic: "topic", Data: []byte(`{"hello":"world"}`)}); err != nil {
		t.Fatal(err)
	}

	_ = subConn.SetReadDeadline(time.Now().Add(time.Second))
	var got pubsub.Data
	if err := json.NewDecoder(subConn).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Source != "source" || got.Topic != "topic" || string(got.Data) != `{"hello":"world"}` {
		t.Fatalf("unexpected message: %+v", got)
	}
}

func TestTCPPublisherDisconnectDoesNotLogEOF(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tcp := NewTCP(func(tcp *TCP) {
		tcp.PubAddress = "127.0.0.1:0"
		tcp.SubAddress = "127.0.0.1:0"
		tcp.Log = log.New(&logs, "", 0)
	})

	done := make(chan error, 1)
	go func() { done <- tcp.Run(ctx) }()
	tcp.Wait()
	defer func() {
		cancel()
		<-done
	}()

	conn, err := net.Dial("tcp", tcp.PubAddr())
	if err != nil {
		t.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	if logs.String() != "" {
		t.Fatalf("expected no log output, got %q", logs.String())
	}
}

func TestTCPUnsubscribesWhenSubscriberDisconnects(t *testing.T) {
	t.Parallel()

	ps := &recordingPubSub{subscribed: make(chan struct{}, 1), unsubscribed: make(chan pubsub.Data, 1)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tcp := NewTCP(func(tcp *TCP) {
		tcp.PubSub = ps
		tcp.PubAddress = "127.0.0.1:0"
		tcp.SubAddress = "127.0.0.1:0"
		tcp.Log = log.New(io.Discard, "", 0)
	})

	done := make(chan error, 1)
	go func() { done <- tcp.Run(ctx) }()
	tcp.Wait()
	defer func() {
		cancel()
		<-done
	}()

	conn, err := net.Dial("tcp", tcp.SubAddr())
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(conn).Encode(pubsub.Data{Source: "subscriber", Topic: "topic"}); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ps.subscribed:
	case <-time.After(time.Second):
		t.Fatal("subscriber was not registered")
	}
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	select {
	case got := <-ps.unsubscribed:
		if got.Source != "subscriber" || got.Topic != "topic" {
			t.Fatalf("unexpected unsubscribe: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber was not unregistered")
	}
}

type recordingPubSub struct {
	subscribed   chan struct{}
	unsubscribed chan pubsub.Data
	messages     chan pubsub.Data
}

func (ps *recordingPubSub) Subscribe(id, topic string) (pubsub.DataChannel, error) {
	ps.messages = make(chan pubsub.Data)
	ps.subscribed <- struct{}{}
	return ps.messages, nil
}

func (ps *recordingPubSub) Unsubscribe(id, topic string) {
	ps.unsubscribed <- pubsub.Data{Source: id, Topic: topic}
	if ps.messages != nil {
		close(ps.messages)
	}
}

func (ps *recordingPubSub) Publish(source, topic string, data []byte) {}
