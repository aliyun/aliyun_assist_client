package outputbuffer

import (
	"bytes"
	"io"
	"sync"
)

type transformFunc func([]byte) []byte

type LegacyOutputBuffer struct {
	buffer    bytes.Buffer
	mutex     sync.Mutex
	preQuota  int
	postQuota int
	hasRead   int
	hasWrote  int64

	inited   bool

	transformer     func([]byte) []byte
	transformerLock sync.RWMutex
}

func (b *LegacyOutputBuffer) Init(preQuota, postQuota int) (io.Writer, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.inited {
		return b, nil
	}
	b.inited = true

	b.preQuota = preQuota
	b.postQuota = postQuota
	b.hasWrote = 0
	b.hasRead = 0
	return b, nil
}

func (b *LegacyOutputBuffer) Uninit() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if !b.inited {
		return ErrUninited
	}
	b.inited = false

	b.buffer.Reset()
	b.hasWrote = 0
	b.hasRead = 0
	return nil
}

func (b *LegacyOutputBuffer) SetTransformer(f transformFunc) {
	b.transformerLock.Lock()
	defer b.transformerLock.Unlock()

	b.transformer = f
}

func (b *LegacyOutputBuffer) getTransformer() transformFunc {
	b.transformerLock.RLock()
	defer b.transformerLock.RUnlock()

	return b.transformer
}

func (b *LegacyOutputBuffer) Write(p []byte) (n int, err error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	n, e := b.buffer.Write(p)
	b.hasWrote += int64(n)
	return n, e
}

func (b *LegacyOutputBuffer) ReadPre() []byte {
	r := b.readPre()
	if r == nil {
		return nil
	}
	if f := b.getTransformer(); f != nil {
		return f(r)
	}
	return r
}

func (b *LegacyOutputBuffer) readPre() []byte {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	if b.hasRead >= b.preQuota {
		return nil
	}
	res := make([]byte, b.preQuota-b.hasRead)
	n, _ := b.buffer.Read(res)
	b.hasRead += n
	return res[:n]
}

func (b *LegacyOutputBuffer) ReadAll() []byte {
	r := b.readAll()
	if r == nil {
		return nil
	}
	if f := b.getTransformer(); f != nil {
		return f(r)
	}
	return r
}

func (b *LegacyOutputBuffer) readAll() []byte {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	var resPre []byte
	if b.hasRead < b.preQuota {
		sizePre := b.preQuota - b.hasRead
		if sizePre > b.buffer.Len() {
			sizePre = b.buffer.Len()
		}
		if sizePre > 0 {
			resPre = make([]byte, sizePre)
			n, _ := b.buffer.Read(resPre)
			b.hasRead += n
		}
	}
	remain := b.buffer.Len()
	couldRead := b.postQuota + b.preQuota - b.hasRead
	if couldRead > remain {
		couldRead = remain
	}
	resPost := make([]byte, couldRead)
	copy(resPost, b.buffer.Bytes()[remain-couldRead:])
	b.buffer.Reset()
	b.hasRead += len(resPost)
	if len(resPre) == 0 {
		return resPost
	} else if len(resPost) == 0 {
		return resPre
	}
	r := make([]byte, len(resPre)+len(resPost))
	copy(r, resPre)
	copy(r[len(resPre):], resPost)
	return r
}

func (b *LegacyOutputBuffer) Dropped() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.hasWrote <= int64(b.preQuota+b.postQuota) {
		return 0
	}
	return int(b.hasWrote - int64(b.preQuota) - int64(b.postQuota))
}
