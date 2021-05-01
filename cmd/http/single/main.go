package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/rkorkosz/go-hook/internal/transport"
	"github.com/rkorkosz/go-hook/pkg/pubsub"
)

func main() {
	ps := pubsub.New(100)
	ht := transport.NewHTTP(func(ht *transport.HTTP) {
		ht.PubSub = ps
	})
	t := transport.NewTCP(func(t *transport.TCP) {
		t.PubSub = ps
	})
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go ht.Run(ctx)
	err := t.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
