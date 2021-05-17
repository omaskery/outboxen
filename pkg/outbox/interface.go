package outbox

import (
	"context"
	"fmt"
	"time"
)

// Clock abstracts interactions with the time package to facilitate testing
type Clock interface {
	Now() time.Time
	After(c time.Duration) <-chan time.Time
}

// Entry is an entry in the Outbox
type Entry struct {
	// ID is a unique identifier for any given Outbox Entry, typically a database primary key
	ID string
	// CreatedAt describes when the Entry was created, currently serves no functional purpose
	// and is kept as a possibly useful debug value
	CreatedAt time.Time
	// Key to be included in the published Message
	Key []byte
	// Payload to be included in the published Message
	Payload []byte
	// ProcessorID is the unique id of the processor currently claiming this Entry for publishing
	ProcessorID string
	// ProcessingDeadline is the deadline after which any outstanding claims on this Entry are void,
	// and the Entry may be claimed by another processor
	ProcessingDeadline *time.Time
}

// ProcessorStorage is the Outbox's interaction with persistence, typically a database
type ProcessorStorage interface {
	// ClaimEntries attempts to update all claimable entries as belonging to the calling processor
	ClaimEntries(ctx context.Context, processorID string, claimDeadline time.Time) error
	// GetClaimedEntries returns a batch of entries currently belonging to the calling processor
	GetClaimedEntries(ctx context.Context, processorID string, batchSize int) ([]Entry, error)
	// DeleteEntries deletes the entries as specified by their Entry.ID
	DeleteEntries(ctx context.Context, entryIDs ...string) error
}

// Message is what will be published over some pubsub/streaming system
type Message struct {
	// Key is an optional value primarily used in streaming systems that partition
	// published messages by keys to facilitate in-order delivery and load balancing
	Key []byte
	// Payload is the actual message contents that should be published
	Payload []byte
}

// Publisher is something that can take a batch of Message objects and attempt to publish them.
// Note that this interface is useful both as:
//   - The destination that the Outbox will write Message objects to, e.g. some external pubsub/stream
//   - A promise from your application's persistence layer which - as part of some ongoing transaction -
//     will write the given Message objects as Entry objects to the underlying ProcessorStorage for
//     later publishing
type Publisher interface {
	// Publish attempts to write the given messages to a destination. It may return a PublishError
	// to indicate which messages were published successfully.
	Publish(ctx context.Context, messages ...Message) error
}

// PublishError allows callers to understand which Message objects, if any, were sent successfully
type PublishError struct {
	// Errors correlates one-to-one with the Message values passed to Publisher.Publish - if a message
	// was sent successfully it will have a nil entry, otherwise it will be an error value
	Errors []error
}

// ErrorCount counts how many messages failed to publish
func (p *PublishError) ErrorCount() (count int) {
	for _, err := range p.Errors {
		if err != nil {
			count += 1
		}
	}
	return
}

// Error provides a brief string summary to implement the Error interface
func (p *PublishError) Error() string {
	return fmt.Sprintf("failed to publish %v/%v messages", p.ErrorCount(), len(p.Errors))
}
