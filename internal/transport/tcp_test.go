package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"testing"

	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

func TestTCPSingle(t *testing.T) {
	s := newStage(t)
	s.subscribeToTopic("user1", "topic").
		addMessageToPublish("user", "topic", `{"a":1}`).
		addMessageToPublish("user", "topic", `{"a":2}`).
		publishMessages().
		checkIfAllMessagesWereReceived()
}

func TestTCPMultipleClients(t *testing.T) {
	s := newStage(t)
	s.subscribeToTopic("user1", "topic").
		subscribeToTopic("user2", "topic").
		addMessageToPublish("user", "topic", `{"a":1}`).
		addMessageToPublish("user", "topic", `{"a":2}`).
		publishMessages().
		checkIfAllMessagesWereReceived()
}

func TestTCPMultipleClientsAndPublishers(t *testing.T) {
	s := newStage(t)
	s.subscribeToTopic("user1", "topic").
		subscribeToTopic("user2", "topic").
		addMessageToPublish("user3", "topic", `{"a":1}`).
		addMessageToPublish("user4", "topic", `{"a":2}`).
		publishMessages().
		checkIfAllMessagesWereReceived()
}

func TestTCPMultipleTopics(t *testing.T) {
	s := newStage(t)
	s.subscribeToTopic("user1", "topic1").
		subscribeToTopic("user2", "topic2").
		addMessageToPublish("user3", "topic1", `{"a":1}`).
		addMessageToPublish("user3", "topic2", `{"a":2}`).
		addMessageToPublish("user4", "topic1", `{"a":1}`).
		addMessageToPublish("user4", "topic2", `{"a":2}`).
		publishMessages().
		checkIfAllMessagesWereReceived()
}

type stage struct {
	t         *testing.T
	subs      []chan pubsub.Data
	tcp       *TCP
	ctx       context.Context
	published []pubsub.Data
}

func newStage(t *testing.T) *stage {
	//t.Parallel()
	tcp := NewTCP(func(t *TCP) {
		t.PubAddress = ":0"
		t.SubAddress = ":0"
	})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func(ctx context.Context) {
		err := tcp.Run(ctx)
		t.Error(err)
	}(ctx)
	tcp.Wait()
	s := &stage{
		t:    t,
		subs: []chan pubsub.Data{},
		tcp:  tcp,
		ctx:  ctx,
	}
	return s
}

func (s *stage) subscribeToTopic(user, topic string) *stage {
	var d net.Dialer
	sub, err := d.DialContext(s.ctx, "tcp", s.tcp.SubAddr())
	if err != nil {
		s.t.Error(err)
	}
	err = json.NewEncoder(sub).Encode(&pubsub.Data{Source: user, Topic: topic})
	if err != nil {
		s.t.Error(err)
	}
	msgs := make(chan pubsub.Data)
	go func() {
		dec := json.NewDecoder(sub)
		for {
			var data pubsub.Data
			err := dec.Decode(&data)
			if err != nil {
				s.t.Error(err)
			}
			msgs <- data
		}
	}()
	s.subs = append(s.subs, msgs)
	return s
}

func (s *stage) addMessageToPublish(user, topic, data string) *stage {
	s.published = append(s.published, pubsub.Data{Source: user, Topic: topic, Data: []byte(data)})
	return s
}

func (s *stage) publishMessages() *stage {
	var d net.Dialer
	p, err := d.DialContext(s.ctx, "tcp", s.tcp.PubAddr())
	if err != nil {
		s.t.Error(err)
	}
	enc := json.NewEncoder(p)
	for _, msg := range s.published {
		err = enc.Encode(&msg)
		if err != nil {
			s.t.Error(err)
		}
	}
	return s
}

func (s *stage) checkIfAllMessagesWereReceived() *stage {
	for _, msgs := range s.subs {
		for msg := range msgs {
			for _, exp := range s.published {
				if exp.Topic == msg.Topic && bytes.Equal(exp.Data, msg.Data) {
					break
				}
			}
		}
	}
	return s
}
