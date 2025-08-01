// Code generated by mockery v2.53.4. DO NOT EDIT.

package mocks

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// Hasher is an autogenerated mock type for the Hasher type
type Hasher struct {
	mock.Mock
}

type Hasher_Expecter struct {
	mock *mock.Mock
}

func (_m *Hasher) EXPECT() *Hasher_Expecter {
	return &Hasher_Expecter{mock: &_m.Mock}
}

// Hash provides a mock function with given fields: ctx, password
func (_m *Hasher) Hash(ctx context.Context, password []byte) (string, error) {
	ret := _m.Called(ctx, password)

	if len(ret) == 0 {
		panic("no return value specified for Hash")
	}

	var r0 string
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, []byte) (string, error)); ok {
		return rf(ctx, password)
	}
	if rf, ok := ret.Get(0).(func(context.Context, []byte) string); ok {
		r0 = rf(ctx, password)
	} else {
		r0 = ret.Get(0).(string)
	}

	if rf, ok := ret.Get(1).(func(context.Context, []byte) error); ok {
		r1 = rf(ctx, password)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Hasher_Hash_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Hash'
type Hasher_Hash_Call struct {
	*mock.Call
}

// Hash is a helper method to define mock.On call
//   - ctx context.Context
//   - password []byte
func (_e *Hasher_Expecter) Hash(ctx interface{}, password interface{}) *Hasher_Hash_Call {
	return &Hasher_Hash_Call{Call: _e.mock.On("Hash", ctx, password)}
}

func (_c *Hasher_Hash_Call) Run(run func(ctx context.Context, password []byte)) *Hasher_Hash_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].([]byte))
	})
	return _c
}

func (_c *Hasher_Hash_Call) Return(_a0 string, _a1 error) *Hasher_Hash_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *Hasher_Hash_Call) RunAndReturn(run func(context.Context, []byte) (string, error)) *Hasher_Hash_Call {
	_c.Call.Return(run)
	return _c
}

// NewHasher creates a new instance of Hasher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewHasher(t interface {
	mock.TestingT
	Cleanup(func())
}) *Hasher {
	mock := &Hasher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
