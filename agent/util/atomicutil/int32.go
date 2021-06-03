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
type AtomicInt32 struct {
	neverDirectRW_atomic_int32 int32
}

func (v *AtomicInt32) Load() int32 {
	return atomic.LoadInt32(&v.neverDirectRW_atomic_int32)
}

func (v *AtomicInt32) Store(value int32) {
	atomic.StoreInt32(&v.neverDirectRW_atomic_int32, value)
}

func (v *AtomicInt32) Add(delta int32) int32 {
	return atomic.AddInt32(&v.neverDirectRW_atomic_int32, delta)
}
