package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// PubSub controls a set of Subscribers and publishes messages to them.
// inspired by https://github.com/nhooyr/websocket/tree/v1.8.7/examples/chat
type PubSub[U any] struct {
	Config PubSubConfig
	Log    interface {
		Printf(fomat string, v ...any)
	}

	once   sync.Once
	mu     sync.Mutex
	subMap map[Subscriber[U]]struct{}

	cacheMu sync.RWMutex
	cache   U
}

type PubSubConfig struct {
	BufferSize uint
	Timeout    time.Duration
}

func (ps *PubSub[U]) init(_ context.Context) {
	ps.once.Do(func() {
		ps.mu.Lock()
		ps.subMap = make(map[Subscriber[U]]struct{})
		ps.mu.Unlock()
	})
}

// Publish writes a message to each subscriber.
func (ps *PubSub[U]) Publish(ctx context.Context, msg U) {
	ps.init(ctx)
	ps.publish(ctx, msg)
}

func (ps *PubSub[U]) publish(_ context.Context, msg U) {
	ps.cacheMu.Lock()
	ps.cache = msg
	ps.cacheMu.Unlock()

	ps.mu.Lock()
	ps.Log.Printf("INFO: publishing to %d subscribers", len(ps.subMap))
	defer ps.mu.Unlock()
	for sub := range ps.subMap {
		select {
		case sub.In <- msg:
		default:
			go sub.CloseSlow()
		}
	}
}

type WSConn interface {
	Close(websocket.StatusCode, string) error
	CloseRead(context.Context) context.Context
	Writer(context.Context, websocket.MessageType) (io.WriteCloser, error)
}

// Subscribe creates a Subscriber for long-running websocket writes.
// errc reports when an error occurs during Subscriber.Listen.
// Use done to cleanup resources.
func (ps *PubSub[U]) Subscribe(ctx context.Context, conn WSConn) (errc <-chan error, done func()) {
	ps.init(ctx)
	return ps.subscribe(ctx, conn)
}

func (ps *PubSub[U]) subscribe(ctx context.Context, conn WSConn) (<-chan error, func()) {
	sub := NewSubscriber[U](ps.Config, conn)
	ps.mu.Lock()
	ps.subMap[sub] = struct{}{}
	ps.mu.Unlock()

	// send the cached messages
	ps.cacheMu.RLock()
	if !reflect.ValueOf(ps.cache).IsZero() {
		sub.In <- ps.cache
	}
	ps.cacheMu.RUnlock()

	return sub.Listen(ctx), func() {
		ps.mu.Lock()
		delete(ps.subMap, sub)
		ps.mu.Unlock()
		close(sub.In)
	}
}

type Subscriber[U any] struct {
	In      chan U
	conn    WSConn
	timeout time.Duration
}

func NewSubscriber[U any](cfg PubSubConfig, conn WSConn) Subscriber[U] {
	return Subscriber[U]{
		In:      make(chan U, cfg.BufferSize),
		conn:    conn,
		timeout: cfg.Timeout,
	}
}

// write encodes json messages to the conn.
// inspired by https://github.com/nhooyr/websocket/blob/v1.8.7/wsjson/wsjson.go
func (s Subscriber[U]) write(ctx context.Context, msg U) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	wc, err := s.conn.Writer(ctx, websocket.MessageText)
	if err != nil {
		return fmt.Errorf("open conn writer failed: %w", err)
	}
	if err := json.NewEncoder(wc).Encode(msg); err != nil {
		return fmt.Errorf("write messages failed: %w", err)
	}
	return wc.Close()
}

// Listen ingests messages until failure or cancel.
func (s Subscriber[U]) Listen(ctx context.Context) <-chan error {
	errc := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithCancel(s.conn.CloseRead(ctx))
		defer cancel()
		for {
			select {
			case msg, ok := <-s.In:
				if !ok {
					break
				}
				if err := s.write(ctx, msg); err != nil {
					errc <- fmt.Errorf("subscriber write failed: %w", err)
					return
				}
			case <-ctx.Done():
				errc <- ctx.Err()
				return
			}
		}
	}()
	return errc
}

func (s Subscriber[U]) CloseSlow() {
	s.conn.Close(
		websocket.StatusPolicyViolation,
		"connection too slow to keep up with messages",
	)
}
