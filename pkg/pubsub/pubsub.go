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

// DataChannel holds one subscription channel
type DataChannel chan Data

// PubSub implements publish subscribe pattern
type PubSub struct {
	rm   sync.RWMutex
	subs map[string]map[string]DataChannel
	cap  int
}

// New creates PubSub object
func New(cap int) *PubSub {
	return &PubSub{
		rm:   sync.RWMutex{},
		subs: make(map[string]map[string]DataChannel, cap),
		cap:  cap,
	}
}

// Subscribe creates new DataChannel as subscription to send messages
func (ps *PubSub) Subscribe(id, topic string) (DataChannel, error) {
	ps.rm.Lock()
	defer ps.rm.Unlock()
	if len(ps.subs) >= ps.cap {
		return nil, errors.New("new topics cannot be added")
	}
	if _, ok := ps.subs[topic]; !ok {
		ps.subs[topic] = make(map[string]DataChannel)
	}
	if len(ps.subs[topic]) >= ps.cap {
		return nil, errors.New("new ids cannot be added")
	}
	if _, ok := ps.subs[topic][id]; !ok {
		ps.subs[topic][id] = make(DataChannel)
	}
	return ps.subs[topic][id], nil
}

// Publish sends a message to DataChannel
func (ps *PubSub) Publish(source, topic string, data []byte) {
	ps.rm.RLock()
	defer ps.rm.RUnlock()
	for _, ch := range ps.subs[topic] {
		ch <- Data{Data: data, Source: source, Topic: topic}
	}
}

// Unsubscribe removes DataChannel subscription from local cache
func (ps *PubSub) Unsubscribe(id, topic string) {
	if _, ok := ps.subs[topic]; !ok {
		return
	}
	if _, ok := ps.subs[topic][id]; !ok {
		return
	}
	close(ps.subs[topic][id])
	delete(ps.subs[topic], id)
	if len(ps.subs[topic]) == 0 {
		delete(ps.subs, topic)
	}
}
