// +build !linux

package websockets

import (
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/julienschmidt/httprouter"
)

// NOTE: Websockes uses linux only syscalls so currently it is not supported for other os.
// For local dev on windows/macos you may need to run the application using Docker
func Serve(connectionLimit int) {
	log.Printf("WARNING: Websockets not implemented for os %s. Only linux is supported", runtime.GOOS)
}

func WsHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprintf(w, "Websockets not implemented for os %s. Only linux is supported", runtime.GOOS)
	return
}
