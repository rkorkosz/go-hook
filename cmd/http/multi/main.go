package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/rkorkosz/go-hook/internal/discovery"
	"github.com/rkorkosz/go-hook/internal/transport"
)

func main() {
	addr := flag.String("bind", ":8000", "address to bind on")
	flag.Parse()
	servers := discovery.New()
	t := transport.NewHTTP(func(ht *transport.HTTP) {
		ht.Server.Addr = *addr
		ht.Servers = servers
	})
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go servers.Run(ctx)
	err := t.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
