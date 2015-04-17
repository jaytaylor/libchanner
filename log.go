package libchanner

import (
	"os"

	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("libchanner")

func init() {
	var format = logging.MustStringFormatter(
		"%{color}%{time:2006-01-02 15:04:05.000} %{id:03x} %{shortpkg} %{shortfile} %{shortfunc} ▶ %{level:.4s} %{color:reset} %{message}",
	)

	backend := logging.NewLogBackend(os.Stdout, "", 0)

	// For messages written to backend we add some additional information to the
	// output, including the used log level and the name of the function.
	backendFormatter := logging.NewBackendFormatter(backend, format)

	// Set the backend(s) to be used.
	logging.SetBackend(backendFormatter)
}
