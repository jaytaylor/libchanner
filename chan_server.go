package libchanner

import (
	"errors"
	"net"
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
	Quiet             bool // When true, logging will be suppressed.
	laddr             string
	receiverHandler   ReceiverHandler
	listener          *stoppableListener.StoppableListener
	transportListener *spdy.TransportListener
	lock              sync.Mutex
	stop              chan chan bool
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

	cs.info("starting laddr=%s", cs.laddr)

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

	transportListener, err := spdy.NewTransportListener(cs.listener, spdy.NoAuthenticator)
	if err != nil {
		return err
	}
	cs.transportListener = transportListener

	go func() {
		for {
			if cs.listener == nil || cs.transportListener == nil {
				cs.info("listener is nil, accept loop exiting")
				break
			}
			t, err := cs.transportListener.AcceptTransport()
			if err != nil {
				cs.info("accepting from transport failed: %s, accept loop exiting", err)
				break
			}

			go func() {
				defer t.Close()
				for {
					receiver, err := t.WaitReceiveChannel()
					if err != nil {
						cs.error("waiting for channel receiver failed: %s, connection loop exiting", err)
						break
					}
					go cs.receiverHandler(receiver)
				}
			}()
		}
	}()
	cs.info("started laddr=%s", cs.laddr)
	return nil
}

// Stop terminates the ChanServer.
func (cs *ChanServer) Stop() error {
	cs.lock.Lock()
	defer cs.lock.Unlock()

	cs.info("stopping laddr=%s", cs.laddr)

	if cs.listener == nil {
		return NotRunningError
	}

	if err := cs.transportListener.Close(); err != nil {
		return err
	}

	if err := cs.listener.StopSafely(); err != nil {
		return err
	}

	cs.transportListener = nil
	cs.listener = nil

	cs.info("stopped laddr=%s", cs.laddr)

	return nil
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
