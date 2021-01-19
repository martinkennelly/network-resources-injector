// Code generated by mockery v2.5.1. DO NOT EDIT.

package mocks

import (
	x509 "crypto/x509"

	mock "github.com/stretchr/testify/mock"
)

// ClientCAPool is an autogenerated mock type for the ClientCAPool type
type ClientCAPool struct {
	mock.Mock
}

// GetCertPool provides a mock function with given fields:
func (_m *ClientCAPool) GetCertPool() *x509.CertPool {
	ret := _m.Called()

	var r0 *x509.CertPool
	if rf, ok := ret.Get(0).(func() *x509.CertPool); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*x509.CertPool)
		}
	}

	return r0
}

// Load provides a mock function with given fields:
func (_m *ClientCAPool) Load() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}