package fake

import (
	"context"
	"sync"
	"time"

	"github.com/omaskery/outboxen/pkg/outbox"

	"github.com/google/uuid"
)

// Clock abstracts the time package
type Clock interface {
	Now() time.Time
}

// EntryStorage is a simple fake implementation of two outbox interfaces:
//   - outbox.ProcessorStorage: for use directly by the outbox.Outbox to process Outbox Entry objects
//   - outbox.Publisher: for applications to treat as the outbox.Outbox that records their
//     messages during a transaction
type EntryStorage struct {
	// Clock abstracts the time package
	Clock   Clock
	lock    sync.RWMutex
	entries []*outbox.Entry
}

// Publish records the provided messages to the outbox.ProcessorStorage
func (e *EntryStorage) Publish(_ context.Context, messages ...outbox.Message) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	for _, message := range messages {
		e.entries = append(e.entries, &outbox.Entry{
			ID:        uuid.NewString(),
			CreatedAt: e.Clock.Now(),
			Key:       message.Key,
			Payload:   message.Payload,
		})
	}

	return nil
}

// ClaimEntries implements outbox.ProcessorStorage interface
func (e *EntryStorage) ClaimEntries(_ context.Context, processorID string, claimDeadline time.Time) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	now := e.Clock.Now()
	for _, entry := range e.entries {
		if entry.ProcessorID != "" && entry.ProcessingDeadline != nil && now.Before(*entry.ProcessingDeadline) {
			continue
		}

		entry.ProcessorID = processorID
		entry.ProcessingDeadline = &claimDeadline
	}

	return nil
}

// GetClaimedEntries implements outbox.ProcessorStorage interface
func (e *EntryStorage) GetClaimedEntries(_ context.Context, processorID string, batchSize int) ([]outbox.Entry, error) {
	var entries []outbox.Entry

	e.lock.RLock()
	defer e.lock.RUnlock()

	for _, entry := range e.entries {
		if entry.ProcessorID != processorID {
			continue
		}

		entries = append(entries, *entry)

		if len(entries) >= batchSize {
			break
		}
	}

	return entries, nil
}

// DeleteEntries implements outbox.ProcessorStorage interface
func (e *EntryStorage) DeleteEntries(_ context.Context, entryIDs ...string) error {
	e.lock.Lock()
	defer e.lock.Unlock()

	entries := make([]*outbox.Entry, 0, len(e.entries))
	for _, entry := range e.entries {
		found := false
		for _, e := range entryIDs {
			if e == entry.ID {
				found = true
				break
			}
		}
		if found {
			continue
		}

		entries = append(entries, entry)
	}

	e.entries = entries

	return nil
}

// CountEntries is a test function for counting the number of entries currently in storage
func (e *EntryStorage) CountEntries() int {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return len(e.entries)
}
