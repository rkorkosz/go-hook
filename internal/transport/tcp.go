package transport

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"

	"github.com/google/uuid"
	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

type TCP struct {
	Servers servers
	PubSub  pubSub
	Log     *log.Logger
	Config  net.ListenConfig
	PubAddr string
	SubAddr string
}

func NewTCP(opts ...func(*TCP)) *TCP {
	t := TCP{
		PubSub:  pubsub.New(100),
		Log:     log.New(os.Stdout, "[TCP] ", log.LstdFlags),
		PubAddr: ":9000",
		SubAddr: ":9001",
	}
	for _, opt := range opts {
		opt(&t)
	}
	return &t
}

func (t *TCP) Run(ctx context.Context) error {
	t.Log.Println("Starting tcp server")
	pln, err := t.Config.Listen(ctx, "tcp", t.PubAddr)
	if err != nil {
		return err
	}
	sln, err := t.Config.Listen(ctx, "tcp", t.SubAddr)
	if err != nil {
		return err
	}
	errCh := make(chan error)
	go func(ln net.Listener, errCh chan error) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
			}
			go t.handlePub(conn)
		}
	}(pln, errCh)
	go func(ln net.Listener, errCh chan error) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
			}
			go t.handleSub(conn)
		}
	}(sln, errCh)
	select {
	case <-ctx.Done():
		pln.Close()
		sln.Close()
	case err := <-errCh:
		close(errCh)
		return err
	}
	return nil
}

func (t *TCP) handlePub(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	for {
		var data pubsub.Data
		err := dec.Decode(&data)
		if err != nil {
			return
		}
		t.PubSub.Publish(data.Source, data.Topic, data.Data)
	}
}

func (t *TCP) handleSub(conn net.Conn) {
	defer conn.Close()
	id := uuid.New().String()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var data pubsub.Data
		err := dec.Decode(&data)
		if err != nil && err == io.EOF {
			t.PubSub.Unsubscribe(id, data.Topic)
			return
		}
		ch, err := t.PubSub.Subscribe(id, data.Topic)
		if err != nil {
			t.Log.Println(err)
			return
		}
		for m := range ch {
			enc.Encode(m)
		}
	}
}
