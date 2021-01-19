package server

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	keyCertMock "github.com/k8snetworkplumbingwg/network-resources-injector/pkg/auth/mocks"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/channel"
	srvMock "github.com/k8snetworkplumbingwg/network-resources-injector/pkg/server/mocks"
)

var _ = Describe("mutate HTTP server", func() {
	t := GinkgoT()
	Describe("Service interface implementation for HTTP server", func() {
		const to = time.Millisecond * 50
		var (
			mutateSrv *mutateServer
			mock      *srvMock.Server
		)
		BeforeEach(func() {
			mutateSrv = &mutateServer{&srvMock.Server{}, to, channel.NewChannel(chBufferSize),
				serviceName}
			mock = &srvMock.Server{}
			mutateSrv.instance = mock
		})
		Context("Run()", func() {
			It("should start server", func() {
				mock.On("Start").Return(nil)
				Expect(mutateSrv.Run()).To(BeNil())
				mock.AssertCalled(t, "Start")
			})
			It("should return error from server if startup generated an error", func() {
				expErr := errors.New("bad start of server")
				mock.On("Start").Return(expErr)
				Expect(mutateSrv.Run()).To(Equal(expErr))
			})
		})
		Context("Quit()", func() {
			It("should stop server", func() {
				mock.On("Stop", to).Return(nil)
				mutateSrv.status.Close() // Close in advance of call to ensure we do not get timeout error
				Expect(mutateSrv.Quit()).To(BeNil())
				mock.AssertCalled(t, "Stop", to)
			})
			It("should return error if shutdown generated an error", func() {
				expErr := errors.New("bad stop of server")
				mock.On("Stop", to).Return(expErr)
				mutateSrv.status.Close() // Close in advance of call to ensure we do not get timeout error
				Expect(mutateSrv.Quit()).To(Equal(expErr))
			})
		})
	})
	Describe("creation of new mutate server", func() {
		const (
			address  = "127.0.0.1"
			port     = 12345
			to       = time.Millisecond * 2
			insecure = false
		)
		var (
			pool    *keyCertMock.ClientCAPool
			keyPair *keyCertMock.Identity
		)
		BeforeEach(func() {
			pool = &keyCertMock.ClientCAPool{}
			pool.On("GetCertPool").Return(nil)
			keyPair = &keyCertMock.Identity{}
			keyPair.On("GetCertificateFunc").Return(nil)
		})
		Context("NewMutateServer()", func() {
			It("should retrieve cert pool", func() {
				NewMutateServer(address, port, insecure, to, to, to, to, pool, keyPair)
				pool.AssertCalled(t, "GetCertPool")
			})
			It("should retrieve certificate function", func() {
				NewMutateServer(address, port, insecure, to, to, to, to, pool, keyPair)
				keyPair.AssertCalled(t, "GetCertificateFunc")
			})
		})
	})
})
