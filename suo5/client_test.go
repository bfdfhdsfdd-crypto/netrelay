package suo5

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
)

func TestPipeWritesDataReturnedWithEOF(t *testing.T) {
	client := &Suo5Client{
		bytesPool: &sync.Pool{New: func() any { return make([]byte, 64) }},
	}
	reader := &dataAndErrorReader{data: []byte("final bytes"), err: io.EOF}
	var destination bytes.Buffer

	err := client.Pipe(reader, &destination)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Pipe returned %v, want EOF", err)
	}
	if got := destination.String(); got != "final bytes" {
		t.Fatalf("Pipe wrote %q, want final bytes", got)
	}
}

func TestPipeRejectsShortWrite(t *testing.T) {
	client := &Suo5Client{
		bytesPool: &sync.Pool{New: func() any { return make([]byte, 64) }},
	}

	err := client.Pipe(bytes.NewReader([]byte("payload")), shortWriter{})
	if !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("Pipe returned %v, want ErrShortWrite", err)
	}
}

type dataAndErrorReader struct {
	data []byte
	err  error
}

func (r *dataAndErrorReader) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, r.err
}

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}
