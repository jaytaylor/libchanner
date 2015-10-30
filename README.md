# libchanner

This package makes it simple to plug into the power of the [libchan](https://github.com/docker/libchan) RPC library.  Elegant simplicity is achieved by providing a set of API functions which perform the repetitive server and client setup.

The ability to stop the server at any time is also included out of the box thanks to [stoppableListener](https://github.com/jaytaylor/stoppableListener).


## Get it

    go get -u https://github.com/jaytaylor/libchanner/...


## Important

This package only works with libchan v0.1.x series.


## Code layout

Server resources are in [chan_server.go](https://github.com/jaytaylor/libchanner/blob/master/chan_server.go)

Client resources are in [conn.go](https://github.com/jaytaylor/libchanner/blob/master/conn.go)


## Example usage

See [example/main.go](https://github.com/jaytaylor/libchanner/blob/master/example/main.go) for the fully working example.

run it:

```bash
go run example/main.go
```

output:

    Successfully decoded time struct! value=2015-04-17 11:34:01.046453821 -0700 PDT

Abreviated highlights:

(for demonstration purposes only, error checking omitted for brevity)

```go
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

func main() {
    // Server setup.
    cs := libchanner.NewChanServer("127.0.0.1:8001", receiverHandler)
    cs.Start()

    // Shutdown the server after we're done.
    defer func() {
        cs.Stop()
    }()

    // Client setup.
    ch, _ := libchanner.DialChan("tcp", "127.0.0.1:8001")

    // Send request.
    req := &RemoteRequest{
        Command:    "CurrentTime",
        StatusChan: ch.RemoteSender,
    }
    ch.Sender.Send(req)

    // Get response.
    response := &RemoteResponse{}
    ch.Receiver.Receive(response)

    // Now do anything you want with the response!
}

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
```

also see the [unit-tests](https://github.com/jaytaylor/libchanner/blob/master/chan_server_test.go) for additional example usage.

## Unit-tests

Running the [unit-tests](https://github.com/jaytaylor/libchanner/blob/master/chan_server_test.go) is straightforward and standard:

```bash
go test
```


# License

Permissive [MIT license](https://github.com/jaytaylor/libchanner/blob/master/LICENSE).


## Contact

You are more than welcome to open issues and send pull requests if you find a bug or want a new feature.

If you appreciate this library please feel free to drop me a line and let me know!  It's always nice to hear from people who have benefitted from my work.

Email: jay at (my github username).com

Twitter: [@jtaylor](https://twitter.com/jtaylor)

## Related work

* _[netchan](https://godoc.org/golang.org/x/exp/old/netchan) (deprecated)_

* [fatchan](https://github.com/kylelemons/fatchan)
