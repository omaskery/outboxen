package outbox

import (
	"errors"
	"time"

	"github.com/go-logr/logr"
	"github.com/jonboulle/clockwork"
)

var (
	DefaultProcessInterval = 10 * time.Second
	DefaultClaimDuration   = 2 * time.Second
	DefaultBatchSize       = 20
)

// Config configures the behaviour of the Outbox
type Config struct {
	// Clock abstracts interactions with the time package, defaults to a real clock implementation
	Clock Clock
	// Storage allows the processing task to claim, retrieve and delete Entry objects
	Storage ProcessorStorage
	// Publisher is used to publish Message objects, made from Entry objects, pulled from ProcessorStorage
	Publisher Publisher
	// ProcessInterval specifies how long the processor should spend idle without checking for work, this
	// is reset if Outbox.WakeProcessor is called
	ProcessInterval time.Duration
	// ClaimDuration specifies how long the processor will claim Entry objects in ProcessorStorage
	ClaimDuration time.Duration
	// ProcessorID is a unique identifier for any instance of the outbox, so a horizontally scaled app
	// can run many Outbox instances, each claiming Entry objects and publishing them
	ProcessorID string
	// BatchSize indicates how many Entry objects to attempt to retrieve & publish in one go
	BatchSize int
	// Logger can be provided to receive logging output
	Logger logr.Logger
}

// DefaultAndValidate ensures the configuration is valid and, where possible, provides reasonable
// default values where no value is provided
func (c *Config) DefaultAndValidate() error {
	if c.Storage == nil {
		return errors.New("no storage provided")
	}

	if c.Publisher == nil {
		return errors.New("no publisher provided")
	}

	if c.ProcessorID == "" {
		return errors.New("no processor ID provided")
	}

	if c.Clock == nil {
		c.Clock = clockwork.NewRealClock()
	}

	if c.Logger == nil {
		c.Logger = &logr.DiscardLogger{}
	}

	if c.ProcessInterval == 0 {
		c.ProcessInterval = DefaultProcessInterval
	}

	if c.ClaimDuration == 0 {
		c.ClaimDuration = DefaultClaimDuration
	}

	if c.BatchSize < 1 {
		c.BatchSize = DefaultBatchSize
	}

	return nil
}
