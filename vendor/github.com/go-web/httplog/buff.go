package httplog

import (
	"bytes"
	"sync"
)

var logBuffer = &sync.Pool{}

func getBuffer() *bytes.Buffer {
	if b, ok := logBuffer.Get().(*bytes.Buffer); ok {
		b.Reset()
		return b
	}
	return &bytes.Buffer{}
}

func putBuffer(b *bytes.Buffer) {
	if b.Len() <= 1<<10 {
		logBuffer.Put(b)
	}
}
