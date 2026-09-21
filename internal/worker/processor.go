package worker

import (
	"context"
	"log"
	"time"
)

const defaultPollInterval = 500 * time.Millisecond

type EventJobStore interface {
	ProcessNextEventJob(context.Context) (bool, error)
}

type Processor struct {
	store        EventJobStore
	logger       *log.Logger
	pollInterval time.Duration
}

func NewProcessor(store EventJobStore, logger *log.Logger) *Processor {
	return &Processor{
		store:        store,
		logger:       logger,
		pollInterval: defaultPollInterval,
	}
}

func (p *Processor) Run(ctx context.Context) {
	for {
		processed, err := p.store.ProcessNextEventJob(ctx)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			p.logger.Printf("process event job: %v", err)
			if !wait(ctx, p.pollInterval) {
				return
			}
			continue
		}
		if processed {
			continue
		}
		if !wait(ctx, p.pollInterval) {
			return
		}
	}
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
