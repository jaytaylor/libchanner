package libchanner

import (
	"net"
	"time"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
)

type Chan struct {
	Sender       libchan.Sender
	Receiver     libchan.Receiver
	RemoteSender libchan.Sender
	Transport    libchan.Transport
}

// DialChan opens a new libchan SPDY channel to the specified address
// (host:port).
//
// Don't forget to close the transport after you're done using the channel.
func DialChan(network string, addr string) (*Chan, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	ch, err := connToChan(conn)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func DialChanTimeout(network string, addr string, timeout time.Duration) (*Chan, error) {
	conn, err := net.DialTimeout(network, addr, timeout)
	if err != nil {
		return nil, err
	}
	ch, err := connToChan(conn)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func connToChan(conn net.Conn) (*Chan, error) {
	provider, err := spdy.NewSpdyStreamProvider(conn, false)
	if err != nil {
		return nil, err
	}
	transport := spdy.NewTransport(provider)

	sender, err := transport.NewSendChannel()
	if err != nil {
		return nil, err
	}

	receiver, remoteSender := libchan.Pipe()

	ch := &Chan{
		Sender:       sender,
		Receiver:     receiver,
		RemoteSender: remoteSender,
		Transport:    transport,
	}
	return ch, nil
}
