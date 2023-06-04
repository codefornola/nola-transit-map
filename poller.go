package main

import (
	"context"
	"fmt"
	"time"
)

type Poller[U any] struct {
	Config    PollerConfig
	APIClient interface {
		Load(context.Context) (U, error)
	}
	Log interface {
		Printf(string, ...any)
	}
	Publisher interface {
		Publish(context.Context, U)
	}
}

type PollerConfig struct {
	Interval time.Duration
}

// Poll polls for results on an interval.
func (p Poller[U]) Poll(ctx context.Context) error {
	tic := time.NewTicker(p.Config.Interval)
	defer tic.Stop()
	if err := p.poll(ctx); err != nil {
		p.Log.Printf("ERROR: startup poll failed: %s", err)
	}
	for {
		select {
		case <-tic.C:
			if err := p.poll(ctx); err != nil {
				p.Log.Printf("ERROR: poll failed: %s", err)
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// poll fetches results on an interval and publishes the results.
func (p Poller[U]) poll(ctx context.Context) error {
	results, err := p.APIClient.Load(ctx)
	if err != nil {
		return fmt.Errorf("apiclient failed to load: %w", err)
	}
	p.Publisher.Publish(ctx, results)
	return nil
}
