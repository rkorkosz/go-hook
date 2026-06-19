package pubsub

import (
	"encoding/json"
	"errors"
	"sync"
)

// Data represents data message
type Data struct {
	Data   json.RawMessage `json:"data"`
	Source string          `json:"source"`
	Topic  string          `json:"topic"`
}

// DataChannel holds one subscription channel.
// Subscribers receive from it; PubSub owns closing and sending.
type DataChannel <-chan Data

type subscription struct {
	ch     chan Data
	done   chan struct{}
	mu     sync.Mutex
	closed bool
}

func newSubscription() *subscription {
	return &subscription{
		ch:   make(chan Data),
		done: make(chan struct{}),
	}
}

func (s *subscription) send(data Data) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}

	select {
	case s.ch <- data:
	case <-s.done:
	}
}

func (s *subscription) close() {
	close(s.done)

	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		close(s.ch)
	}
}

// PubSub implements publish subscribe pattern
type PubSub struct {
	rm   sync.RWMutex
	subs map[string]map[string]*subscription
	// maxTopics caps the number of distinct topics.
	// Both the topic count and per-topic subscriber count share this limit.
	maxTopics int
}

// New creates PubSub object. The cap argument limits the total number of
// distinct topics (and also the number of subscribers per topic).
func New(cap int) *PubSub {
	return &PubSub{
		subs:      make(map[string]map[string]*subscription, cap),
		maxTopics: cap,
	}
}

// Subscribe creates new DataChannel as subscription to send messages
func (ps *PubSub) Subscribe(id, topic string) (DataChannel, error) {
	ps.rm.Lock()
	defer ps.rm.Unlock()

	if _, ok := ps.subs[topic]; !ok {
		if len(ps.subs) >= ps.maxTopics {
			return nil, errors.New("new topics cannot be added")
		}
		ps.subs[topic] = make(map[string]*subscription)
	}

	if sub, ok := ps.subs[topic][id]; ok {
		return sub.ch, nil
	}

	if len(ps.subs[topic]) >= ps.maxTopics {
		return nil, errors.New("new ids cannot be added")
	}

	sub := newSubscription()
	ps.subs[topic][id] = sub
	return sub.ch, nil
}

// Publish sends a message to DataChannel
func (ps *PubSub) Publish(source, topic string, data []byte) {
	ps.rm.RLock()
	subs := make([]*subscription, 0, len(ps.subs[topic]))
	for _, sub := range ps.subs[topic] {
		subs = append(subs, sub)
	}
	ps.rm.RUnlock()

	message := Data{Data: data, Source: source, Topic: topic}
	for _, sub := range subs {
		sub.send(message)
	}
}

// Unsubscribe removes DataChannel subscription from local cache
func (ps *PubSub) Unsubscribe(id, topic string) {
	ps.rm.Lock()
	defer ps.rm.Unlock()

	topicSubs, ok := ps.subs[topic]
	if !ok {
		return
	}

	sub, ok := topicSubs[id]
	if !ok {
		return
	}

	delete(topicSubs, id)
	if len(topicSubs) == 0 {
		delete(ps.subs, topic)
	}

	sub.close()
}
