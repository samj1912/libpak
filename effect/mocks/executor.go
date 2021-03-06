// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	effect "github.com/paketo-buildpacks/libpak/effect"
	mock "github.com/stretchr/testify/mock"
)

// Executor is an autogenerated mock type for the Executor type
type Executor struct {
	mock.Mock
}

// Execute provides a mock function with given fields: execution
func (_m *Executor) Execute(execution effect.Execution) error {
	ret := _m.Called(execution)

	var r0 error
	if rf, ok := ret.Get(0).(func(effect.Execution) error); ok {
		r0 = rf(execution)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
