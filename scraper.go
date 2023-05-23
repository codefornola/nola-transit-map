package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	scrapeInterval = flag.Duration("scrape-interval", 10*time.Second, "scraper fetch interval")
	scrapeKey      = flag.String("scrape-key", "", "scraper api key")
	scrapeHost     = flag.String("scrape-host", "", "scraper host/ip")
)

type ScraperConfig struct {
	Interval time.Duration
	Key      string // env:CLEVER_DEVICES_KEY
	Host     string // env:CLEVER_DEVICES_IP
}

type Scraper struct {
	Config ScraperConfig
	Client interface {
		Do(*http.Request) (http.Response, error)
	}
	Log interface {
		Printf(format string, v ...any)
	}
	Publisher interface {
		Publish(context.Context, []json.RawMessage)
	}
	once sync.Once
	req  *http.Request
}

func (s *Scraper) init(ctx context.Context) (err error) {
	s.once.Do(func() {
		u := &url.URL{
			Scheme: "https",
			Host:   s.Config.Host,
			Path:   "/bustime/api/v3/getvehicles",
			RawQuery: url.Values(map[string][]string{
				"key":          []string{s.Config.Key},
				"tmres":        []string{"m"},
				"rtpidatafeed": []string{"bustime"},
				"format":       []string{"json"},
			}).Encode(),
		}
		s.req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			err = fmt.Errorf("failed to build request: %w", err)
			return
		}
	})
	return
}

func (s *Scraper) Scrape(ctx context.Context) error {
	if err := s.init(ctx); err != nil {
		return fmt.Errorf("failed to init Scraper: %w", err)
	}
	tic := time.NewTicker(s.Config.Interval)
	defer tic.Stop()
	for {
		select {
		case <-tic.C:
			results, err := s.fetch(ctx)
			if err != nil {
				s.Log.Printf("ERROR: failed to fetch: %s", err)
				continue
			}
			s.Log.Printf("INFO: Found %d vehicles", len(results))
			s.Publisher.Publish(ctx, results)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (s *Scraper) fetch(ctx context.Context) ([]json.RawMessage, error) {
	resp, err := s.Client.Do(s.req.Clone(ctx))
	if err != nil {
		return nil, fmt.Errorf("GET failed: %w", err)
	}
	defer resp.Body.Close()
	var body struct {
		Data struct {
			Vehicles []json.RawMessage `json:"vehicle"`
			// Errors   []struct {
			// 	Rt  string `json:"rt"`
			// 	Msg string `json:"msg"`
			// } `json:"error"`
		} `json:"bustime-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}
	return body.Data.Vehicles, nil
}
