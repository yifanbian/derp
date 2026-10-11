package main

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	tailscale "tailscale.com/client/tailscale/v2"
	"tailscale.com/types/key"
	"tailscale.com/util/set"
)

type tailnetSource struct {
	name   string
	client *tailscale.Client
}

// NodeCache periodically refreshes the node keys of all configured tailnets.
type NodeCache struct {
	sources  []tailnetSource
	interval time.Duration
	logger   *slog.Logger

	// Guards per-tailnet state; readers use the immutable merged snapshot instead.
	mu     sync.Mutex
	nodes  map[string]set.Set[key.NodePublic]
	merged atomic.Pointer[set.Set[key.NodePublic]]
}

func NewNodeCache(cfg *Config, logger *slog.Logger) (*NodeCache, error) {
	c := &NodeCache{
		interval: cfg.RefreshInterval,
		logger:   logger,
		nodes:    map[string]set.Set[key.NodePublic]{},
	}
	empty := set.Set[key.NodePublic]{}
	c.merged.Store(&empty)
	for _, t := range cfg.Tailnets {
		client, err := newTailscaleClient(t)
		if err != nil {
			return nil, err
		}
		c.sources = append(c.sources, tailnetSource{name: t.Name, client: client})
	}
	return c, nil
}

// Contains reports whether any tailnet currently has the node key.
func (c *NodeCache) Contains(k key.NodePublic) bool {
	return c.merged.Load().Contains(k)
}

// update stores a tailnet's keys and publishes a new merged snapshot.
func (c *NodeCache) update(name string, keys set.Set[key.NodePublic]) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.nodes[name] = keys
	merged := set.Set[key.NodePublic]{}
	for _, s := range c.nodes {
		merged.AddSet(s)
	}
	c.merged.Store(&merged)
}

// Refresh updates every tailnet concurrently; a failed tailnet keeps its previous entries.
func (c *NodeCache) Refresh(ctx context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, s := range c.sources {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()

			keys, err := fetchNodeKeys(rctx, s.client)
			if err != nil {
				c.logger.Error("refresh tailnet failed", "tailnet", s.name, "error", err)
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
				return
			}
			c.update(s.name, keys)
			c.logger.Info("refreshed tailnet", "tailnet", s.name, "nodes", len(keys))
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// Run refreshes on every interval until ctx is cancelled.
func (c *NodeCache) Run(ctx context.Context) {
	t := time.NewTicker(c.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = c.Refresh(ctx)
		}
	}
}
