package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type publisherStub[U any] struct {
	results []U
}

func (s *publisherStub[U]) Publish(_ context.Context, results U) {
	s.results = append(s.results, results)
}

type apiClientStub[U any] struct {
	result U
	err    error
}

func (s apiClientStub[U]) Load(context.Context) (U, error) {
	return s.result, s.err
}

func TestPoller_poll_success(t *testing.T) {
	var gotLog bytes.Buffer
	pStub := new(publisherStub[[]int])
	err := Poller[[]int]{
		Config: PollerConfig{
			Interval: 10 * time.Millisecond,
		},
		APIClient: apiClientStub[[]int]{
			result: []int{1, 2, 3},
			err:    nil,
		},
		Log:       log.New(&gotLog, "", 0),
		Publisher: pStub,
	}.poll(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if want := ""; gotLog.String() != want {
		t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
	}
	if want := [][]int{{1, 2, 3}}; !cmp.Equal(pStub.results, want) {
		t.Errorf("mismatch results. diff: %v", cmp.Diff(pStub.results, want))
	}
}

func TestPoller_poll_fail(t *testing.T) {
	errFailed := errors.New("failed")
	t.Run("apiclient err", func(t *testing.T) {
		err := Poller[[]int]{
			Config: PollerConfig{
				Interval: 10 * time.Millisecond,
			},
			APIClient: apiClientStub[[]int]{
				result: []int{1, 2, 3},
				err:    errFailed,
			},
			Log:       log.New(io.Discard, "", 0),
			Publisher: new(publisherStub[[]int]),
		}.poll(context.Background())
		if !errors.Is(err, errFailed) {
			t.Fatalf("mismatch err: %v", err)
		}
	})
}
