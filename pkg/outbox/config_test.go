package outbox_test

import (
	"github.com/go-logr/logr"
	"github.com/jonboulle/clockwork"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"outbox/pkg/fake"
	"outbox/pkg/outbox"
)

var _ = Describe("Config", func() {
	var cfg outbox.Config

	BeforeEach(func() {
		cfg = outbox.Config{
			Publisher:   &fake.Publisher{},
			Storage:     &fake.EntryStorage{},
			ProcessorID: "test-processor-id",
		}
	})

	DescribeTable(
		"required fields are missing",
		func(mutator func()) {
			mutator()
			Expect(cfg.DefaultAndValidate()).ToNot(Succeed())
		},
		Entry("fails without storage", func() { cfg.Storage = nil }),
		Entry("fails without a publisher", func() { cfg.Publisher = nil }),
		Entry("fails without a processor ID", func() { cfg.ProcessorID = "" }),
	)

	It("correctly sets defaults", func() {
		Expect(cfg.DefaultAndValidate()).To(Succeed())

		Expect(cfg.Clock).To(Equal(clockwork.NewRealClock()))
		Expect(cfg.Logger).To(Equal(&logr.DiscardLogger{}))
		Expect(cfg.BatchSize).To(Equal(outbox.DefaultBatchSize))
		Expect(cfg.ClaimDuration).To(Equal(outbox.DefaultClaimDuration))
		Expect(cfg.ProcessInterval).To(Equal(outbox.DefaultProcessInterval))
	})
})
