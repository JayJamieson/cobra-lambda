package wrapper

import (
	"bytes"
	"sync"
)

type threadSafeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (t *threadSafeBuffer) Write(p []byte) (n int, err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.Write(p)
}

func (t *threadSafeBuffer) String() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.buf.String()
}
