package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type clientStub struct {
	resp *http.Response
	err  error

	gotReq *http.Request
}

func (s *clientStub) Do(req *http.Request) (*http.Response, error) {
	s.gotReq = req
	return s.resp, s.err
}

type logStub struct {
	w io.Writer
}

func (s logStub) Printf(format string, v ...any) {
	fmt.Fprintf(s.w, format, v...)
}

type publisherStub struct {
	results [][]json.RawMessage
}

func (s *publisherStub) Publish(_ context.Context, results []json.RawMessage) {
	s.results = append(s.results, results)
}

func newAPIRespBody(t testing.TB, results []json.RawMessage) io.ReadCloser {
	var payload struct {
		Data struct {
			Vehicles []json.RawMessage `json:"vehicle"`
		} `json:"bustime-response"`
	}
	payload.Data.Vehicles = results
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		t.Fatalf("failed to encode response body: %s", err)
	}
	return io.NopCloser(bytes.NewReader(buf.Bytes()))
}

func TestAPIPoller_Poll_success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()

	cStub := &clientStub{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Status:     http.StatusText(http.StatusOK),
			Body: newAPIRespBody(t, []json.RawMessage{
				json.RawMessage(`{"a":"b"}`),
				json.RawMessage(`{"c":"d"}`),
			}),
		},
		err: nil,
	}
	var buf bytes.Buffer
	pubStub := new(publisherStub)

	err := (&APIPoller{
		Config: APIPollerConfig{
			Interval: 10 * time.Millisecond,
			Key:      "key",
			Host:     "host",
		},
		Client:    cStub,
		Log:       logStub{w: &buf},
		Publisher: pubStub,
	}).Poll(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unexpected err: %s", err)
	}
	wantUrl := `https://host/bustime/api/v3/getvehicles?format=json&key=key&rtpidatafeed=bustime&tmres=m`
	if got := cStub.gotReq.URL.String(); got != wantUrl {
		t.Errorf("mismatch url. got: %s, want: %s", got, wantUrl)
	}
	wantLog := `INFO: poller fetched 2 vehicles`
	if got := buf.String(); got != wantLog {
		t.Errorf("mismatch log. got: %s, want: %s", got, wantLog)
	}
	wantResults := [][]json.RawMessage{
		{
			json.RawMessage(`{"a":"b"}`),
			json.RawMessage(`{"c":"d"}`),
		},
	}
	if got := pubStub.results; !cmp.Equal(got, wantResults) {
		t.Errorf("mismatch published results. diff: %v", cmp.Diff(got, wantResults))
	}
}

// TODO func TestAPIPoller_Poll_fail(t *testing.T) { }
