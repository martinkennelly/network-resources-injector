// Code generated by mockery v2.5.1. DO NOT EDIT.

package mocks

import (
	tls "crypto/tls"

	mock "github.com/stretchr/testify/mock"
)

// Identity is an autogenerated mock type for the Identity type
type Identity struct {
	mock.Mock
}

// GetCertPath provides a mock function with given fields:
func (_m *Identity) GetCertPath() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// GetCertificateFunc provides a mock function with given fields:
func (_m *Identity) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	ret := _m.Called()

	var r0 func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	if rf, ok := ret.Get(0).(func() func(*tls.ClientHelloInfo) (*tls.Certificate, error)); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(func(*tls.ClientHelloInfo) (*tls.Certificate, error))
		}
	}

	return r0
}

// GetKeyPath provides a mock function with given fields:
func (_m *Identity) GetKeyPath() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// Reload provides a mock function with given fields:
func (_m *Identity) Reload() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
