package fake

import (
	"context"
	"sync"

	"github.com/go-logr/logr"

	"github.com/omaskery/outboxen/pkg/outbox"
)

// Publisher is a simple in-memory fake for publishing messages. As it doesn't actually
// deliver messages anywhere, even in-memory particularly, it's almost a mock instead of
// a fake, but it does function without configuration from the caller's point of view.
type Publisher struct {
	// Logger can be provided to receive log output
	Logger    logr.Logger
	published []outbox.Message
	lock      sync.RWMutex
}

// Publish implements the outbox.Publisher interface
func (p *Publisher) Publish(_ context.Context, messages ...outbox.Message) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Logger.Info("publishing messages", "count", len(messages))
	p.published = append(p.published, messages...)

	return nil
}

// GetPublished retrieves a copy of the published messages
func (p *Publisher) GetPublished() []outbox.Message {
	p.lock.RLock()
	defer p.lock.RUnlock()

	published := make([]outbox.Message, 0, len(p.published))
	for _, p := range p.published {
		published = append(published, p)
	}

	return published
}

// GetPublishedCount retrieves a count of published messages
func (p *Publisher) GetPublishedCount() int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return len(p.published)
}

// Clear wipes the internal state of the Publisher as if nothing had ever been
// published. It returns the previously published Message objects for convenience.
func (p *Publisher) Clear() []outbox.Message {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.Logger.Info("clearing published messages")
	result := p.published
	p.published = nil

	return result
}
