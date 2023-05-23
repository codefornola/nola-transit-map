package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
)

var args = struct {
	ServerConfig
	ScraperConfig
	PubSubConfig
}{}

func init() {
	flag.StringVar(&args.ServerConfig.Addr, "addr", ":8080", "http service address")
	flag.DurationVar(&args.ServerConfig.Timeout, "timeout", 10*time.Second, "server read and write timeouts")
	flag.DurationVar(&args.ScraperConfig.Interval, "scrape-interval", 10*time.Second, "scraper fetch interval")
	flag.UintVar(&args.PubSubConfig.BufferSize, "sub-buffer", 200, "size of buffer for subscribers. min: one array of vehicle responses")
	flag.DurationVar(&args.PubSubConfig.Timeout, "sub-timeout", 10*time.Second, "time allowed to write messages to client")
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()
	log := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.LUTC|log.Lshortfile)
	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, log *log.Logger) error {
	if err := args.ScraperConfig.Env(); err != nil {
		return fmt.Errorf("failed to setup scraper config: %w", err)
	}

	pubSub := &PubSub[[]json.RawMessage]{
		Config: args.PubSubConfig,
	}
	go (&Scraper{
		Config:    args.ScraperConfig,
		Client:    cleanhttp.DefaultPooledClient(),
		Log:       log,
		Publisher: pubSub,
	}).Scrape(ctx)
	return Server{
		Config:     args.ServerConfig,
		Subscriber: pubSub,
		Log:        log,
		Mux:        http.NewServeMux(),
	}.Start(ctx)
}
