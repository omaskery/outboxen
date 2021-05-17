package outbox_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestOutbox(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Outbox Suite")
}
