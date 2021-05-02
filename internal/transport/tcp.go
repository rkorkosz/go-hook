package transport

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"

	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

type TCP struct {
	Servers     servers
	PubSub      pubSub
	Config      net.ListenConfig
	Log         *log.Logger
	PubAddress  string
	SubAddress  string
	started     chan struct{}
	pubListener net.Listener
	subListener net.Listener
}

func NewTCP(opts ...func(*TCP)) *TCP {
	t := TCP{
		PubSub:     pubsub.New(100),
		PubAddress: ":9000",
		SubAddress: ":9001",
		Log:        log.New(os.Stdout, "[TCP] ", log.LstdFlags),
		started:    make(chan struct{}),
	}
	for _, opt := range opts {
		opt(&t)
	}
	return &t
}

func (t *TCP) Wait() {
	<-t.started
	<-t.started
}

func (t *TCP) PubAddr() string {
	return t.pubListener.Addr().String()
}

func (t *TCP) SubAddr() string {
	return t.subListener.Addr().String()
}

func (t *TCP) Run(ctx context.Context) error {
	errCh := make(chan error)

	var err error
	t.subListener, err = t.run(ctx, errCh, t.handleSub, t.SubAddress)
	if err != nil {
		return err
	}

	t.pubListener, err = t.run(ctx, errCh, t.handlePub, t.PubAddress)
	if err != nil {
		return err
	}

	close(t.started)

	select {
	case <-ctx.Done():
		err = t.subListener.Close()
		err = t.pubListener.Close()
		return err
	case err := <-errCh:
		close(errCh)
		return err
	}
}

func (t *TCP) run(ctx context.Context, errCh chan error, handle func(net.Conn), addr string) (net.Listener, error) {
	ln, err := t.Config.Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}
	t.started <- struct{}{}
	go func(ln net.Listener, errCh chan error) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				errCh <- err
			}
			go handle(conn)
		}
	}(ln, errCh)
	return ln, nil
}

func (t *TCP) handlePub(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	for {
		var data pubsub.Data
		err := dec.Decode(&data)
		if err != nil {
			t.Log.Println(err)
			return
		}
		t.PubSub.Publish(data.Source, data.Topic, data.Data)
	}
}

func (t *TCP) handleSub(conn net.Conn) {
	defer conn.Close()
	dec := json.NewDecoder(conn)
	enc := json.NewEncoder(conn)
	for {
		var data pubsub.Data
		err := dec.Decode(&data)
		if err != nil && err == io.EOF {
			t.PubSub.Unsubscribe(data.Source, data.Topic)
			return
		}
		ch, err := t.PubSub.Subscribe(data.Source, data.Topic)
		if err != nil {
			t.Log.Println(err)
			return
		}
		for m := range ch {
			enc.Encode(m)
		}
	}
}
