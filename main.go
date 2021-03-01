package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	addr := flag.String("bind", ":8000", "address to bind")
	cap := flag.Int("cap", 100, "server topic/user capacity")
	flag.Parse()
	ps := NewPubSub(*cap)
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	srv := &http.Server{
		Addr:    *addr,
		Handler: ps,
	}
	srv.RegisterOnShutdown(ps.Close)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal(err)
	}
}
