package main

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
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

type Subscriber[U any] struct {
	In   chan U
	conn *websocket.Conn
}

func (s Subscriber[U]) CloseSlow() {
	s.conn.Close(
		websocket.StatusPolicyViolation,
		"connection too slow to keep up with messages",
	)
}

// Subscribe maintains long-running websocket writes.
func (ps *PubSub[U]) Subscribe(ctx context.Context, conn *websocket.Conn) error {
	ps.init(ctx)
	ctx = conn.CloseRead(ctx)
	sub := Subscriber[U]{
		In:   make(chan U, ps.Config.BufferSize),
		conn: conn,
	}
	ps.mu.Lock()
	ps.subMap[sub] = struct{}{}
	ps.mu.Unlock()
	defer func() {
		ps.mu.Lock()
		delete(ps.subMap, sub)
		ps.mu.Unlock()
		close(sub.In)
	}()

	// send the cached results
	if !reflect.ValueOf(ps.cache).IsZero() {
		ps.cacheMu.RLock()
		sub.In <- ps.cache
		ps.cacheMu.RUnlock()
	}

	for {
		select {
		case msg := <-sub.In:
			{
				ctx, cancel := context.WithTimeout(ctx, ps.Config.Timeout)
				defer cancel()
				if err := wsjson.Write(ctx, conn, msg); err != nil {
					return fmt.Errorf("write messages failed: %w", err)
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
