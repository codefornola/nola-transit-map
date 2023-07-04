package main

import (
	"context"
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

// Poll fetches results on an interval and publishes the results.
func (p Poller[U]) Poll(ctx context.Context) error {
	tic := time.NewTicker(p.Config.Interval)
	defer tic.Stop()
	for {
		select {
		case <-tic.C:
			results, err := p.APIClient.Load(ctx)
			if err != nil {
				p.Log.Printf("ERROR: apiclient failed to load: %s", err)
				continue
			}
			p.Publisher.Publish(ctx, results)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
