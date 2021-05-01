package pubsub

import (
	"encoding/json"
	"errors"
	"sync"
)

type Data struct {
	Data   json.RawMessage `json:"data"`
	Source string          `json:"source"`
	Topic  string          `json:"topic"`
}

type DataChannel chan Data

type PubSub struct {
	rm   sync.RWMutex
	subs map[string]map[string]DataChannel
	cap  int
}

func New(cap int) *PubSub {
	return &PubSub{
		rm:   sync.RWMutex{},
		subs: make(map[string]map[string]DataChannel, cap),
		cap:  cap,
	}
}

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

func (ps *PubSub) Publish(source, topic string, data []byte) {
	ps.rm.RLock()
	defer ps.rm.RUnlock()
	for _, ch := range ps.subs[topic] {
		ch <- Data{Data: data, Source: source, Topic: topic}
	}
}

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
