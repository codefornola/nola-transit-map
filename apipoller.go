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

type APIPoller struct {
	Config APIPollerConfig
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

type APIPollerConfig struct {
	Interval time.Duration
	Key      string // env:CLEVER_DEVICES_KEY
	Host     string // env:CLEVER_DEVICES_IP
}

func (c *APIPollerConfig) Env() error {
	var ok bool
	if c.Key, ok = os.LookupEnv("CLEVER_DEVICES_KEY"); !ok {
		return errors.New("Need to set environment variable CLEVER_DEVICES_KEY. " +
			"Try `make run CLEVER_DEVICES_KEY=theKey`. " +
			"Get key from Ben on slack")
	}
	if c.Host, ok = os.LookupEnv("CLEVER_DEVICES_IP"); !ok {
		return errors.New("Need to set environment variable CLEVER_DEVICES_IP. " +
			"Try `make run CLEVER_DEVICES_IP=theIP`. " +
			"Get key from Ben on slack")
	}
	return nil
}

func (a *APIPoller) init(ctx context.Context) (err error) {
	a.once.Do(func() {
		u := &url.URL{
			Scheme: "https",
			Host:   a.Config.Host,
			Path:   "/bustime/api/v3/getvehicles",
			RawQuery: url.Values(map[string][]string{
				"key":          {a.Config.Key},
				"tmres":        {"m"},
				"rtpidatafeed": {"bustime"},
				"format":       {"json"},
			}).Encode(),
		}
		a.req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		if err != nil {
			err = fmt.Errorf("failed to build request: %w", err)
			return
		}
	})
	return
}

// Poll fetches results on an interval and publishes the results.
func (a *APIPoller) Poll(ctx context.Context) error {
	if err := a.init(ctx); err != nil {
		return fmt.Errorf("failed to init poller: %w", err)
	}
	tic := time.NewTicker(a.Config.Interval)
	defer tic.Stop()
	for {
		select {
		case <-tic.C:
			results, err := a.fetch(ctx)
			if err != nil {
				a.Log.Printf("ERROR: poller failed to fetch: %s", err)
				continue
			}
			a.Log.Printf("INFO: poller fetched %d vehicles", len(results))
			a.Publisher.Publish(ctx, results)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// fetch GETs the api data and parses the response body.
//
//	json.RawMessage(`{
//	  "vid": "155",
//	  "tmstmp": "20200827 11:51",
//	  "lat": "29.962149326173048",
//	  "lon": "-90.05214051918121",
//	  "hdg": "357",
//	  "pid": 275,
//	  "rt": "5",
//	  "des": "Saratoga at Canal",
//	  "pdist": 10122,
//	  "dly": false,
//	  "spd": 20,
//	  "tatripid": "3130339",
//	  "tablockid": "15",
//	  "zone": "",
//	  "srvtmstmp": "20200827 11:51",
//	  "oid": "445",
//	  "or": true,
//	  "rid": "501",
//	  "blk": 2102,
//	  "tripid": 982856020
//	}`),
func (a *APIPoller) fetch(ctx context.Context) ([]json.RawMessage, error) {
	resp, err := a.Client.Do(a.req.Clone(ctx))
	if err != nil {
		return nil, fmt.Errorf("GET failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response returned with Status '%s'", resp.Status)
	}
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
