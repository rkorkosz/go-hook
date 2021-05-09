package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

// HTTP represents transport over HTTP protocol
type HTTP struct {
	Server  *http.Server
	Servers Servers
	PubSub  PubSub
	Log     *log.Logger
}

// NewHTTP creates HTTP object with sensible defaults
func NewHTTP(opts ...func(ht *HTTP)) *HTTP {
	ht := &HTTP{
		Server: &http.Server{Addr: ":8000"},
		PubSub: pubsub.New(100),
		Log:    log.New(os.Stdout, "[HTTP] ", log.LstdFlags),
	}
	ht.Server.Handler = ht
	for _, opt := range opts {
		opt(ht)
	}
	return ht
}

// Run creates a main transport loop
func (ht *HTTP) Run(ctx context.Context) error {
	errCh := make(chan error)
	defer close(errCh)
	go func() {
		ht.Log.Println("Starting server")
		if err := ht.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := ht.Server.Shutdown(shutdownCtx); err != nil {
			return err
		}
	case err := <-errCh:
		return err
	}
	return nil
}

// ServeHTTP implements http.Handler interface
func (ht *HTTP) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		ht.subscribe(w, r)
	case "POST":
		ht.publish(w, r)
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func (ht *HTTP) publishToServer(server, source, topic string, data []byte) {
	uri := fmt.Sprintf("%s/%s/%s", server, topic, source)
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer(data))
	if err != nil {
		ht.Log.Println(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		ht.Log.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", hostname)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		ht.Log.Println(err)
	}
	ht.Log.Printf("Publish status code: %d", resp.StatusCode)
}

func (ht *HTTP) subscribe(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	chunks := strings.Split(r.URL.Path, "/")
	if len(chunks) != 3 {
		http.Error(w, "path should be as follows /topic/name", http.StatusBadRequest)
		return
	}
	topic := chunks[1]
	source := chunks[2]
	ht.Log.Printf("subscribing to %s", topic)

	ch, err := ht.PubSub.Subscribe(source, topic)
	if err != nil {
		ht.Log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	flusher.Flush()
	enc := json.NewEncoder(w)
	for {
		select {
		case m := <-ch:
			_ = enc.Encode(m)
			flusher.Flush()
		case <-r.Context().Done():
			ht.Log.Println("Unsubscribe")
			ht.PubSub.Unsubscribe(source, topic)
			return
		}
	}
}

func (ht *HTTP) publish(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		ht.Log.Println(err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	chunks := strings.Split(r.URL.Path[1:], "/")
	if len(chunks) < 2 {
		http.Error(w, "please provide topic and id in path (/topic/id)", http.StatusBadRequest)
		return

	}
	topic, source := chunks[0], chunks[1]
	ht.Log.Printf("publishing message to: %s", topic)

	origin := r.Header.Get("Origin")
	ht.PubSub.Publish(source, topic, data)
	if origin == "" && ht.Servers != nil {
		for srv := range ht.Servers.Iter() {
			go ht.publishToServer(srv, source, topic, data)
		}
	}
	w.WriteHeader(202)
}
