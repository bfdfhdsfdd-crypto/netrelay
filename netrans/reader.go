package netrans

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var ErrReadTimeout = errors.New("read timeout")
var ErrReaderClosed = errors.New("reader has been closed")

type readResult struct {
	data   []byte
	buffer *[]byte
	err    error
}

type TimeoutReader struct {
	rc            io.ReadCloser
	t             time.Duration
	readCh        chan readResult
	buffers       sync.Pool
	pending       []byte
	pendingBuffer *[]byte
	pendingErr    error
	mu            sync.Mutex
	closed        bool
	ctx           context.Context
	cancel        func()
}

func NewTimeoutReader(ctx context.Context, r io.Reader, timeout time.Duration) io.Reader {
	return NewTimeoutReadCloser(ctx, io.NopCloser(r), timeout)
}

func NewTimeoutReadCloser(ctx context.Context, rc io.ReadCloser, timeout time.Duration) io.ReadCloser {
	if timeout < 0 {
		panic("invalid timeout")
	}
	ctx, cancel := context.WithCancel(ctx)

	tr := &TimeoutReader{
		rc:     rc,
		t:      timeout,
		readCh: make(chan readResult),
		ctx:    ctx,
		cancel: cancel,
	}
	tr.buffers.New = func() any {
		buf := make([]byte, 4096)
		return &buf
	}
	tr.startLoop()
	return tr
}

func (r *TimeoutReader) startLoop() {
	go func() {
		defer close(r.readCh)
		reader := bufio.NewReaderSize(r.rc, 4096)
		for {
			buffer := r.buffers.Get().(*[]byte)
			buf := *buffer
			n, err := reader.Read(buf)
			if n > 0 {
				select {
				case r.readCh <- readResult{data: buf[:n], buffer: buffer, err: err}:
					if err != nil {
						return
					}
				case <-r.ctx.Done():
					r.buffers.Put(buffer)
					return
				}
				continue
			}
			r.buffers.Put(buffer)
			if err != nil {
				select {
				case r.readCh <- readResult{err: err}:
				case <-r.ctx.Done():
				}
				return
			}
		}
	}()
}

func (r *TimeoutReader) Read(b []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return 0, ErrReaderClosed
	}
	if len(b) == 0 {
		return 0, nil
	}

	if len(r.pending) > 0 {
		n := copy(b, r.pending)
		r.pending = r.pending[n:]
		if len(r.pending) == 0 {
			r.buffers.Put(r.pendingBuffer)
			r.pendingBuffer = nil
		}
		return n, nil
	}
	if r.pendingErr != nil {
		err := r.pendingErr
		r.pendingErr = nil
		return 0, err
	}

	timer := time.NewTimer(r.t)
	defer timer.Stop()
	select {
	case result, ok := <-r.readCh:
		if !ok {
			r.closed = true
			return 0, ErrReaderClosed
		}
		if result.err != nil && len(result.data) == 0 {
			return 0, result.err
		}
		r.pending = result.data
		r.pendingBuffer = result.buffer
		r.pendingErr = result.err
		n := copy(b, r.pending)
		r.pending = r.pending[n:]
		if len(r.pending) == 0 {
			r.buffers.Put(r.pendingBuffer)
			r.pendingBuffer = nil
		}
		return n, nil
	case <-r.ctx.Done():
		r.closed = true
		return 0, ErrReaderClosed
	case <-timer.C:
		return 0, ErrReadTimeout
	}
}

func (r *TimeoutReader) Close() error {
	err := r.rc.Close()
	r.cancel()
	r.mu.Lock()
	if !r.closed {
		r.closed = true
		if r.pendingBuffer != nil {
			r.buffers.Put(r.pendingBuffer)
			r.pendingBuffer = nil
			r.pending = nil
		}
	}
	r.mu.Unlock()
	return err
}

type channelReader struct {
	ch  chan []byte
	mu  sync.Mutex
	buf bytes.Buffer
}

func NewChannelReader(ch chan []byte) io.Reader {
	return &channelReader{ch: ch}
}

func (c *channelReader) Read(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.buf.Len() != 0 {
		return c.buf.Read(p)
	}
	var data []byte
	var ok bool
	for {
		data, ok = <-c.ch
		if !ok {
			return 0, io.EOF
		}
		if len(data) != 0 {
			break
		}
	}
	c.buf.Reset()
	c.buf.Write(data)
	return c.buf.Read(p)
}

type channelWriterCloser struct {
	ch     chan []byte
	mu     sync.Mutex
	closed bool
	ctx    context.Context
	cancel func()
}

func NewChannelWriteCloser(ctx context.Context) (chan []byte, io.WriteCloser) {
	ch := make(chan []byte, 64)
	ctx, cancel := context.WithCancel(ctx)
	return ch, &channelWriterCloser{
		ch:     ch,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *channelWriterCloser) Close() error {
	c.cancel()

	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.closed {
		close(c.ch)
		c.closed = true
	}
	return nil
}

func (c *channelWriterCloser) Write(p []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.EOF
	}
	select {
	case c.ch <- p:
		return len(p), nil
	case <-c.ctx.Done():
		return 0, c.ctx.Err()
	default:
		// don't block on write
		return 0, errors.New("write channel is full")
	}
}

type multiReadCloser struct {
	rcs []io.ReadCloser
	r   io.Reader
}

func MultiReadCloser(rcs ...io.ReadCloser) io.ReadCloser {
	var rs []io.Reader
	for _, rc := range rcs {
		rs = append(rs, rc.(io.Reader))
	}
	return &multiReadCloser{
		rcs: rcs,
		r:   io.MultiReader(rs...),
	}
}

func (m *multiReadCloser) Read(p []byte) (n int, err error) {
	return m.r.Read(p)
}

func (m *multiReadCloser) Close() error {
	var err error
	for _, rc := range m.rcs {
		if e := rc.Close(); e != nil {
			err = e
		}
	}
	return err
}
func NoOpReader(r io.Reader) io.Reader {
	return r
}

// OffsetReader 创建一个从指定偏移量开始读取的 Reader
func OffsetReader(r io.Reader, offset int64) io.Reader {
	// 如果 Reader 支持 Seek，直接使用 Seek
	if seeker, ok := r.(io.Seeker); ok {
		_, err := seeker.Seek(offset, io.SeekStart)
		if err == nil {
			return r
		}
	}

	// 否则通过读取来跳过到指定偏移量
	_, err := io.CopyN(io.Discard, r, offset)
	if err != nil && err != io.EOF {
		return &errorReader{err: err}
	}
	return r
}

type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}
