// Code generated by mockery v1.0.0
package mocks

import mock "github.com/stretchr/testify/mock"
import v1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"

// ActionSetsGetter is an autogenerated mock type for the ActionSetsGetter type
type ActionSetsGetter struct {
	mock.Mock
}

// ActionSets provides a mock function with given fields: namespace
func (_m *ActionSetsGetter) ActionSets(namespace string) v1alpha1.ActionSetInterface {
	ret := _m.Called(namespace)

	var r0 v1alpha1.ActionSetInterface
	if rf, ok := ret.Get(0).(func(string) v1alpha1.ActionSetInterface); ok {
		r0 = rf(namespace)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(v1alpha1.ActionSetInterface)
		}
	}

	return r0
}
