package main

import (
	"context"
	"log"
	"os"
	"os/signal"
)

func NewScraper() *Scraper {
	api_key, ok := os.LookupEnv("CLEVER_DEVICES_KEY")
	if !ok {
		panic("Need to set environment variable CLEVER_DEVICES_KEY. Try `make run CLEVER_DEVICES_KEY=thekey`. Get key from Ben on slack")
	}
	ip, ok := os.LookupEnv("CLEVER_DEVICES_IP")
	if !ok {
		panic("Need to set environment variable CLEVER_DEVICES_KEY. Try `make run CLEVER_DEVICES_KEY=thekey`. Get key from Ben on slack")
	}
}

func run(ctx context.Context) error {
	// TODO read vars
	// TODO create structs
	// TODO start server
	return server.Start(ctx)
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}
