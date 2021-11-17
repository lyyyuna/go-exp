package streamchan

import (
	"fmt"
	"io"
	"time"
)

// Chunk is a slice of data needs to sent/received from channel
type Chunk struct {
	Data      []byte
	Timestamp time.Time
}

// Writer implements the io.WriteCloser interface,
// data will be sent to the internal channel
type Writer struct {
	output chan<- *Chunk
}

var _ io.WriteCloser = (*Writer)(nil)

// NewChanWriter creates a new channel writer
func NewChanWriter(output chan<- *Chunk) *Writer {
	return &Writer{output}
}

// Write writes data into channel as one chunk
func (w *Writer) Write(buf []byte) (int, error) {
	data := &Chunk{make([]byte, len(buf)), time.Now()}
	copy(data.Data, buf)

	w.output <- data
	return len(buf), nil
}

// Close closes the internal channel, fail if already closed
func (w *Writer) Close() error {
	close(w.output)

	return nil
}

// Reader implements the io.Reader interface,
// can read data from the channel, also can be interruped
type Reader struct {
	input     <-chan *Chunk
	interrupt <-chan struct{}
	buffer    []byte
}

var _ io.Reader = (*Reader)(nil)

var ErrReadInterrupted = fmt.Errorf("read innterrupted")

// NewChanReader creates a new channel reader
func NewChanReader(input <-chan *Chunk) *Reader {
	return &Reader{
		input,
		make(<-chan struct{}),
		[]byte{},
	}
}

func (r *Reader) SetInterrupt(interrupt <-chan struct{}) {
	r.interrupt = interrupt
}

// Read reads from the channel, read block until data is available.
// if out slice is not big enough, the remaining data will be kept in the buffer
func (r *Reader) Read(out []byte) (int, error) {
	if r.buffer == nil {
		return 0, io.EOF
	}

	n := copy(out, r.buffer)
	r.buffer = r.buffer[n:]
	if len(out) <= len(r.buffer) {
		return n, nil
	} else if n > 0 {
		// buffer is clean now, needs read more data from the channel
		select {
		case p := <-r.input:
			if p == nil { // channel is closed
				r.buffer = nil
				if n > 0 {
					return n, nil
				} else {
					return 0, io.EOF
				}
			}
			n2 := copy(out[n:], p.Data)
			r.buffer = p.Data[n2:]
			return n + n2, nil
		default:
			return n, nil
		}
	}

	var p *Chunk
	select {
	case p = <-r.input:
	case <-r.interrupt:
		r.buffer = r.buffer[:0]
		return 0, ErrReadInterrupted
	}
	if p == nil {
		r.buffer = nil
		return 0, io.EOF
	}
	n2 := copy(out[n:], p.Data)
	r.buffer = p.Data[n2:]

	return n + n2, nil
}
