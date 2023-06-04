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
)

type APIClient[U ~[]V, V any] struct {
	Config APIClientConfig
	Client interface {
		Do(*http.Request) (*http.Response, error)
	}
	Log interface {
		Printf(string, ...any)
	}
	once sync.Once
	req  *http.Request
}

type APIClientConfig struct {
	Key  string // env:CLEVER_DEVICES_KEY
	Host string // env:CLEVER_DEVICES_IP
}

func (c *APIClientConfig) Env() error {
	var ok bool
	if c.Key, ok = os.LookupEnv("CLEVER_DEVICES_KEY"); !ok {
		return errors.New("Need to set environment variable CLEVER_DEVICES_KEY. " +
			"Try `make run CLEVER_DEVICES_KEY=theKey`. " +
			"Get key from Ben on slack")
	}
	if c.Host, ok = os.LookupEnv("CLEVER_DEVICES_IP"); !ok {
		return errors.New("Need to set environment variable CLEVER_DEVICES_IP. " +
			"Try `make run CLEVER_DEVICES_IP=theIP`. " +
			"Get ip from Ben on slack")
	}
	return nil
}

func (a *APIClient[U, V]) init(ctx context.Context) (err error) {
	a.once.Do(func() {
		target := &url.URL{
			Scheme: "https",
			Host:   a.Config.Host,
			Path:   "/bustime/api/v3/getvehicles",
			RawQuery: url.Values(map[string][]string{
				"key":          {a.Config.Key},
				"tmres":        {"s"},
				"rtpidatafeed": {"bustime"},
				"format":       {"json"},
			}).Encode(),
		}
		a.req, err = http.NewRequest(http.MethodGet, target.String(), nil)
		if err != nil {
			err = fmt.Errorf("failed to build request: %w", err)
			return
		}
	})
	return
}

// Load GETs the api data and parses the response body.
//
// example vehicle:
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
func (a *APIClient[U, V]) Load(ctx context.Context) (U, error) {
	if err := a.init(ctx); err != nil {
		var u U
		return u, fmt.Errorf("failed to init client: %w", err)
	}
	results, err := a.load(ctx)
	if err != nil {
		return results, err
	}
	a.Log.Printf("INFO: client fetched %d vehicles", len(results))
	return results, err
}

func (a *APIClient[U, V]) load(ctx context.Context) (U, error) {
	resp, err := a.Client.Do(a.req.Clone(ctx))
	if err != nil {
		var u U
		return u, fmt.Errorf("GET failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var u U
		return u, fmt.Errorf("response returned with Status '%s'", resp.Status)
	}
	var body struct {
		Data struct {
			Vehicles U `json:"vehicle"`
			// Errors   []struct {
			// 	Rt  string `json:"rt"`
			// 	Msg string `json:"msg"`
			// } `json:"error"`
		} `json:"bustime-response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		var u U
		return u, fmt.Errorf("failed to decode response body: %w", err)
	}
	return body.Data.Vehicles, nil
}
