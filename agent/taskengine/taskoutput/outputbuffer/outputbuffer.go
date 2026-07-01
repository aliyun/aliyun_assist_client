package outputbuffer

import (
	"errors"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

type OutputBuf interface {
	// Need to initialize before use
	Init(preQuota, postQuota int) (io.Writer, error)
	// Need to be reset after use
	Uninit() error
	// Read pre-part content
	ReadPre() []byte
	// Read all content
	ReadAll() []byte
	// Read all content from start. 
	// ReadPre and ReadAll cannot read content repeatedly, but ReadAllFromStart can
	ReadAllFromStart() []byte
	// How many bytes were dropped
	Dropped() int
}

// OutputBuffer is specially used to store the output content of task execution.
// It only stores the beginning and end of the output.
type OutputBuffer struct {
	r, w *os.File

	postPart  *RingBuffer
	postQuota int
	postReaded bool
	lockPost  sync.Mutex

	prePart      []byte
	preQuota     int
	prePartEnd   atomic.Int32
	prePartStart atomic.Int32

	putCount atomic.Int64

	inited   atomic.Bool
	scanDone chan int

	err error
}

var (
	ErrBadReadCount = errors.New("Read returned impossible count")
	ErrUninited     = errors.New("Uninited")
)

func (b *OutputBuffer) Init(preQuota, postQuota int) (io.Writer, error) {
	if !b.inited.CompareAndSwap(false, true) {
		return b.w, nil
	}

	var err error
	b.r, b.w, err = os.Pipe()
	if err != nil {
		b.inited.Store(false)
		return nil, err
	}
	b.preQuota, b.postQuota = preQuota, postQuota

	b.lockPost.Lock()
	defer b.lockPost.Unlock()
	b.postPart = NewRingBuffer(b.postQuota)
	b.prePartEnd.Store(0)
	b.prePartStart.Store(0)
	b.prePart = make([]byte, b.preQuota)

	b.scanDone = make(chan int)
	b.err = nil
	go b.scan()
	return b.w, nil
}

// clean all buf and close pipe, wait scan return
func (b *OutputBuffer) Uninit() error {
	if !b.inited.CompareAndSwap(true, false) {
		return ErrUninited
	}
	b.w.Close()
	b.r.Close() // repeat close is ok
	<-b.scanDone

	b.lockPost.Lock()
	defer b.lockPost.Unlock()
	b.postPart.Clear()
	b.postPart = nil
	b.postReaded = false

	b.prePartEnd.Store(0)
	b.prePartStart.Store(0)
	b.prePart = nil

	b.putCount.Store(0)

	b.scanDone = nil

	return b.err
}

// Scan will run a loop to read content from r and store content into prePart or postPart, until EOF or
// other error occur when read.
func (b *OutputBuffer) scan() {
	defer close(b.scanDone)
	defer b.r.Close() // close b.r to prevent b.w block

	var end, start int
	var prePartFull bool
	const maxConsecutiveEmptyReads = 100
	buf := make([]byte, 32*1024)

	for {
		if !prePartFull {
			preEnd := int(b.prePartEnd.Load())
			if preEnd >= len(b.prePart) {
				prePartFull = true
			} else {
				n := copy(b.prePart[preEnd:], buf[start:end])
				b.prePartEnd.Add(int32(n))
				b.putCount.Add(int64(n))
				start += n
			}
		}
		if start < end {
			b.lockPost.Lock()
			b.postPart.Put(buf[start:end])
			b.putCount.Add(int64(end - start))
			start = end
			b.lockPost.Unlock()
		}
		if end == len(buf) {
			start, end = 0, 0
		}
		for loop := 0; ; {
			n, err := b.r.Read(buf[end:])
			if n < 0 || len(buf)-end < n {
				b.err = ErrBadReadCount
				return
			}
			end += n
			if err != nil {
				if !errors.Is(err, io.EOF) && !errors.Is(err, os.ErrClosed) {
					b.err = err
				}
				return
			}
			if n > 0 {
				break
			}
			if loop > maxConsecutiveEmptyReads {
				b.err = io.ErrNoProgress
			}
		}
	}
}

func (b *OutputBuffer) readPost(fromStart bool) []byte {
	b.lockPost.Lock()
	defer b.lockPost.Unlock()
	if b.postPart != nil {
		if fromStart {
			return b.postPart.Get()	
		}
		if b.postReaded {
			return nil
		}
		b.postReaded = true
		return b.postPart.Get()
	}
	return nil
}

func (b *OutputBuffer) readPre(fromStart bool) []byte {
	if b.prePart == nil {
		return nil
	}
	if fromStart {

	}

	// b.prePart may be set to nil by a goroutine while another goroutine is
	// reading, so read from tmpPrePart instead of b.prePart

	tmpPrePart := b.prePart
	if tmpPrePart == nil {
		return nil
	}
	preEnd := b.prePartEnd.Load()
	preStart := b.prePartStart.Load()
	if fromStart {
		return tmpPrePart[:preEnd]
	}

	if preEnd <= preStart {
		return nil
	}
	if b.prePartStart.CompareAndSwap(preStart, preEnd) {
		if preEnd == int32(len(tmpPrePart)) {
			res := tmpPrePart[preStart:preEnd]
			// b.prePart = nil
			return res
		}
		return tmpPrePart[preStart:preEnd]
	}
	return nil
}

// Read from prePart, the content in b.prePart will only returned once,
// b.prePart will be set to nil after all content returned
func (b *OutputBuffer) ReadPre() []byte {
	r := b.readPre(false)
	if r == nil {
		return nil
	}
	return r
}

// Read from b.prePart and b.postPart, the content will only returned once
func (b *OutputBuffer) ReadAll() []byte {
	r := b.readAll(false)
	if r == nil {
		return nil
	}
	return r
}

func (b *OutputBuffer) ReadAllFromStart() []byte {
	r := b.readAll(true)
	if r == nil {
		return nil
	}
	return r
}

func (b *OutputBuffer) readAll(fromStart bool) (res []byte) {
	prePart := b.readPre(fromStart)
	postPart := b.readPost(fromStart)

	if prePart == nil {
		return postPart
	}
	if postPart == nil {
		return prePart
	}

	res = make([]byte, len(prePart)+len(postPart))
	n := copy(res, prePart)
	copy(res[n:], postPart)
	return
}

func (b *OutputBuffer) Dropped() int {
	if !b.inited.Load() {
		return 0
	}
	putCount := b.putCount.Load()
	if putCount <= int64(b.preQuota+b.postQuota) {
		return 0
	}
	return int(putCount - int64(b.preQuota) - int64(b.postQuota))
}
