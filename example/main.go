package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/docker/libchan"
	"github.com/jaytaylor/libchanner"
)

type (
	RemoteRequest struct {
		Command    string
		StatusChan libchan.Sender
	}

	RemoteResponse struct {
		StatusCode int
		Message    string
		Data       []byte
	}
)

// receiverHandler conforms to the ReceiverHandler type.
func receiverHandler(receiver libchan.Receiver) {
	for {
		req := &RemoteRequest{}
		if err := receiver.Receive(req); err != nil {
			panic(fmt.Sprintf("unexpected error receiving remote struct: %s", err))
			break
		}

		response := requestHandler(req)

		if err := req.StatusChan.Send(response); err != nil {
			panic(fmt.Sprintf("unexpected error sending result: %s", err))
		}
	}
}

// requestHandler is the core remote request processor.
func requestHandler(rr *RemoteRequest) *RemoteResponse {
	response := &RemoteResponse{}

	switch rr.Command {
	case "CurrentTime":
		data, err := json.Marshal(time.Now())
		if err != nil {
			response.StatusCode = 1
			response.Message = err.Error()
			break
		}
		response.Data = data

	default:
		response.StatusCode = 1
		response.Message = "unknown command"
	}

	return response
}

func main() {
	addr := "127.0.0.1:8001"
	cs := libchanner.NewChanServer(addr, receiverHandler)
	cs.Quiet = true // Suppress internal ChanServer logging.
	if err := cs.Start(); err != nil {
		errExitf("error starting server: %s", err)
	}

	// Shutdown the server after we're done.
	defer func() {
		if err := cs.Stop(); err != nil {
			errExitf("error stopping server: %s", err)
		}
	}()

	// Connect to the libchan server.
	ch, err := libchanner.DialChan("tcp", addr)
	if err != nil {
		errExitf("error connecting to server: %s", err)
	}

	req := &RemoteRequest{
		Command:    "CurrentTime",
		StatusChan: ch.RemoteSender,
	}
	if err := ch.Sender.Send(req); err != nil {
		errExitf("error sending remote request: %s", err)
	}

	response := &RemoteResponse{}
	if err := ch.Receiver.Receive(response); err != nil {
		errExitf("error receiving remote response: %s", err)
	}

	if response.StatusCode != 0 {
		errExitf("non-zero status code in response: %v, message=%s", response.StatusCode, response.Message)
	}

	ts := &time.Time{}
	if err := json.Unmarshal(response.Data, ts); err != nil {
		errExitf("failed to decode time struct from json: %s", err)
	}

	fmt.Printf("Successfully decoded time struct! value=%s\n", *ts)
}

func errExitf(format string, args ...interface{}) {
	os.Stderr.WriteString(fmt.Sprintf(format+"\n", args...))
	os.Exit(1)
}
