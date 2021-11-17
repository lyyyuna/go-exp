package streamchan

import (
	"bytes"
	"io"
	"testing"
	"time"
)

func TestBasicReadWrite(t *testing.T) {
	send := []byte("hello world")
	c := make(chan *Chunk)

	writer := NewChanWriter(c)
	reader := NewChanReader(c)

	buf := make([]byte, len(send))

	go writer.Write(send)
	n, err := reader.Read(buf)

	if err != nil {
		t.Fatalf("read has error: %v", err)
	}

	if n != len(send) {
		t.Fatalf("read wrong number of bytes: %v, expected: %v", n, len(send))
	}

	if !bytes.Equal(send, buf) {
		t.Fatalf("data not equal")
	}

	writer.Close()

	n, err = reader.Read(buf)
	if err != io.EOF {
		t.Fatal("read from closed stream chan should return EOF", err)
	}

	if n != 0 {
		t.Fatal("read still return data")
	}
}

func TestReadMoreThanWrite(t *testing.T) {
	send := []byte("hello world")
	c := make(chan *Chunk)

	writer := NewChanWriter(c)
	reader := NewChanReader(c)

	buf := make([]byte, len(send)+10)

	go writer.Write(send)
	n, err := reader.Read(buf)

	if err != nil {
		t.Fatalf("read has error: %v", err)
	}

	if n != len(send) {
		t.Fatalf("read wrong number of bytes: %v, expected: %v", n, len(send))
	}

	if !bytes.Equal(send, buf[:n]) {
		t.Fatalf("data not equal")
	}

	writer.Close()

	n, err = reader.Read(buf)
	if err != io.EOF {
		t.Fatal("read from closed stream chan should return EOF", err)
	}

	if n != 0 {
		t.Fatal("read still return data")
	}
}

func TestReadLessThanWrite(t *testing.T) {
	send := []byte("hello world")
	c := make(chan *Chunk)
	writer := NewChanWriter(c)
	reader := NewChanReader(c)

	go writer.Write(send)
	buf := make([]byte, 6)
	n, err := reader.Read(buf)

	if err != nil {
		t.Fatal("could not read from the channel", err)
	}

	if n != len(buf) {
		t.Fatalf("read wrong number from steam: %v, expected: %v", n, len(buf))
	}

	if !bytes.Equal(buf, send[:len(buf)]) {
		t.Fatalf("got wrong message %v", string(buf))
	}

	writer.Close()

	n, err = reader.Read(buf)

	if err != nil {
		t.Fatal("could not read from the channel")
	}

	if n != len(send)-len(buf) {
		t.Fatalf("read wrong number from steam: %v, expected: %v", n, len(send)-len(buf))
	}

	if !bytes.Equal(buf[:n], send[len(buf):]) {
		t.Fatal("got wrong message", string(buf[:n]))
	}

	n, err = reader.Read(buf)
	if err != io.EOF {
		t.Fatal("read from closed stream chan should return EOF", err)
	}

	if n != 0 {
		t.Fatal("read still return data")
	}
}

func TestMultiReadWrite(t *testing.T) {
	send := []byte("hello world, this message is longer")
	c := make(chan *Chunk)
	writer := NewChanWriter(c)
	reader := NewChanReader(c)

	go func() {
		writer.Write(send[:9])
		writer.Write(send[9:19])
		writer.Write(send[19:])
		writer.Close()
	}()

	buf := make([]byte, 10)
	for readIndex := 0; readIndex < len(send); {
		n, err := reader.Read(buf)
		if err != nil {
			t.Fatal("could not read from the stream channel", err)
		}
		if !bytes.Equal(buf[:n], send[readIndex:readIndex+n]) {
			t.Fatal("got wrong message:", string(buf[:n]))
		}
		readIndex += n
	}

	n, err := reader.Read(buf)
	if err != io.EOF {
		t.Fatal("read from closed stream chan should return EOF", err)
	}

	if n != 0 {
		t.Fatal("read still return data")
	}
}

func TestMultiWriteWithCopy(t *testing.T) {
	send := []byte("hello world, this message is longer")
	c := make(chan *Chunk)
	writer := NewChanWriter(c)
	reader := NewChanReader(c)

	go func() {
		writer.Write(send[:9])
		writer.Write(send[9:19])
		writer.Write(send[19:])
		writer.Close()
	}()

	buf := new(bytes.Buffer)
	n, err := io.Copy(buf, reader)

	if err != nil {
		t.Fatal("copy failed", err)
	}

	if int(n) != len(send) {
		t.Fatalf("read wrong number from steam: %v, expected: %v", n, len(send))
	}

	if !bytes.Equal(buf.Bytes(), send) {
		t.Fatal("got wrong message", buf.String())
	}
}

func TestReadInterrupt(t *testing.T) {
	send := []byte("hello world")
	c := make(chan *Chunk)
	interrupt := make(chan struct{})

	reader := NewChanReader(c)
	reader.SetInterrupt(interrupt)
	writer := NewChanWriter(c)

	go writer.Write(send)
	buf := make([]byte, len(send))
	reader.Read(buf)

	go func() {
		time.Sleep(50 * time.Millisecond)
		interrupt <- struct{}{}
	}()

	n, err := reader.Read(buf)
	if err != ErrReadInterrupted {
		t.Fatal("should return interrupt error, but got", err)
	}

	if n != 0 {
		t.Fatal("no data should read, but got", n)
	}

	// try again
	go writer.Write(send)
	n, err = reader.Read(buf)

	if err != nil {
		t.Fatalf("read has error: %v", err)
	}

	if n != len(send) {
		t.Fatalf("read wrong number of bytes: %v, expected: %v", n, len(send))
	}

	if !bytes.Equal(send, buf) {
		t.Fatalf("data not equal")
	}

	writer.Close()
}
