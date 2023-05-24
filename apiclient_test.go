package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/google/go-cmp/cmp"
)

type clientStub struct {
	newResp func() *http.Response
	err     error

	gotReq *http.Request
}

func (s *clientStub) Do(req *http.Request) (*http.Response, error) {
	s.gotReq = req
	return s.newResp(), s.err
}

func TestAPIClient_load_success(t *testing.T) {
	got, err := (&APIClient[[]json.RawMessage, json.RawMessage]{
		Client: &clientStub{
			newResp: func() *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Status:     http.StatusText(http.StatusOK),
					Body: io.NopCloser(strings.NewReader(
						`{"bustime-response":{"vehicle":[{"a":1},{"a":2}]}}`)),
				}
			},
			err: nil,
		},
		req: new(http.Request),
	}).load(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %s", err)
	}
	want := []json.RawMessage{
		json.RawMessage(`{"a":1}`),
		json.RawMessage(`{"a":2}`),
	}
	if !cmp.Equal(got, want) {
		t.Errorf("mismatch results. diff: %v", cmp.Diff(got, want))
	}
}

func TestAPIClient_load_fail(t *testing.T) {
	t.Run("bad status code", func(t *testing.T) {
		_, err := (&APIClient[[]json.RawMessage, json.RawMessage]{
			Client: &clientStub{
				newResp: func() *http.Response {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Status:     http.StatusText(http.StatusInternalServerError),
						Body: io.NopCloser(strings.NewReader(
							`{"bustime-response":{"vehicle":[{"a":1},{"a":2}]}}`)),
					}
				},
				err: nil,
			},
			req: new(http.Request),
		}).load(context.Background())
		if err == nil {
			t.Fatalf("missing err: %s", err)
		}
		want := http.StatusText(http.StatusInternalServerError)
		if !strings.Contains(err.Error(), want) {
			t.Errorf("mismatch err. diff: %v", cmp.Diff(err, want))
		}
	})
	t.Run("err from client", func(t *testing.T) {
		errFailed := errors.New("failed")
		_, err := (&APIClient[[]json.RawMessage, json.RawMessage]{
			Client: &clientStub{
				newResp: func() *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     http.StatusText(http.StatusOK),
						Body: io.NopCloser(strings.NewReader(
							`{"bustime-response":{"vehicle":[{"a":1},{"a":2}]}}`)),
					}
				},
				err: errFailed,
			},
			req: new(http.Request),
		}).load(context.Background())
		if err == nil {
			t.Fatalf("missing err: %s", err)
		}
		if !errors.Is(err, errFailed) {
			t.Errorf("mismatch err. got: %v, want: %v", err, errFailed)
		}
	})
	t.Run("bad json", func(t *testing.T) {
		errFailed := errors.New("failed")
		_, err := (&APIClient[[]json.RawMessage, json.RawMessage]{
			Client: &clientStub{
				newResp: func() *http.Response {
					return &http.Response{
						StatusCode: http.StatusOK,
						Status:     http.StatusText(http.StatusOK),
						Body:       io.NopCloser(iotest.ErrReader(errFailed)),
					}
				},
				err: nil,
			},
			req: new(http.Request),
		}).load(context.Background())
		if err == nil {
			t.Fatalf("missing err: %s", err)
		}
		if !errors.Is(err, errFailed) {
			t.Errorf("mismatch err. got: %v, want: %v", err, errFailed)
		}
	})
}
