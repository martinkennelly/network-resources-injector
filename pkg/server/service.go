package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"

	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/auth"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/channel"
	"github.com/k8snetworkplumbingwg/network-resources-injector/pkg/service"
)

const (
	startupInterval = time.Millisecond * 50
	endpoint        = "/mutate"
	chBufferSize    = 1
	serviceName     = "network resources injector /mutate server"
)

type mutateServer struct {
	instance Server
	timeout  time.Duration
	status   *channel.Channel
	name     string
}

// NewMutateServer generate a new server to serve endpoint /mutate. Server will only serve /mutate endpoint and POST
// HTTP verb. When arg insecure is false, it forces client certificate validation based on CAs in argument pool
// otherwise no client certificate validation is required. Various timeout args exist to prevent DOS. Arg keypair contains
// server TLS key/cert
func NewMutateServer(address string, port int, insecure bool, readT, writeT, readHT, to time.Duration, pool auth.ClientCAPool,
	keyCert auth.Identity) service.Service {
	if insecure {
		glog.Warning("HTTP server is configured not to require client certificate")
	}
	srvAddr := fmt.Sprintf("%s:%d", address, port)
	glog.Infof("HTTP server address and port: '%s'", srvAddr)
	mux := http.NewServeMux()
	mux.HandleFunc("/mutate", httpServerHandler)

	httpServer := &http.Server{
		Addr:              srvAddr,
		Handler:           mux,
		ReadTimeout:       readT,
		WriteTimeout:      writeT,
		MaxHeaderBytes:    1 << 20,
		ReadHeaderTimeout: readHT,
		TLSConfig: &tls.Config{
			ClientAuth:               getClientAuth(insecure),
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384},
			ClientCAs:                pool.GetCertPool(),
			PreferServerCipherSuites: true,
			InsecureSkipVerify:       false,
			CipherSuites: []uint16{
				// tls 1.2
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				// tls 1.3 configuration not supported
			},
			GetCertificate: keyCert.GetCertificateFunc(),
		},
	}
	return &mutateServer{&server{httpServer}, to, nil, serviceName}
}

// Run starts HTTP server in goroutine, waits a period of time and returns any potential errors from server start
func (mSrv *mutateServer) Run() error {
	var httpSrvMsg error
	glog.Info("starting HTTP server")
	mSrv.status = channel.NewChannel(chBufferSize)

	go func() {
		mSrv.status.Open()
		defer mSrv.status.Close()
		if httpSrvMsg = mSrv.instance.Start(); httpSrvMsg != nil &&
			httpSrvMsg != http.ErrServerClosed {
			glog.Errorf("HTTP server message: '%s'", httpSrvMsg.Error())
		}
		glog.Info("HTTP server finished")
	}()
	// give server time to start and return error if startup failed
	time.Sleep(startupInterval)
	return httpSrvMsg
}

// Quit attempts to shutdown HTTP server and waits for HTTP server status channel to close
func (mSrv *mutateServer) Quit() error {
	glog.Info("terminating HTTP server")
	if err := mSrv.instance.Stop(mSrv.timeout); err != nil && err != http.ErrServerClosed {
		return err
	}
	return mSrv.status.WaitUntilClosed(mSrv.timeout)
}

// StatusSignal returns a channel which indicates whether mutate server has ended when channel closes
func (mSrv *mutateServer) StatusSignal() chan struct{} {
	return mSrv.status.GetCh()
}

// GetName returns service name
func (mSrv *mutateServer) GetName() string {
	return mSrv.name
}
