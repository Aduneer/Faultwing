package worker

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"
)

type fakeEventJobStore struct {
	calls  int
	cancel context.CancelFunc
}

func (s *fakeEventJobStore) ProcessNextEventJob(context.Context) (bool, error) {
	s.calls++
	switch s.calls {
	case 1:
		return false, errors.New("database temporarily unavailable")
	case 2:
		return true, nil
	default:
		s.cancel()
		return false, nil
	}
}

func TestProcessorContinuesUntilContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store := &fakeEventJobStore{cancel: cancel}
	var logs bytes.Buffer
	processor := NewProcessor(store, log.New(&logs, "", 0))
	processor.pollInterval = time.Millisecond

	processor.Run(ctx)

	if store.calls != 3 {
		t.Fatalf("expected three processing attempts, got %d", store.calls)
	}
	if !strings.Contains(logs.String(), "database temporarily unavailable") {
		t.Fatalf("expected processing error to be logged, got %q", logs.String())
	}
}
