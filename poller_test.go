package main

import (
	"bytes"
	"context"
	"errors"
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

func TestPoller_Poll_success(t *testing.T) {
	t.Run("no poll", func(t *testing.T) {
		// intervals determine the number of polls
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
		defer cancel()
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
		}.Poll(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		if want := ""; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		var want [][]int
		if !cmp.Equal(pStub.results, want) {
			t.Errorf("mismatch results. diff: %v", cmp.Diff(pStub.results, want))
		}
	})
	t.Run("single poll", func(t *testing.T) {
		// intervals determine the number of polls
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
		defer cancel()
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
		}.Poll(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		if want := ""; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		if want := [][]int{{1, 2, 3}}; !cmp.Equal(pStub.results, want) {
			t.Errorf("mismatch results. diff: %v", cmp.Diff(pStub.results, want))
		}
	})
	t.Run("two times poll", func(t *testing.T) {
		// intervals determine the number of polls
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()
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
		}.Poll(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		if want := ""; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		if want := [][]int{{1, 2, 3}, {1, 2, 3}}; !cmp.Equal(pStub.results, want) {
			t.Errorf("mismatch results. diff: %v", cmp.Diff(pStub.results, want))
		}
	})
}

func TestPoller_Poll_fail(t *testing.T) {
	errFailed := errors.New("failed")
	t.Run("apiclient err", func(t *testing.T) {
		// intervals determine the number of polls
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
		defer cancel()
		var gotLog bytes.Buffer
		pStub := new(publisherStub[[]int])
		err := Poller[[]int]{
			Config: PollerConfig{
				Interval: 10 * time.Millisecond,
			},
			APIClient: apiClientStub[[]int]{
				result: []int{1, 2, 3},
				err:    errFailed,
			},
			Log:       log.New(&gotLog, "", 0),
			Publisher: pStub,
		}.Poll(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		wantLog := "ERROR: apiclient failed to load: failed\n"
		if want := wantLog; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		var want [][]int
		if !cmp.Equal(pStub.results, want) {
			t.Errorf("mismatch results. diff: %v", cmp.Diff(pStub, want))
		}
	})
	t.Run("two apiclient err", func(t *testing.T) {
		// intervals determine the number of polls
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()
		var gotLog bytes.Buffer
		pStub := new(publisherStub[[]int])
		err := Poller[[]int]{
			Config: PollerConfig{
				Interval: 10 * time.Millisecond,
			},
			APIClient: apiClientStub[[]int]{
				result: []int{1, 2, 3},
				err:    errFailed,
			},
			Log:       log.New(&gotLog, "", 0),
			Publisher: pStub,
		}.Poll(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		wantLog := "ERROR: apiclient failed to load: failed\nERROR: apiclient failed to load: failed\n"
		if want := wantLog; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		var want [][]int
		if !cmp.Equal(pStub.results, want) {
			t.Errorf("mismatch results. diff: %v", cmp.Diff(pStub, want))
		}
	})
}
