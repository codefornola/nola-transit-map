package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/sync/errgroup"
	"nhooyr.io/websocket"
)

type writeCloser struct{ io.Writer }

func (writeCloser) Close() error { return nil }

type connStub struct {
	wc        io.WriteCloser
	writerErr error
}

func (s connStub) Close(websocket.StatusCode, string) error      { return nil }
func (s connStub) CloseRead(ctx context.Context) context.Context { return ctx }
func (s connStub) Writer(context.Context, websocket.MessageType) (io.WriteCloser, error) {
	return s.wc, s.writerErr
}

func TestPubSub_publish_success(t *testing.T) {
	t.Run("no subscribers", func(t *testing.T) {
		var gotLog bytes.Buffer
		cfg := PubSubConfig{BufferSize: 1}
		subMap := map[Subscriber[int]]struct{}{}
		pubSub := &PubSub[int]{
			Config: cfg,
			Log:    log.New(&gotLog, "", 0),
			subMap: subMap,
		}
		pubSub.publish(context.Background(), 1)
		wantLog := "INFO: publishing to 0 subscribers\n"
		if want := wantLog; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		if got, want := pubSub.cache, 1; got != want {
			t.Errorf("mismatch cache. diff: %v", cmp.Diff(got, want))
		}
		want := 1
		for sub := range subMap {
			if got, ok := <-sub.In; !ok || got != want {
				t.Errorf("mismatch subscriber message. diff: %v", cmp.Diff(got, want))
			}
		}
	})
	t.Run("single subscriber", func(t *testing.T) {
		var gotLog bytes.Buffer
		cfg := PubSubConfig{BufferSize: 1}
		subMap := map[Subscriber[int]]struct{}{
			NewSubscriber[int](cfg, connStub{}): {},
		}
		pubSub := &PubSub[int]{
			Config: cfg,
			Log:    log.New(&gotLog, "", 0),
			subMap: subMap,
		}
		pubSub.publish(context.Background(), 1)
		wantLog := "INFO: publishing to 1 subscribers\n"
		if want := wantLog; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		if got, want := pubSub.cache, 1; got != want {
			t.Errorf("mismatch cache. diff: %v", cmp.Diff(got, want))
		}
		want := 1
		for sub := range subMap {
			if got, ok := <-sub.In; !ok || got != want {
				t.Errorf("mismatch subscriber message. diff: %v", cmp.Diff(got, want))
			}
		}
	})
	t.Run("multiple subscribers", func(t *testing.T) {
		var gotLog bytes.Buffer
		cfg := PubSubConfig{BufferSize: 1}
		subMap := map[Subscriber[int]]struct{}{
			NewSubscriber[int](cfg, connStub{}): {},
			NewSubscriber[int](cfg, connStub{}): {},
		}
		pubSub := &PubSub[int]{
			Config: cfg,
			Log:    log.New(&gotLog, "", 0),
			subMap: subMap,
		}
		pubSub.publish(context.Background(), 1)
		wantLog := "INFO: publishing to 2 subscribers\n"
		if want := wantLog; gotLog.String() != want {
			t.Errorf("mismatch log. diff: %v", cmp.Diff(gotLog.String(), want))
		}
		if got, want := pubSub.cache, 1; got != want {
			t.Errorf("mismatch cache. diff: %v", cmp.Diff(got, want))
		}
		want := 1
		for sub := range subMap {
			if got, ok := <-sub.In; !ok || got != want {
				t.Errorf("mismatch subscriber message. diff: %v", cmp.Diff(got, want))
			}
		}
	})
}

// TODO test subscriber CloseSlow during publish

func TestPubSub_subscribe_success(t *testing.T) {
	t.Run("no cache, single message", func(t *testing.T) {
		var got bytes.Buffer
		pubSub := &PubSub[int]{
			Config: PubSubConfig{BufferSize: 1, Timeout: time.Second},
			Log:    log.New(io.Discard, "", 0),
			subMap: make(map[Subscriber[int]]struct{}),
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		g.Go(func() error {
			errc, done := pubSub.subscribe(ctx, connStub{
				wc:        writeCloser{Writer: &got},
				writerErr: nil,
			})
			defer done()
			wg.Done()
			return <-errc
		})
		wg.Wait()
		for sub := range pubSub.subMap {
			sub.In <- 1
		}
		if err := g.Wait(); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		want := "1\n"
		if got.String() != want {
			t.Errorf("mismatch. diff: %v", cmp.Diff(got.String(), want))
		}
	})
	t.Run("with cache, single message", func(t *testing.T) {
		var got bytes.Buffer
		pubSub := &PubSub[int]{
			Config: PubSubConfig{BufferSize: 1, Timeout: time.Second},
			Log:    log.New(io.Discard, "", 0),
			subMap: make(map[Subscriber[int]]struct{}),
			cache:  1,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		g.Go(func() error {
			errc, done := pubSub.subscribe(ctx, connStub{
				wc:        writeCloser{Writer: &got},
				writerErr: nil,
			})
			defer done()
			wg.Done()
			return <-errc
		})
		wg.Wait()
		for sub := range pubSub.subMap {
			sub.In <- 1
		}
		if err := g.Wait(); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		want := "1\n1\n"
		if got.String() != want {
			t.Errorf("mismatch. diff: %v", cmp.Diff(got.String(), want))
		}
	})
	t.Run("with cache, multiple messages", func(t *testing.T) {
		var got bytes.Buffer
		pubSub := &PubSub[int]{
			Config: PubSubConfig{BufferSize: 1, Timeout: time.Second},
			Log:    log.New(io.Discard, "", 0),
			subMap: make(map[Subscriber[int]]struct{}),
			cache:  1,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		g, ctx := errgroup.WithContext(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		g.Go(func() error {
			errc, done := pubSub.subscribe(ctx, connStub{
				wc:        writeCloser{Writer: &got},
				writerErr: nil,
			})
			defer done()
			wg.Done()
			return <-errc
		})
		wg.Wait()
		for sub := range pubSub.subMap {
			sub.In <- 1
			sub.In <- 1
		}
		if err := g.Wait(); err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("unexpected err: %v", err)
		}
		want := "1\n1\n1\n"
		if got.String() != want {
			t.Errorf("mismatch. diff: %v", cmp.Diff(got.String(), want))
		}
	})
}
