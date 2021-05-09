package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/rkorkosz/go-hook/internal/discovery"
	"github.com/rkorkosz/go-hook/internal/transport"
)

func main() {
	addr := flag.String("bind", ":8000", "address to bind on")
	flag.Parse()
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}
	servers := discovery.New(func(d *discovery.Discovery) {
		d.Current = fmt.Sprintf("http://%s%s", hostname, *addr)
	})
	t := transport.NewHTTP(func(ht *transport.HTTP) {
		ht.Server.Addr = *addr
		ht.Servers = servers
	})
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	go func(ctx context.Context) {
		err := servers.Run(ctx)
		if err != nil {
			log.Fatal(err)
		}
	}(ctx)
	err = t.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
