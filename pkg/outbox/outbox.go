package outbox

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	"go.uber.org/multierr"
)

// Outbox is the primary object in the package that implements the transactional outbox pattern.
type Outbox struct {
	config      Config
	wakeSignal  chan struct{}
	stoppedLock sync.RWMutex
}

// New attempts to construct an Outbox from the provided Config, if the Config is valid
func New(cfg Config) (*Outbox, error) {
	if err := cfg.DefaultAndValidate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	o := &Outbox{
		config:      cfg,
		wakeSignal:  make(chan struct{}, 1),
		stoppedLock: sync.RWMutex{},
	}

	return o, nil
}

// WakeProcessor is used to notify the outbox processor that new data has been written
// to the outbox and it should wake up and process them, rather than wait for the
// Config.ProcessInterval. For batch write operations, try to only call this once so the
// processor is likely to wake up fewer times and process them as a batch. This function
// does not block.
func (o *Outbox) WakeProcessor() {
	o.stoppedLock.RLock()
	defer o.stoppedLock.RUnlock()

	if o.wakeSignal == nil {
		return
	}

	select {
	case o.wakeSignal <- struct{}{}:
	default:
	}
}

// Publish publishes the provided messages to the outbox, and will be forwarded to the configured Publisher during
// one of the subsequent PumpOutbox calls
func (o *Outbox) Publish(ctx context.Context, txn interface{}, messages ...Message) error {
	return o.config.Storage.Publish(ctx, txn, messages...)
}

// StartProcessing blocks, processing the outbox until its context is cancelled.
// It wakes up to process regularly based on the Config.ProcessInterval and can be woken
// manually using WakeProcessor.
func (o *Outbox) StartProcessing(ctx context.Context) error {
	logger := o.config.Logger.WithName("processor")
	logger.Info("outbox processor starting")
	defer logger.Info("outbox processor exiting")

	for {
		select {
		case <-ctx.Done():
			logger.Info("context cancelled", "reason", ctx.Err())
			return nil
		case _, more := <-o.wakeSignal:
			logger.V(1).Info("wake signal received")
			if !more {
				return nil
			}
		case <-o.config.Clock.After(o.config.ProcessInterval):
			logger.V(1).Info("woken by processing interval")
		}

		op := func() error {
			if err := o.PumpOutbox(ctx); err != nil {
				return fmt.Errorf("error pumping outbox: %w", err)
			}
			return nil
		}
		notify := func(err error, duration time.Duration) {
			logger.Error(err, "transient error, will retry", "backoff", duration)
		}
		bo := backoff.WithContext(backoff.NewExponentialBackOff(), ctx)
		if err := backoff.RetryNotify(op, bo, notify); err != nil {
			logger.Error(err, "error, giving up for now")
		}
	}
}

// PumpOutbox causes the Outbox to process entries immediately. This is typically not called directly,
// instead called from StartProcessing. However, this is exposed partially for ease of testing, but
// also to facilitate customising the processing logic if the provided StartProcessing function isn't
// suitable for your application.
func (o *Outbox) PumpOutbox(ctx context.Context) (err error) {
	o.config.Logger.V(1).Info("pumping outbox")

	deadline := o.config.Clock.Now().Add(o.config.ClaimDuration)
	if err := o.config.Storage.ClaimEntries(ctx, o.config.ProcessorID, deadline); err != nil {
		return fmt.Errorf("error claiming entries: %w", err)
	}

	for {
		more, err := o.processBatch(ctx)
		if err != nil {
			return fmt.Errorf("error processing batch of outbox entries: %w", err)
		}

		if !more {
			break
		}
	}

	return nil
}

func (o *Outbox) processBatch(ctx context.Context) (more bool, err error) {
	entries, err := o.config.Storage.GetClaimedEntries(ctx, o.config.ProcessorID, o.config.BatchSize)
	if err != nil {
		return false, fmt.Errorf("error getting claimed entries: %w", err)
	}

	more = len(entries) >= o.config.BatchSize

	entryIDs := make([]string, 0, len(entries))
	namespaced := make(map[string][]Message)
	for _, entry := range entries {
		entryIDs = append(entryIDs, entry.ID)

		msg := Message{
			Key:     entry.Key,
			Payload: entry.Payload,
		}

		namespaced[entry.Namespace] = append(namespaced[entry.Namespace], msg)
	}

	defer func() {
		deletableIDs := entryIDs

		if err != nil {
			deletableIDs = make([]string, 0, len(entries))

			var publishErr *PublishError
			if errors.As(err, &publishErr) {
				for idx, err := range publishErr.Errors {
					if err != nil {
						continue
					}

					deletableIDs = append(deletableIDs, entryIDs[idx])
				}
			}
		}

		if deleteErr := o.config.Storage.DeleteEntries(ctx, deletableIDs...); deleteErr != nil {
			err = multierr.Combine(err, deleteErr)
		}
	}()

	for namespace, messages := range namespaced {
		publishCtx := WithNamespace(ctx, namespace)

		if err := o.config.Publisher.Publish(publishCtx, messages...); err != nil {
			return more, fmt.Errorf("error publishing: %w", err)
		}
	}

	return more, nil
}
