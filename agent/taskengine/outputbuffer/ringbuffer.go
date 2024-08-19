package outputbuffer

// RingBuffer is not thread safe
type RingBuffer struct {
	buf   []byte
	quota int
	start int
}

func NewRingBuffer(quota int) *RingBuffer {
	if quota < 0 {
		quota = 0
	}
	r := &RingBuffer{
		quota: quota,
	}
	return r
}

// extendCapacity try to extend capacity of r.buf to expectedCap, and
// return the capacity provided by r.buf, providedCap <= expectedCap. 
// r.buf will be resized to providedCap.
func (r *RingBuffer) extendCapacity(expectedCap int) (providedCap int) {
	defer func() {
		if len(r.buf) != providedCap {
			r.buf = r.buf[:providedCap]
		}
	}()

	if expectedCap > r.quota {
		expectedCap = r.quota
	}
	if expectedCap <= cap(r.buf) {
		providedCap = expectedCap
		return
	}

	newCap := cap(r.buf) + 1
	for newCap < expectedCap && newCap < r.quota {
		newCap *= 2
	}
	if newCap > r.quota {
		newCap = r.quota
	}
	if newCap > cap(r.buf) {
		newBuf := make([]byte, len(r.buf), newCap)
		copy(newBuf, r.buf)
		r.buf = newBuf
	}

	providedCap = expectedCap
	if providedCap > newCap {
		providedCap = newCap
	}

	return
}

func (r *RingBuffer) Put(data []byte) {
	size := len(r.buf)
	providedCap := r.extendCapacity(len(r.buf) + len(data))
	if providedCap <= len(data) {
		copy(r.buf, data[len(data)-providedCap:])
		r.start = 0
	} else {
		offset := size
		if providedCap == size {
			offset = r.start
		}
		n := copy(r.buf[offset:], data)
		if n < len(data) {
			copy(r.buf, data[n:])
		}
		r.start = (offset + len(data)) % len(r.buf)
	}
}

// Get will return r.buf self if r.start == 0, else return a copy of r.buf with normal byte sequence
func (r *RingBuffer) Get() []byte {
	if len(r.buf) == 0 {
		return nil
	}
	if r.start == 0 {
		return r.buf
	}
	newBuf := make([]byte, len(r.buf))
	copy(newBuf, r.buf[r.start:])
	copy(newBuf[len(r.buf)-r.start:], r.buf[:r.start])
	return newBuf
}

func (r *RingBuffer) Clear() {
	r.start = 0
	r.buf = r.buf[:0]
}
