// Code generated by mockery v2.5.1. DO NOT EDIT.

package mocks

import (
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// Server is an autogenerated mock type for the Server type
type Server struct {
	mock.Mock
}

// Start provides a mock function with given fields:
func (_m *Server) Start() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Stop provides a mock function with given fields: timeout
func (_m *Server) Stop(timeout time.Duration) error {
	ret := _m.Called(timeout)

	var r0 error
	if rf, ok := ret.Get(0).(func(time.Duration) error); ok {
		r0 = rf(timeout)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
