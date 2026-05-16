package transport

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"

	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

// TCP implements transport interface over tcp protocol
type TCP struct {
	Servers     Servers
	PubSub      PubSub
	Config      net.ListenConfig
	Log         *log.Logger
	PubAddress  string
	SubAddress  string
	started     chan struct{}
	pubListener net.Listener
	subListener net.Listener
}

// NewTCP creates new TCP object
func NewTCP(opts ...func(*TCP)) *TCP {
	t := TCP{
		PubSub:     pubsub.New(100),
		PubAddress: ":9000",
		SubAddress: ":9001",
		Log:        log.New(os.Stdout, "[TCP] ", log.LstdFlags),
		started:    make(chan struct{}, 2),
	}
	for _, opt := range opts {
		opt(&t)
	}
	return &t
}

// Wait blocks until all listeners started
func (t *TCP) Wait() {
	<-t.started
	<-t.started
}

// PubAddr returns tcp address used for publishing messages
func (t *TCP) PubAddr() string {
	return t.pubListener.Addr().String()
}

// SubAddr returns tcp address used for subscribing to topics
func (t *TCP) SubAddr() string {
	return t.subListener.Addr().String()
}

// Run creates a main tcp transport loop
func (t *TCP) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

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
		_ = t.subListener.Close()
		_ = t.pubListener.Close()
		return nil
	case err := <-errCh:
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
				if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
					return
				}
				errCh <- err
				return
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
			if err != io.EOF {
				t.Log.Println(err)
			}
			return
		}
		t.PubSub.Publish(data.Source, data.Topic, data.Data)
	}
}

func (t *TCP) handleSub(conn net.Conn) {
	defer conn.Close()

	var data pubsub.Data
	if err := json.NewDecoder(conn).Decode(&data); err != nil {
		if err != io.EOF {
			t.Log.Println(err)
		}
		return
	}

	ch, err := t.PubSub.Subscribe(data.Source, data.Topic)
	if err != nil {
		t.Log.Println(err)
		return
	}
	defer t.PubSub.Unsubscribe(data.Source, data.Topic)

	connDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, conn)
		close(connDone)
	}()

	enc := json.NewEncoder(conn)
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return
			}
			if err := enc.Encode(m); err != nil {
				return
			}
		case <-connDone:
			return
		}
	}
}
