package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Data struct {
	Data   json.RawMessage `json:"data"`
	Source string          `json:"source"`
	close  bool
}

type DataChannel chan Data

type PubSub struct {
	rm   sync.RWMutex
	subs map[string]map[string]DataChannel
	cap  int
}

func NewPubSub(cap int) *PubSub {
	return &PubSub{
		rm:   sync.RWMutex{},
		subs: make(map[string]map[string]DataChannel, cap),
		cap:  cap,
	}
}

func (ps *PubSub) Close() {
	for _, topicChannels := range ps.subs {
		for _, idChannel := range topicChannels {
			idChannel <- Data{close: true}
		}
	}
}

func (ps *PubSub) Subscribe(id, topic string, ch DataChannel) error {
	ps.rm.Lock()
	defer ps.rm.Unlock()
	if len(ps.subs) >= ps.cap {
		return errors.New("new topics cannot be added")
	}
	if _, ok := ps.subs[topic]; !ok {
		ps.subs[topic] = make(map[string]DataChannel)
	}
	if len(ps.subs[topic]) >= ps.cap {
		return errors.New("new ids cannot be added")
	}
	if _, ok := ps.subs[topic][id]; !ok {
		ps.subs[topic][id] = ch
	}
	return nil
}

func (ps *PubSub) Publish(source, topic string, data []byte) {
	if _, ok := ps.subs[topic]; !ok {
		return
	}
	ps.rm.RLock()
	defer ps.rm.RUnlock()
	for _, ch := range ps.subs[topic] {
		ch <- Data{Data: data, Source: source}
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

func (ps *PubSub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		subscribe(w, r, ps)
	case "POST":
		publish(w, r, ps)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func subscribe(w http.ResponseWriter, r *http.Request, ps *PubSub) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(DataChannel)
	topic := r.URL.Path[1:]
	id := uuid.New().String()
	err := ps.Subscribe(id, topic, ch)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	flusher.Flush()
	enc := json.NewEncoder(w)
	for {
		select {
		case m := <-ch:
			if m.close {
				close(ch)
				return
			}
			enc.Encode(m)
			flusher.Flush()
		case <-r.Context().Done():
			ps.Unsubscribe(id, topic)
			return
		}
	}
}

func publish(w http.ResponseWriter, r *http.Request, ps *PubSub) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	chunks := strings.Split(r.URL.Path[1:], "/")
	if len(chunks) < 2 {
		http.Error(w, "please provide topic and id in path (/topic/id)", http.StatusBadRequest)
		return

	}
	topic, source := chunks[0], chunks[1]
	ps.Publish(source, topic, data)
	w.WriteHeader(202)
}
