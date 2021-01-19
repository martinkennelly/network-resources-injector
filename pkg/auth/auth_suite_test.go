package auth

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestKeyCert(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "KeyCert test suite")
}
