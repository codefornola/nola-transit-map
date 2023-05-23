package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	subscriberBuffer  = flag.Uint("sub-buffer", 200, "size of buffer for subscribers. Should be able to hold one set of vehicle responses")
	subscriberTimeout = flag.Duration("sub-timeout", 10*time.Second, "time allowed to write messages to client")
)

type PubSubConfig struct {
	BufferSize uint
	Timeout    time.Duration
}

type PubSub struct {
	Config PubSubConfig

	mu     sync.Mutex
	subMap map[Subscriber]struct{}
}

func (ps PubSub) Publish(ctx context.Context, msgs []json.RawMessage) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for sub := range ps.subMap {
		select {
		case sub.In <- msgs:
		default:
			go sub.CloseSlow()
		}
	}
}

type Subscriber struct {
	In   chan []json.RawMessage
	conn *websocket.Conn
}

func (s Subscriber) CloseSlow() {
	s.conn.Close(
		websocket.StatusPolicyViolation,
		"connection too slow to keep up with messages",
	)
}

func (ps PubSub) Subscribe(ctx context.Context, conn *websocket.Conn) error {
	ctx = conn.CloseRead(ctx)
	sub := Subscriber{
		In:   make(chan []json.RawMessage, ps.Config.BufferSize),
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
