package atomicutil

import (
	"sync/atomic"
)

// AtomicBoolean provides boolean type with atomic opeartion, based on uint32
// primitive type with atomic operators in sync/atomic package.
//
// The zero-value of AtomicBoolean type relies on and is the zero-value of
// underlying uint32 primitive type, i.e., 0. This also allows users to
// instantiate an AtomicBoolean value without explicitly calling any constructor
// function.
type AtomicBoolean struct {
	neverDirectRW_atomic_boolean uint32
}

func (v *AtomicBoolean) IsSet() bool {
	return atomic.LoadUint32(&v.neverDirectRW_atomic_boolean) == 1
}

func (v *AtomicBoolean) Set() {
	atomic.StoreUint32(&v.neverDirectRW_atomic_boolean, 1)
}

func (v *AtomicBoolean) Clear() {
	atomic.StoreUint32(&v.neverDirectRW_atomic_boolean, 0)
}

func (v *AtomicBoolean) CompareAndSwap(old, new bool) (bool) {
	var oldI, newI uint32
	if old {
		oldI = 1
	}
	if new {
		newI = 1
	}
	return atomic.CompareAndSwapUint32(&v.neverDirectRW_atomic_boolean, oldI, newI)
}