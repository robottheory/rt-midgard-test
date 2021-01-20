package websockets

import (
	"log"

	"gitlab.com/thorchain/midgard/chain"
)

func Serve() {
	log.Println("Starting websocket process.")
	for {
		<-*chain.WebsocketNotify
		// If more notifications happened, eat all future ones.
		hadMore := true
		for hadMore {
			select {
			case <-*chain.WebsocketNotify:
			default:
				hadMore = false
			}
		}

		log.Println("Sending push notifications...")
	}
}
