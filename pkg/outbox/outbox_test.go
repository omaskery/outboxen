package outbox_test

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/jonboulle/clockwork"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/omaskery/outboxen/pkg/fake"
	"github.com/omaskery/outboxen/pkg/outbox"
)

var _ = Describe("Outbox", func() {
	var logger logr.Logger

	BeforeEach(func() {
		zapLogger := zap.New(zapcore.NewCore(
			zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(GinkgoWriter),
			zap.NewAtomicLevelAt(zapcore.DebugLevel),
		))
		logger = zapr.NewLogger(zapLogger)
		logger = logger.WithName("test")
	})

	It("fails to construct an invalid outbox", func() {
		_, err := outbox.New(outbox.Config{})
		Expect(err).ToNot(Succeed())
	})

	Context("with a valid outbox", func() {
		var storage *fake.EntryStorage
		var publisher *fake.Publisher
		var ctx context.Context
		var clock clockwork.FakeClock
		var cfg outbox.Config
		var ob *outbox.Outbox
		var err error

		BeforeEach(func() {
			logger.Info("initialising context")
			ctx = context.Background()

			logger.Info("initialising fake clock")
			clock = clockwork.NewFakeClock()

			logger.Info("initialising fake storage")
			storage = &fake.EntryStorage{
				Clock: clock,
			}

			logger.Info("initialising fake publisher")
			publisher = &fake.Publisher{
				Logger: logger.WithName("publisher"),
			}

			cfg = outbox.Config{
				Clock:           clock,
				Storage:         storage,
				Publisher:       publisher,
				ProcessInterval: 10 * time.Second,
				ClaimDuration:   5 * time.Second,
				ProcessorID:     "test",
				BatchSize:       5,
				Logger:          logger.WithName("outbox"),
			}

			ob = nil
		})

		JustBeforeEach(func() {
			logger.Info("creating outbox")
			ob, err = outbox.New(cfg)
			Expect(err).To(Succeed())
		})

		It("stops on context cancellation", func() {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			errChan := make(chan error, 1)
			go func() {
				var processingErr error = nil
				defer func() {
					errChan <- processingErr
				}()

				processingErr = ob.StartProcessing(ctx)
			}()

			cancel()
			Eventually(errChan, 1 * time.Second).Should(Receive(nil))
		})

		When("the outbox is pumped manually", func() {
			JustBeforeEach(func() {
				logger.Info("manually pumping outbox")
				Expect(ob.PumpOutbox(ctx)).To(Succeed())
			})

			When("the outbox was empty", func() {
				It("publishes nothing to the publisher", func() {
					Expect(publisher.GetPublishedCount()).To(BeNumerically("==", 0))
				})
			})

			When("the outbox contained a message", func() {
				var testMessage outbox.Message

				BeforeEach(func() {
					testMessage = outbox.Message{
						Key:     []byte("test-key"),
						Payload: []byte("test-payload"),
					}

					logger.Info("storing a message in the outbox")
					Expect(storage.Publish(ctx, testMessage)).To(Succeed())
				})

				It("publishes the message", func() {
					Expect(publisher.GetPublished()).To(ConsistOf(testMessage))
				})

				It("clears the outbox", func() {
					Expect(storage.CountEntries()).To(BeNumerically("==", 0))
				})
			})
		})

		When("the outbox is processing automatically", func() {
			var cancel context.CancelFunc
			var errChan chan error

			JustBeforeEach(func() {
				ctx, cancel = context.WithCancel(ctx)

				errChan = make(chan error, 1)
				go func() {
					var processingErr error = nil
					defer func() {
						errChan <- processingErr
					}()

					processingErr = ob.StartProcessing(ctx)
				}()

				logger.Info("waiting for processor to block on timer")
				clock.BlockUntil(1)
				logger.Info("processor blocking on timer")
			})

			JustAfterEach(func() {
				cancel()
				Eventually(errChan, 1 * time.Second).Should(Receive(nil))
			})

			When("a message is published", func() {
				JustBeforeEach(func() {
					logger.Info("publishing a message")
					Expect(ob.Publish(ctx, outbox.Message{})).To(Succeed())
				})

				It("publishes after the processing interval", func() {
					initialDelay := cfg.ProcessInterval / 2
					logger.Info("advancing time", "delay", initialDelay)
					clock.Advance(initialDelay)
					Expect(publisher.GetPublishedCount()).To(BeNumerically("==", 0))

					tolerance := 2 * time.Second
					remainingDelay := (cfg.ProcessInterval - initialDelay) + tolerance
					logger.Info("advancing time", "delay", remainingDelay)
					clock.Advance(remainingDelay)
					Eventually(func() int {
						return publisher.GetPublishedCount()
					}).Should(BeNumerically("==", 1))
				})

				When("the wake signal is raised", func() {
					JustBeforeEach(func() {
						logger.Info("waking the processor")
						ob.WakeProcessor()
					})

					It("publishes the message immediately", func() {
						Eventually(func() int {
							return publisher.GetPublishedCount()
						}).Should(BeNumerically("==", 1))
					})
				})
			})
		})
	})
})
