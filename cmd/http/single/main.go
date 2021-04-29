package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/rkorkosz/go-hook/internal/transport"
)

func main() {
	t := transport.NewHTTP()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	err := t.Run(ctx)
	if err != nil {
		log.Fatal(err)
	}
}
