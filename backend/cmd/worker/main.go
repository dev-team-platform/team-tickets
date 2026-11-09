package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/trannhanlv2004/team-tickets/internal/configs"
)

func main() {
	if _, err := configs.LoadAppConfig(); err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Print("worker started; no jobs configured")
	<-ctx.Done()
	log.Print("worker stopped")
}
