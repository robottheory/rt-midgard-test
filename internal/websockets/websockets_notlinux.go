// +build !linux

package websockets

import (
	"log"
	"runtime"
)

// NOTE: Websockes uses linux only syscalls so currently it is not supported for other os.
// For local dev on windows/macos you may need to run the application using Docker
func Serve(listenPort int, connectionLimit int) {
	log.Printf("WARNING: Websockets not implemented for os %s. Only linux is supported", runtime.GOOS)
}
