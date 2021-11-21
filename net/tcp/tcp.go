package tcp

import (
	"context"
	"net"
)

// HandlerFunc represents the application handler function
type HandleFunc func(ctx context.Context, conn net.Conn)

// Handler is interface for TCP server, application can be constructed
// on this
type Handler interface {
	Handle(ctx context.Context, conn net.Conn)
	Close() error
}
