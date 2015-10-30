package libchanner

import (
	"errors"
	"net"
	"strings"
	"sync"

	"github.com/docker/libchan"
	"github.com/docker/libchan/spdy"
	"github.com/jaytaylor/stoppableListener"
)

// ReceiverHandler is the type of the handler function to pass in when creating
// a new ChanServer.
type ReceiverHandler func(receiver libchan.Receiver)

// ChanServer provides a generic stoppable TCP server which is wired to
// any desired channel receiver handler.
type ChanServer struct {
	Quiet           bool // When true, logging will be suppressed.
	laddr           string
	receiverHandler ReceiverHandler
	listener        *stoppableListener.StoppableListener
	lock            sync.Mutex
	stop            chan chan bool
}

var (
	AlreadyRunningError = errors.New("already running")
	NotRunningError     = errors.New("not running")
)

// NewChanServer creates a new instance of TCP ChanServer.
//
// laddr is a string representing the interface and port to listen on.
// e.g. ":8001", "127.0.0.1:8001", etc.
//
// receiverHandler is a ReceiverHandler function to send inbound channel
// requests to.
func NewChanServer(laddr string, receiverHandler ReceiverHandler) *ChanServer {
	cs := &ChanServer{
		laddr:           laddr,
		receiverHandler: receiverHandler,
		stop:            make(chan chan bool, 1),
	}
	return cs
}

// Start launches the ChanServer.
func (cs *ChanServer) Start() error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.info("Starting laddr=%s", cs.laddr)

	if cs.listener != nil {
		return AlreadyRunningError
	}

	// Underlying remote channel listener.
	tcpLn, err := net.Listen("tcp", cs.laddr)
	if err != nil {
		return err
	}
	stoppable, err := stoppableListener.New(tcpLn)
	if err != nil {
		return err
	}
	cs.listener = stoppable
	if strings.HasSuffix(cs.laddr, ":0") {
		cs.laddr = cs.Addr()
	}

	go func() {
		for {
			if cs.listener == nil {
				cs.info("Nil listener detected (accept loop exiting)")
				break
			}

			conn, err := cs.listener.Accept()
			if err != nil {
				cs.info("Accepting from transport failed: %s (accept loop exiting)", err)
				break
			}

			provider, err := spdy.NewSpdyStreamProvider(conn, true)
			if err != nil {
				cs.info("Getting new SpdyStreamProvider failed: %s (accept loop exiting)", err)
				break
			}

			transport := spdy.NewTransport(provider)

			go func() {
				defer func() {
					if err := provider.Close(); err != nil {
						cs.error("Closing provider: %s", err)
					}
					if err := conn.Close(); err != nil {
						cs.error("Closing conn: %s", err)
					}
				}()
				for {
					receiver, err := transport.WaitReceiveChannel()
					if err != nil {
						cs.error("Waiting for channel receiver failed: %s, connection loop exiting", err)
						break
					}
					go cs.receiverHandler(receiver)
				}
			}()
		}
	}()
	cs.info("Started laddr=%s", cs.laddr)
	return nil
}

// Stop terminates the ChanServer.
func (cs *ChanServer) Stop() error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.info("Stopping laddr=%s", cs.laddr)

	if cs.listener == nil {
		return NotRunningError
	}

	if err := cs.listener.StopSafely(); err != nil {
		return err
	}

	cs.listener = nil

	cs.info("Stopped laddr=%s", cs.laddr)

	return nil
}

// Addr returns the listener address.
func (cs *ChanServer) Addr() string {
	if cs.listener == nil {
		return ""
	}
	addr := cs.listener.Addr().String()
	return addr
}

// info provides suppressable Info-level logging.
func (cs *ChanServer) info(format string, args ...interface{}) {
	if !cs.Quiet {
		log.Info(format, args...)
	}
}

// error provides suppressable Error-level logging.
func (cs *ChanServer) error(format string, args ...interface{}) {
	if !cs.Quiet {
		log.Error(format, args...)
	}
}
