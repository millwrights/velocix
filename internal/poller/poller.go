package poller

import (
	"context"
	"log/slog"
	"time"

	gh "github.com/skalluru/velocix/internal/github"
	"github.com/skalluru/velocix/internal/store"
)

type Poller struct {
	client   *gh.Client
	store    *store.Store
	org      string
	interval time.Duration
	logger   *slog.Logger
}

func New(client *gh.Client, store *store.Store, org string, interval time.Duration, logger *slog.Logger) *Poller {
	return &Poller{
		client:   client,
		store:    store,
		org:      org,
		interval: interval,
		logger:   logger,
	}
}

func (p *Poller) Start(ctx context.Context) {
	go func() {
		// Initial fetch immediately
		p.poll(ctx)

		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				p.logger.Info("poller stopped")
				return
			case <-ticker.C:
				p.poll(ctx)
			}
		}
	}()
}

func (p *Poller) RunOnce(ctx context.Context) error {
	return p.poll(ctx)
}

func (p *Poller) poll(ctx context.Context) error {
	p.logger.Info("polling workflow runs", "org", p.org)
	runs, err := p.client.FetchAllWorkflowRuns(ctx, p.org)
	if err != nil {
		p.logger.Error("poll failed", "error", err)
		return err
	}
	p.store.Update(runs)
	p.logger.Info("poll complete", "runs", len(runs))
	return nil
}
