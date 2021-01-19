package auth

import (
	"errors"
	"io/ioutil"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/auth/mocks"
)

var _ = Describe("cert & key watcher", func() {
	t := GinkgoT()
	const (
		to        = time.Millisecond * 10
		interval  = to
		keyFName  = "nri-watcher-test-key"
		certFName = "nri-watcher-test-cert"
		TempDir   = "/tmp"
		svcName   = "key cert watcher"
	)
	var (
		keyPair *mocks.Identity
		kcw     *keyCertUpdate
		certF   *os.File
		keyF    *os.File
	)
	BeforeEach(func() {
		keyPair = &mocks.Identity{}
		certF, _ = ioutil.TempFile(TempDir, certFName) // need to create files because watcher isn't a dep injected
		keyF, _ = ioutil.TempFile(TempDir, keyFName)
		kcw = &keyCertUpdate{nil, nil, to, keyPair, svcName}
		keyPair.On("GetCertPath").Return(certF.Name())
		keyPair.On("GetKeyPath").Return(keyF.Name())
	})

	AfterEach(func() {
		_ = kcw.Quit()
		_ = os.Remove(certF.Name())
		_ = os.Remove(keyF.Name())
	})

	Context("Run()", func() {
		It("should retrieve cert and key path", func() {
			_ =kcw.Run()
			keyPair.AssertCalled(t, "GetCertPath")
			keyPair.AssertCalled(t, "GetKeyPath")
		})
		It("should return error if cert doesn't exist", func() {
			_ = os.Remove(certF.Name())
			Expect(kcw.Run().Error()).To(ContainSubstring("cert file does not exist"))
		})
		It("should return error if key doesn't exist", func() {
			_ = os.Remove(keyF.Name())
			Expect(kcw.Run().Error()).To(ContainSubstring("key file does not exist"))
		})
		It("should not reload cert/key if only key is altered", func() {
			_ = kcw.Run()
			_ = os.Chtimes(certF.Name(), time.Now(), time.Now()) // touch file
			time.Sleep(interval)                             // wait for Identity function to be possibly called
			keyPair.AssertNotCalled(t, "Reload")
		})
		It("should not reload cert/key if only cert is altered", func() {
			_ = kcw.Run()
			_ = os.Chtimes(keyF.Name(), time.Now(), time.Now()) // touch file
			time.Sleep(interval)                            // wait for Identity function to be possibly called
			keyPair.AssertNotCalled(t, "Reload")
		})
		It("should reload cert/key if cert and key are altered", func() {
			keyPair.On("Reload").Return(nil)
			_ = kcw.Run()
			_ = os.Chtimes(certF.Name(), time.Now(), time.Now()) // touch file
			_ = os.Chtimes(keyF.Name(), time.Now(), time.Now())
			time.Sleep(interval) // wait for Identity function to be called
			keyPair.AssertExpectations(t)
		})
		It("should terminate watcher when reload fails", func() {
			keyPair.On("Reload").Return(errors.New("failed to reload keys"))
			_ = kcw.Run()
			_ = os.Chtimes(certF.Name(), time.Now(), time.Now()) // touch file
			_ = os.Chtimes(keyF.Name(), time.Now(), time.Now())
			time.Sleep(interval) // wait for Identity function to be called
			Expect(kcw.status.IsOpen()).To(BeFalse())
		})
		It("should tolerate restart", func() {
			_ = kcw.Run()
			_ = kcw.Quit()
			Expect(kcw.status.IsOpen()).To(BeFalse())
			_ = kcw.Run() // restart
			Expect(kcw.status.IsOpen()).To(BeTrue())
			_ = kcw.Quit()
			Expect(kcw.status.IsOpen()).To(BeFalse())
		})
	})

	Context("Quit()", func() {
		It("should terminate watcher", func() {
			_ = kcw.Run()
			time.Sleep(interval)
			Expect(kcw.status.IsOpen()).To(BeTrue()) // ensure it is running before test
			Expect(kcw.Quit()).To(BeNil())
			Expect(kcw.status.IsClosed()).To(BeTrue())
		})
	})
})
