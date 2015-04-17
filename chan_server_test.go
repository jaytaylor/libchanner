package libchanner

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/docker/libchan"
)

type (
	RemoteRequest struct {
		Command    string
		Args       []string
		Stdin      io.Reader
		Stdout     io.WriteCloser
		Stderr     io.WriteCloser
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

func Test_ChanServer(t *testing.T) {
	addr := "127.0.0.1:8001"
	cs := NewChanServer(addr, receiverHandler)
	if err := cs.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := cs.Stop(); err != nil {
			t.Error(err)
		}
	}()

	ch, err := DialChanTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}

	execRemote := func(req *RemoteRequest) (*RemoteResponse, error) {
		if err := ch.Sender.Send(req); err != nil {
			return nil, fmt.Errorf("error sending remote request: %s", err)
		}
		response := &RemoteResponse{}
		if err := ch.Receiver.Receive(response); err != nil {
			return nil, fmt.Errorf("error receiving remote response: %s", err)
		}
		if response.StatusCode != 0 {
			return nil, fmt.Errorf("non-zero status code in remote response, StatusCode=%v Message=%s", response.StatusCode, response.Message)
		}
		return response, nil
	}

	// Valid command should succeed.
	{
		req := &RemoteRequest{
			Command:    "CurrentTime",
			Args:       []string{},
			Stdin:      os.Stdin,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
			StatusChan: ch.RemoteSender,
		}
		resp, err := execRemote(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("resp=%s", resp)
	}

	// Unknown command should fail.
	{
		req := &RemoteRequest{
			Command:    "what!!? is this a valid command, sir?",
			Args:       []string{},
			Stdin:      os.Stdin,
			Stdout:     os.Stdout,
			Stderr:     os.Stderr,
			StatusChan: ch.RemoteSender,
		}
		resp, err := execRemote(req)
		if err == nil {
			t.Errorf("Expected invalid command to generate an error, but err=%v", nil)
		}
		t.Logf("resp=%s", resp)
	}
}
