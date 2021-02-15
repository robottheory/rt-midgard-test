// +build !linux

package websockets

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/julienschmidt/httprouter"
)

// NOTE: Websockes uses linux only syscalls so currently it is not supported for other os.
// For local dev on windows/macos you may need to run the application using Docker

type Quit struct {
	quitTriggered chan struct{}
	quitFinished  chan struct{}
}

func Start(connectionLimit int) *Quit {
	log.Printf("WARNING: Websockets not implemented for os %s. Only linux is supported", runtime.GOOS)
	return &Quit{}
}

func (q *Quit) Quit(timeout time.Duration) {
}

func WsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintf(w, "Websockets not implemented for os %s. Only linux is supported", runtime.GOOS)
	return
}
