package tcp

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Config for this TCP server
type Config struct {
	Address string
	MaxConn uint32
	Timeout time.Duration
}

// ListenAndServeWithSignal binds port and the TCP handler,
// block until receive signal
func ListenAndServeWithSignal(cfg *Config, handler Handler) error {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	closeCh := make(chan struct{})
	go func() {
		<-sigCh

		closeCh <- struct{}{}
	}()

	listener, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return err
	}

	log.Printf("bing: %v, starting...", cfg.Address)
	listenAndServe(listener, handler, closeCh)

	return nil
}

func listenAndServe(listener net.Listener, handler Handler, closeChan <-chan struct{}) {
	go func() {
		<-closeChan
		_ = listener.Close()
		_ = handler.Close()
	}()

	defer func() {
		_ = listener.Close()
		_ = handler.Close()
	}()

	ctx := context.Background()
	var wg sync.WaitGroup

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}

		wg.Add(1)

		go func() {
			defer wg.Done()

			handler.Handle(ctx, conn)
		}()
	}

	wg.Wait()
}
