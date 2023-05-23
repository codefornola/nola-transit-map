package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

type Scraper struct {
	Config ScraperConfig
	Client interface {
		Do(*http.Request) (*http.Response, error)
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

type ScraperConfig struct {
	Interval time.Duration
	Key      string // env:CLEVER_DEVICES_KEY
	Host     string // env:CLEVER_DEVICES_IP
}

func (sc *ScraperConfig) Env() error {
	var ok bool
	if sc.Key, ok = os.LookupEnv("CLEVER_DEVICES_KEY"); !ok {
		return errors.New("Need to set environment variable CLEVER_DEVICES_KEY. " +
			"Try `make run CLEVER_DEVICES_KEY=theKey`. " +
			"Get key from Ben on slack")
	}
	if sc.Host, ok = os.LookupEnv("CLEVER_DEVICES_IP"); !ok {
		return errors.New("Need to set environment variable CLEVER_DEVICES_IP. " +
			"Try `make run CLEVER_DEVICES_IP=theIP`. " +
			"Get key from Ben on slack")
	}
	return nil
}

func (s *Scraper) init(ctx context.Context) (err error) {
	s.once.Do(func() {
		u := &url.URL{
			Scheme: "https",
			Host:   s.Config.Host,
			Path:   "/bustime/api/v3/getvehicles",
			RawQuery: url.Values(map[string][]string{
				"key":          {s.Config.Key},
				"tmres":        {"m"},
				"rtpidatafeed": {"bustime"},
				"format":       {"json"},
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

// Scrape fetches results on an interval and publishes them.
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
				s.Log.Printf("ERROR: scraper failed to fetch: %s", err)
				continue
			}
			s.Log.Printf("INFO: scraper fetched %d vehicles", len(results))
			s.Publisher.Publish(ctx, results)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// fetch GETs the scape data and parses the response body.
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
