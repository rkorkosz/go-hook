package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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
	rm      sync.RWMutex
	subs    map[string]map[string]DataChannel
	cap     int
	servers map[string]struct{}
}

func NewPubSub(cap int) *PubSub {
	return &PubSub{
		rm:      sync.RWMutex{},
		subs:    make(map[string]map[string]DataChannel, cap),
		cap:     cap,
		servers: make(map[string]struct{}),
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

func (ps *PubSub) Publish(source, topic string, data []byte, publishNext bool) {
	ps.rm.RLock()
	defer ps.rm.RUnlock()
	for _, ch := range ps.subs[topic] {
		log.Println("Publishing locally")
		ch <- Data{Data: data, Source: source}
	}
	log.Println("Publish next")
	if publishNext {
		for srv := range ps.servers {
			log.Printf("Publishing to remote %s", srv)
			ps.PublishToServer(srv, source, topic, data)
		}
	}
}

func (ps *PubSub) PublishToServer(server, source, topic string, data []byte) {
	uri := fmt.Sprintf("%s/%s/%s", server, topic, source)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(data))
	if err != nil {
		log.Println(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		log.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", hostname)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
	}
	log.Printf("Publish status code: %d", resp.StatusCode)
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
	log.Printf("publishing message to: %s", topic)
	origin := r.Header.Get("Origin")
	log.Printf("Origin: %s", origin)
	ps.Publish(source, topic, data, origin == "")
	w.WriteHeader(202)
}
