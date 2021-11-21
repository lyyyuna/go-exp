package tcp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lyyyuna/go-exp/sync/wait"
)

type EchoHanlder struct {
	conns   sync.Map
	closing int32
}

var _ Handler = (*EchoHanlder)(nil)

type EchoClient struct {
	conn net.Conn
	wg   wait.WaitGroup
}

func (ec *EchoClient) Close() error {
	ec.wg.WaitWithTimeout(10 * time.Second)

	return ec.conn.Close()
}

func NewEchoHandler() *EchoHanlder {
	return &EchoHanlder{}
}

func (h *EchoHanlder) Handle(ctx context.Context, conn net.Conn) {
	// check if closing
	if atomic.LoadInt32(&h.closing) == 1 {
		conn.Close()
	}

	client := &EchoClient{
		conn: conn,
	}
	h.conns.Store(client, struct{}{})

	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				h.conns.Delete(client)
			} else {
				log.Println("[warn] conn closed", err)
			}
			return
		}
		client.wg.Add(1)
		conn.Write([]byte(msg))
		client.wg.Done()
	}
}

func (h *EchoHanlder) Close() error {
	// set closing
	atomic.StoreInt32(&h.closing, 1)

	h.conns.Range(func(key, value interface{}) bool {
		client := key.(*EchoClient)
		client.Close()
		return true
	})

	return nil
}

func TestListenAndServe(t *testing.T) {

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}

	addr := listener.Addr().String()
	closech := make(chan struct{})
	go listenAndServe(listener, NewEchoHandler(), closech)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		sent := fmt.Sprintf("sent: %v", i)
		_, err := conn.Write([]byte(sent + "\n"))
		if err != nil {
			t.Fatal(err)
		}

		reader := bufio.NewReader(conn)
		line, _, err := reader.ReadLine()
		if err != nil {
			t.Fatal(err)
		}

		if string(line) != sent {
			t.Errorf("sent and received not match, sent: %v, recv: %v", sent, string(line))
		}
	}

	conn.Close()

	closech <- struct{}{}

	time.Sleep(1 * time.Second)
}
